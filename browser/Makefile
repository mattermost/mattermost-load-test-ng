.PHONY: clean install build start dev dev-watch schema help

# Environment variables
export PLAYWRIGHT_BROWSERS_PATH ?= 0
BIN := ./node_modules/.bin

# Sets Node.js version from .nvmrc via nvm if it is installed, otherwise falls back to system Node
nvm_install_node = if [ -s "$$NVM_DIR/nvm.sh" ]; then . "$$NVM_DIR/nvm.sh" && nvm install; fi

install-dependencies:
	@echo "Installing node_modules of LTBrowser app"
	@$(nvm_install_node) && npm install
	@echo "Successfully installed node_modules of LTBrowser app"

install-playwright: install-dependencies
	@echo "Installing Playwright Chromium browser of LTBrowser app"
	$(BIN)/playwright install --with-deps chromium
	@echo "Successfully installed Playwright Chromium browser of LTBrowser app"

## Install npm dependencies and Playwright Chromium browser with its dependencies
install: install-dependencies install-playwright

## Clean build artifacts, node_modules, and test results
clean:
	@echo "Cleaning build artifacts, node_modules, and test results of LTBrowser app"
	rm -rf dist
	rm -rf node_modules
	rm -rf e2e_results
	rm -rf .cache
	rm -rf screenshots

build-server:
	@echo "Building LTBrowser app"
	@$(nvm_install_node) && $(BIN)/tsc
	@echo "LTBrowser app built successfully"

## Builds the LTBrowser server, by first installing the dependencies and then building the server. This does not install the Playwright browser which should be done separately
build: clean install-dependencies build-server

## Used by service to start the LTBrowser server in linux environments
start-service: install
	@echo "Starting LTBrowser app service"
	node dist/server.js

## Start the LTBrowser server by first doing a cleanup and then performing a production mode build
start: build install-playwright
	@echo "Starting LTBrowser app"
	node dist/server.js

## Start the LTBrowser server in development mode
dev: install
	@echo "Starting LTBrowser app in development mode"
	$(BIN)/tsx src/server.ts

## Start the LTBrowser server in development watch mode
dev-watch: install
	@echo "Starting LTBrowser app in development watch mode"
	$(BIN)/tsx watch src/server.ts

## Run the smoke simulations
smoke-simulation: install
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

## Generate the OpenAPI schemas for the API in JSON and YAML format
schema:
	@echo "Generating OpenAPI schema in JSON format"
	NODE_OPTIONS='--import=tsx/esm' $(BIN)/fastify generate-swagger src/app.ts > ./openapi.json
	@echo "Generating OpenAPI schema in YAML format"
	NODE_OPTIONS='--import=tsx/esm' $(BIN)/fastify generate-swagger --yaml=true src/app.ts > ./openapi.yaml

## Shows the list of available commands
help:
	@awk '/^##/{c=substr($$0,3); getline; if($$1 ~ /^[a-zA-Z_-]+:/) printf "\033[36m%-15s\033[0m %s\n", $$1, c}' ./Makefile | sort
	@echo
