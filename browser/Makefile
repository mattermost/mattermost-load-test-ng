.PHONY: clean install build start dev dev-watch schema help

# Environment variables
export PLAYWRIGHT_HEADLESS_ONLY ?= true
BIN := ./node_modules/.bin

install-dependencies:
	@echo "Installing node_modules of LTBrowser"
	npm install
	@echo "Successfully installed node_modules of LTBrowser"

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

## Clean build artifacts, node_modules, and test results
clean:
	@echo "Cleaning build artifacts, node_modules, and test results"
	rm -rf build
	rm -rf node_modules
	rm -rf e2etest_results
	rm -rf .cache
	@echo "Cleaning complete"

## Builds the LTBrowser server, by first installing the dependencies and then building the server. This does not install the Playwright browser which should be done separately
build: clean install-dependencies
	@echo "Building LTBrowser server"
	$(BIN)/tsc
	@echo "LTBrowser server built successfully"

start-service: install
	node server.js

## Start the LTBrowser server by first doing a cleanup and then performing a production mode build
start: build install-playwright
	@echo "Starting LTBrowser API server"
	node dist/server.js

## Start the LTBrowser server in development mode
dev: install
	@echo "Starting LTBrowser API server in development mode"
	$(BIN)/tsx src/server.ts

## Start the LTBrowser server in development watch mode
dev-watch: install
	@echo "Starting LTBrowser API server in development watch mode"
	$(BIN)/tsx watch src/server.ts

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
