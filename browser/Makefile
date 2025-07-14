.PHONY: install build start dev clean test lint help

# Environment variables
export PLAYWRIGHT_HEADLESS_ONLY ?= true
BIN := ./node_modules/.bin

## Clean build artifacts, node_modules, and test results
clean:
	@echo "Cleaning build artifacts, node_modules, and test results"
	rm -rf build
	rm -rf node_modules
	rm -rf e2etest_results
	rm -rf .cache
	@echo "Cleaning complete"

## Install npm dependencies
install-dependencies:
	@echo "Installing dependencies"
	npm install
	@echo "Successfully installed dependencies"

## Install Playwright Chromium browser with its dependencies
install-playwright: export PLAYWRIGHT_BROWSERS_PATH=0
install-playwright: install-dependencies
	@if [ "$(PLAYWRIGHT_HEADLESS_ONLY)" = "true" ]; then \
		echo "Installing Playwright Headless Chromium browser"; \
		$(BIN)/playwright install --with-deps --only-shell chromium; \
	else \
		echo "Installing Playwright Chromium browser"; \
		$(BIN)/playwright install --with-deps chromium; \
	fi
	@echo "Successfully installed browser binaries in node_modules/playwright-core/.local-browsers"

## Install npm dependencies and Playwright Chromium browser with its dependencies
install: install-dependencies install-playwright

## Reinstall npm dependencies and Playwright Chromium browser with its dependencies
clean-install: clean install-dependencies install-playwright

## Build the Browser API server
build: clean-install install-dependencies install-playwright
	@echo "Building server"
	$(BIN)/tsc
	@echo "Browser API server built successfully"

## Start the browser API server in production mode
## Note: This installs the browser in headless mode only
start: install
	@echo "Starting LTBrowser API server"
	node build/server.js

## Start the browser API server in development mode
dev: install
	@echo "Starting LTBrowser API server in development mode"
	$(BIN)/tsx src/server.ts

## Start the browser API server in development watch mode
dev-watch: install
	@echo "Starting LTBrowser API server in development watch mode"
	$(BIN)/tsx watch src/server.ts

## Generate the OpenAPI schemas for the API in JSON and YAML format
schema:
	@echo "Generating OpenAPI schema in JSON format"
	NODE_OPTIONS='--import=tsx/esm' $(BIN)/fastify generate-swagger src/app.ts > ./openapi.json
	@echo "Generating OpenAPI schema in YAML format"
	NODE_OPTIONS='--import=tsx/esm' $(BIN)/fastify generate-swagger --yaml=true src/app.ts > ./openapi.yaml

## Show this help message
help:
	@awk '/^##/{c=substr($$0,3); getline; if($$1 ~ /^[a-zA-Z_-]+:/) printf "\033[36m%-15s\033[0m %s\n", $$1, c}' ./Makefile | sort
	@echo
