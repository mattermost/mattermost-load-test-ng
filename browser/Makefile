.PHONY: all help clean nvm-install-node install-dependencies install-playwright install build-server build start-service start dev dev-watch smoke-simulation schema

# Environment variables
export PLAYWRIGHT_BROWSERS_PATH ?= 0
BIN := ./node_modules/.bin

all: build

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "%-24s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

clean: ## Clean build artifacts, node_modules, and other cache files
	@echo "Cleaning build artifacts, node_modules, and test results of LTBrowser app"
	rm -rf dist
	rm -rf node_modules
	rm -rf e2e_results
	rm -rf .cache
	rm -rf screenshots

# Sets Node.js version from .nvmrc via nvm if it is installed, otherwise falls back to system Node
nvm-install-node:
	@if [ -s "$$NVM_DIR/nvm.sh" ]; then . "$$NVM_DIR/nvm.sh" && nvm install; fi

install-dependencies:
	@echo "Installing node_modules of LTBrowser app"
	@npm install
	@echo "Successfully installed node_modules of LTBrowser app"

install-playwright: install-dependencies
	@echo "Installing Playwright Chromium browser of LTBrowser app"
	$(BIN)/playwright install --with-deps chromium
	@echo "Successfully installed Playwright Chromium browser of LTBrowser app"

install: install-dependencies install-playwright ## Installs the dependencies and Playwright Chromium browser with its dependencies

build-server:
	@echo "Building LTBrowser app"
	@$(BIN)/tsc
	@echo "LTBrowser app built successfully"

build: clean install-dependencies build-server ## Builds the LTBrowser server by first installing the dependencies and then building the server

start-service: install ## Used by service to start the LTBrowser server in linux environments
	@echo "Starting LTBrowser app service"
	node dist/server.js

start: build install-playwright ## Start the LTBrowser server by first doing a cleanup and then performing a production mode build
	@echo "Starting LTBrowser app"
	node dist/server.js

dev: nvm-install-node install ## Start the LTBrowser server in development mode
	@echo "Starting LTBrowser app in development mode"
	$(BIN)/tsx src/server.ts

dev-watch: nvm-install-node install ## Start the LTBrowser server in development watch mode
	@echo "Starting LTBrowser app in development watch mode"
	$(BIN)/tsx watch src/server.ts

smoke-simulation: nvm-install-node install ## Run the smoke simulations
	@if [ -f "src/smoke_simulations/smoke_simulation.json" ]; then \
		echo "Running smoke simulation of LTBrowser app"; \
	else \
		echo "Error: smoke_simulation.json file not found, copy the smoke_simulation.sample.json file in the same directory to smoke_simulation.json and edit it"; \
		exit 1; \
	fi

	@if [ -f "src/smoke_simulations/index.ts" ]; then \
		$(BIN)/tsx src/smoke_simulations/index.ts; \
	elif [ -f "src/smoke_simulations/index.js" ]; then \
		node src/smoke_simulations/index.js; \
	else \
		echo "Error: file not found (checked .ts and .js extensions)"; \
		exit 1; \
	fi

schema: ## Generate the OpenAPI schemas for the API in JSON and YAML format
	@echo "Generating OpenAPI schema in JSON format"
	NODE_OPTIONS='--import=tsx/esm' $(BIN)/fastify generate-swagger src/app.ts > ./openapi.json
	@echo "Generating OpenAPI schema in YAML format"
	NODE_OPTIONS='--import=tsx/esm' $(BIN)/fastify generate-swagger --yaml=true src/app.ts > ./openapi.yaml
