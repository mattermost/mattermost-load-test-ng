.PHONY: install build start dev clean test lint help

# Environment variables
export PLAYWRIGHT_BROWSERS_PATH ?= 0
export PATH := ./node_modules/.bin:$(PATH)

# Default target
all: build ## Default: build the browser server.

install-dependencies:
	@echo "Installing dependencies"
	npm install
	@echo "Successfully installed dependencies"

install-playwright: export PLAYWRIGHT_BROWSERS_PATH=0
install-playwright: install-dependencies
	@echo "Installing Playwright Chromium browser"
	./node_modules/.bin/playwright install --with-deps chromium
	@echo "Successfully installed Playwright Chromium browser binaries in node_modules/playwright-core/.local-browsers"

install: install-dependencies install-playwright

build: install
	@echo "Building server"
	./node_modules/.bin/tsc
	@echo "Server built successfully"

start: install build
	@echo "Starting browser api server"
	node build/server.js

start-dev: install
	@echo "Starting browser api server in development mode"
	./node_modules/.bin/tsx src/server.ts

start-watch: install
	@echo "Starting browser api server in development watch mode"
	./node_modules/.bin/tsx watch src/server.ts

clean:
	rm -rf build
	rm -rf node_modules
	rm -rf test-results
	rm -rf .cache

help:
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' ./Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo
