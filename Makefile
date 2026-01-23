.PHONY: run build clean prep container-dev container-prod

PORT ?= 3000
PREP_DIR := tmp/assets
IMAGE_DEV := webapp-dev
IMAGE_PRD := webapp-prod
POLICY_FILE := config/policy.json

# Runs the app locally using Air (requires Nix shell)
run:
	@echo "# Running locally on http://localhost:$(PORT)"
	PORT=$(PORT) air

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
	@cp -f "$(HTMX_SRC)" $(PREP_DIR)/htmx.min.js
	@cp -f "$(REDHAT_GPG)" $(PREP_DIR)/redhat-release.gpg

# Starts a Live-Reload Dev Container
container-dev: prep
	@echo "# Building Dev Image..."
	podman build --signature-policy $(POLICY_FILE) --target dev -t $(IMAGE_DEV) .
	
	@echo "# Starting Dev Container..."
	@# We add --userns=keep-id so you own the files in the mounted volume
	@echo "# Mapping: Host($(PORT)) -> Container(8080)"
	@podman run --rm -it \
		-p $(PORT):8080 \
		-v "$(PWD):/app" \
		--userns=keep-id \
		--user $$(id -u):0 \
		--name webapp-dev \
		$(IMAGE_DEV)

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

clean:
	@echo "# Cleaning up..."
	rm -rf bin tmp vendor
