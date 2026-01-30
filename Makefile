# --- Configuration ---
PORT         ?= 3000
PREP_DIR     := tmp/assets
IMAGE_DEV    := webapp-dev
IMAGE_PRD    := webapp-prod
POLICY_FILE  := config/policy.json
PODMAN_NET   := webapp-net

# Database Names & Credentials
PG_NAME      := webapp-postgres
ATLAS_DEV_DB := webapp-atlas-dev
PG_USER      := user
PG_PASS      := pass
PG_DB        := webapp
SQLITE_VOL   := webapp-sqlite-data
SQLITE_PATH  := /app/tmp/

# Connection Strings
# App DB (Standard)
DATABASE_URL  := postgres://$(PG_USER):$(PG_PASS)@localhost:5432/$(PG_DB)?sslmode=disable
# Atlas Shadow DB
ATLAS_DEV_URL := postgres://postgres:$(PG_PASS)@localhost:5433/postgres?sslmode=disable

DOCKER_HOST_URL := unix://$(shell podman info --format '{{.Host.RemoteSocket.Path}}')
ATLAS           := DOCKER_HOST=$(DOCKER_HOST_URL) atlas
SCHEMA_FILE     := file://schema.sql

.PHONY: run build clean prep container-dev container-prod \
        databases-up stop-postgres clean-postgres \
        db-apply db-status db-inspect network

databases-up: network
	@if ! podman ps --format '{{.Names}}' | grep -q "^$(PG_NAME)$$"; then \
		$(MAKE) postgres-start; \
	fi
	@if ! podman ps --format '{{.Names}}' | grep -q "^$(ATLAS_DEV_DB)$$"; then \
		$(MAKE) atlas-dev-start; \
	fi
	@echo "Creating $(SQLITE_VOL) volume..."
	podman volume create $(SQLITE_VOL) || true
	@echo "Initializing SQLITE DB file..."
	podman run --rm \
		-v $(SQLITE_VOL):$(SQLITE_PATH):Z \
		registry.access.redhat.com/ubi9/ubi-minimal:latest \
		sh -c "touch $(SQLITE_PATH)/app.db && chown -R 1001:0 $(SQLITE_PATH) && chmod -R 775 $(SQLITE_PATH)"

# Stop containers without deleting data
stop-postgres:
	@echo "# Stopping database containers..."
	@podman stop $(PG_NAME) $(ATLAS_DEV_DB) || true

# Nuke everything to start fresh
clean-postgres:
	@echo "# Removing containers and volumes..."
	@podman rm -f $(PG_NAME) $(ATLAS_DEV_DB) || true
	@podman volume rm -f webapp-db-data || true
	@podman volume rm -f $(SQLITE_VOL) || true


# --- Atlas Migration Targets ---
db-apply:
	@echo "# Syncing database schema..."
	@$(ATLAS) schema apply \
		-u $(DATABASE_URL) \
		--to $(SCHEMA_FILE) \
		--dev-url "$(ATLAS_DEV_URL)"

db-status:
	@$(ATLAS) schema diff \
		--from $(DATABASE_URL) \
		--to $(SCHEMA_FILE) \
		--dev-url "$(ATLAS_DEV_URL)"

db-inspect:
	@$(ATLAS) schema inspect -u $(DATABASE_URL) --format '{{ sql . }}' > schema.sql


# --- Base Infrastructure ---
network:
	@podman network exists $(PODMAN_NET) || podman network create $(PODMAN_NET)

postgres-start: network
	@echo "# Starting Persistent App Postgres..."
	@podman run -d \
		--name $(PG_NAME) \
		--net $(PODMAN_NET) \
		-p 5432:5432 \
		-e POSTGRES_USER=$(PG_USER) \
		-e POSTGRES_PASSWORD=$(PG_PASS) \
		-e POSTGRES_DB=$(PG_DB) \
		-v webapp-db-data:/var/lib/postgresql/data \
		public.ecr.aws/docker/library/postgres:alpine
	@until podman exec $(PG_NAME) pg_isready -U $(PG_USER); do sleep 1; done

atlas-dev-start: network
	@echo "# Starting Ephemeral Shadow DB..."
	@podman run -d --rm \
		--name $(ATLAS_DEV_DB) \
		--net $(PODMAN_NET) \
		-p 5433:5432 \
		-e POSTGRES_PASSWORD=$(PG_PASS) \
		public.ecr.aws/docker/library/postgres:alpine
	@until podman exec $(ATLAS_DEV_DB) pg_isready -U postgres; do sleep 1; done

# --- App Logic ---

run: databases-up db-apply
	@mkdir -p $(PREP_DIR)
	@echo "# Running locally on http://localhost:$(PORT)"
	DATABASE_URL=$(DATABASE_URL) PORT=$(PORT) air

build:
	@echo "# Building binary to ./bin/web"
	mkdir -p bin
	go build -o bin/web cmd/web/main.go

clean:
	@echo "# Cleaning up build artifacts..."
	rm -rf bin vendor
	rm -rf tmp/build/*
	rm -rf $(PREP_DIR)
	@podman volume rm -f webapp-sqlite-data || true



# --- Container Targets ---

# Prepares all assets so the build is self-contained
prep:
	go mod vendor
	@mkdir -p $(PREP_DIR)
	@cp -f "$(REDHAT_GPG)" $(PREP_DIR)/redhat-release.gpg
	# We copy schema.sql just in case you want it inside the image for reference
	@cp schema.sql $(PREP_DIR)/schema.sql

# 1. Dev Container (Live Reload with Air)
# Connects to 'webapp-postgres' by default using the internal Podman network
container-dev: prep databases-up
	@echo "# Building Dev Image..."
	podman build --signature-policy $(POLICY_FILE) --target dev -t $(IMAGE_DEV) .

	@echo "# Starting Dev Container..."
	@echo "# To override DB: make container-dev DATABASE_URL='postgres://...'"
	@podman run --rm -it \
		-p $(PORT):8080 \
		-v "$(PWD):/app" \
		--net $(PODMAN_NET) \
		--userns=keep-id \
		--user $$(id -u):0 \
		-e DATABASE_URL="postgres://$(PG_USER):$(PG_PASS)@$(PG_NAME):5432/$(PG_DB)?sslmode=disable" \
		-e GOCACHE=/app/tmp/build/.cache \
		-e GOPATH=/app/tmp/build/go \
		--name webapp-dev \
		$(IMAGE_DEV)

# 2. Prod Container (Static Binary)
# Default behavior: Runs locally connected to your local DB for testing.
# OpenShift behavior: You (or the DeploymentConfig) override DATABASE_URL env var.
container-prod: prep databases-up
	@echo "# Building Prod Image..."
	podman build --signature-policy $(POLICY_FILE) --target prod -t $(IMAGE_PRD) .

	@echo "# Running Prod Container (Local Test Mode)..."
	podman run  \
		--name webapp-prod \
		--net $(PODMAN_NET) \
		-p 8080:8080 \
		-v webapp-sqlite-data:/app/tmp/ \
		-v $(SQLITE_VOL):$(SQLITE_PATH):Z \
		--rm \
		-e DATABASE_URL="postgres://$(PG_USER):$(PG_PASS)@$(PG_NAME):5432/$(PG_DB)?sslmode=disable" \
		$(IMAGE_PRD)
