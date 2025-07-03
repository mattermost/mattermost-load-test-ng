.PHONY: install build start dev clean test lint help

# Environment variables
export PLAYWRIGHT_BROWSERS_PATH ?= 0
export PATH := ./node_modules/.bin:$(PATH)

## Install npm dependencies
install-dependencies:
	@echo "Installing dependencies"
	npm install
	@echo "Successfully installed dependencies"

## Install Playwright Chromium browser with its dependencies
install-playwright: export PLAYWRIGHT_BROWSERS_PATH=0
install-playwright: install-dependencies
	@echo "Installing Playwright Chromium browser"
	./node_modules/.bin/playwright install --with-deps chromium
	@echo "Successfully installed Playwright Chromium browser binaries in node_modules/playwright-core/.local-browsers"

## Install npm dependencies and Playwright Chromium browser with its dependencies
install: install-dependencies install-playwright

## Build the TypeScript server
build: install
	@echo "Building server"
	./node_modules/.bin/tsc
	@echo "Server built successfully"

## Start the browser API server in production mode
start: install build
	@echo "Starting browser api server"
	node build/server.js

## Start the browser API server in development mode
start-dev: install
	@echo "Starting browser api server in development mode"
	./node_modules/.bin/tsx src/server.ts

## Start the browser API server in development watch mode
start-watch: install
	@echo "Starting browser api server in development watch mode"
	./node_modules/.bin/tsx watch src/server.ts

## Clean build artifacts, node_modules, and test results
clean:
	rm -rf build
	rm -rf node_modules
	rm -rf test-results
	rm -rf .cache

## Show this help message
help:
	@awk '/^##/{c=substr($$0,3); getline; if($$1 ~ /^[a-zA-Z_-]+:/) printf "\033[36m%-15s\033[0m %s\n", $$1, c}' ./Makefile | sort
	@echo
