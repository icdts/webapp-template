.PHONY: run build clean

# Defaults
PORT ?= 3000

# "make run" now starts the Live Reloader
run:
	@echo "# Starting Air (Live Reload) on http://localhost:$(PORT)..."
	PORT=$(PORT) air

build:
	@echo "# Building binary to ./bin/web"
	mkdir -p bin
	go build -o bin/web cmd/web/main.go

clean:
	@echo "# Cleaning bin and tmp directories"
	rm -rf bin tmp
