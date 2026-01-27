.PHONY: run build clean prep container-dev container-prod

PORT ?= 3000
PREP_DIR := tmp/assets
IMAGE_DEV := webapp-dev
IMAGE_PRD := webapp-prod
POLICY_FILE := config/policy.json
PODMAN_NET := webapp-net

PG_NAME := webapp-postgres
PG_USER := user
PG_PASS := pass
PG_DB   := webapp
DATABASE_URL := postgres://$(PG_USER):$(PG_PASS)@$(PG_NAME):5432/$(PG_DB)?sslmode=disable

# Runs the app locally using Air (requires Nix shell)
run: postgres-start
	@mkdir -p $(PREP_DIR)
	@echo "# Running locally on http://localhost:$(PORT)"
	DATABASE_URL="postgres://$(PG_USER):$(PG_PASS)@localhost:5432/$(PG_DB)?sslmode=disable" \
	PORT=$(PORT) air
	@make postgres-stop

# Builds the binary locally
build:
	@echo "# Building binary to ./bin/web"
	mkdir -p bin
	go build -o bin/web cmd/web/main.go

# Prepares all assets (Deps & HTMX) so Docker doesn't need internet
prep:
	go mod vendor
	@mkdir -p vendor
	
	@mkdir -p $(PREP_DIR)
	@cp -f "$(REDHAT_GPG)" $(PREP_DIR)/redhat-release.gpg

# Starts a Live-Reload Dev Container
container-dev: prep postgres-start
	@echo "# Building Dev Image..."
	podman build --signature-policy $(POLICY_FILE) --target dev -t $(IMAGE_DEV) .
	
	@echo "# Starting Dev Container..."
	@echo "# Mapping: Host($(PORT)) -> Container(8080)"
	@echo "# Postgres connection = $(DATABASE_URL)"
	@podman run --rm -it \
		-p $(PORT):8080 \
		-v "$(PWD):/app" \
		--net $(PODMAN_NET) \
		--userns=keep-id \
		--user $$(id -u):0 \
		-e DATABASE_URL=$(DATABASE_URL) \
		-e GOCACHE=/app/tmp/build/.cache \
		-e GOPATH=/app/tmp/build/go \
		--name webapp-dev \
		$(IMAGE_DEV)
	@make postgres-stop

# This is a static, immutable image (no live reload)
container-prod: prep
	@echo "# Building Prod Image..."
	podman build --signature-policy $(POLICY_FILE) --target prod -t $(IMAGE_PRD) .
	
	@echo "# Running Prod Container..."
	@podman run -d \
		-p 8080:8080 \
		--rm \
		--name webapp-prod \
		$(IMAGE_PRD)
	@echo "# App running at http://localhost:8080"

network:
	@echo "# Checking for network $(PODMAN_NET)"
	@podman network exists $(PODMAN_NET) || podman network create $(PODMAN_NET)

postgres-start: network
	@echo "# Starting Postgres (Official Alpine Image)..."
	@# No build step needed anymore. Podman pulls automatically if missing.
	@podman run -d --rm \
		--signature-policy $(POLICY_FILE) \
		--name $(PG_NAME) \
		--net $(PODMAN_NET) \
		-p 5432:5432 \
		-e POSTGRES_USER=$(PG_USER) \
		-e POSTGRES_PASSWORD=$(PG_PASS) \
		-e POSTGRES_DB=$(PG_DB) \
		public.ecr.aws/docker/library/postgres:alpine

	@echo "# Waiting for Postgres..."
	@until podman exec $(PG_NAME) pg_isready -U $(PG_USER); do sleep 1; done
	@echo "# Postgres is ready!"

postgres-stop:
	@podman stop $(PG_NAME) || true

clean:
	@echo "# Cleaning up..."
	rm -rf bin tmp vendor
