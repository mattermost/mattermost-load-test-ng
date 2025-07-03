.PHONY: install build start dev clean test lint help

# Environment variables
export PLAYWRIGHT_HEADLESS_ONLY ?= true
export PATH := ./node_modules/.bin:$(PATH)

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
		./node_modules/.bin/playwright install --with-deps --only-shell chromium; \
	else \
		echo "Installing Playwright Chromium browser"; \
		./node_modules/.bin/playwright install --with-deps chromium; \
	fi
	@echo "Successfully installed browser binaries in node_modules/playwright-core/.local-browsers"

## Install npm dependencies and Playwright Chromium browser with its dependencies
install: install-dependencies install-playwright

## Build the Browser API server
build: clean install-dependencies install-playwright
	@echo "Building server"
	./node_modules/.bin/tsc
	@echo "Browser API server built successfully"

## Start the browser API server in production mode
## Note: This installs the browser in headless mode only
start: build
	@echo "Starting Browser API server"
	node build/server.js

## Start the browser API server in development mode
dev: install
	@echo "Starting Browser API server in development mode"
	./node_modules/.bin/tsx src/server.ts

## Start the browser API server in development watch mode
dev-watch: install
	@echo "Starting browser api server in development watch mode"
	./node_modules/.bin/tsx watch src/server.ts

## Show this help message
help:
	@awk '/^##/{c=substr($$0,3); getline; if($$1 ~ /^[a-zA-Z_-]+:/) printf "\033[36m%-15s\033[0m %s\n", $$1, c}' ./Makefile | sort
	@echo
