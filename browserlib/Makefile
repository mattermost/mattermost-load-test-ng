.PHONY: all help clean install-dependencies build-bundle build-types build-pack build-dev build

all: build

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "%-24s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

clean: ## Clean build artifacts, node_modules, and other cache files
	@echo "Cleaning build artifacts, node_modules, and other cache files of @mattermost/loadtest-browser-lib"
	rm -rf dist
	rm -rf node_modules
	rm -rf *.tsbuildinfo

# Install npm dependencies of @mattermost/loadtest-browser-lib
install-dependencies:
	@echo "Installing node_modules of @mattermost/loadtest-browser-lib"
	npm install
	@echo "Successfully installed node_modules of @mattermost/loadtest-browser-lib"

# Build the bundle of @mattermost/loadtest-browser-lib
build-bundle:
	@echo "Building bundle of @mattermost/loadtest-browser-lib"
	node scripts/generate_bundle.mjs
	@echo "Successfully built bundle of @mattermost/loadtest-browser-lib"

# Build the type declarations of @mattermost/loadtest-browser-lib
build-types:
	@echo "Building types of @mattermost/loadtest-browser-lib"
	node scripts/generate_types.mjs
	@echo "Successfully built types of @mattermost/loadtest-browser-lib"

# Build the npm pack of @mattermost/loadtest-browser-lib and put it in the browser/packs directory for local development
build-pack:
	@echo "Building npm pack of @mattermost/loadtest-browser-lib"
	node scripts/generate_pack.mjs
	@echo "Successfully built npm pack of @mattermost/loadtest-browser-lib"

build-dev: clean install-dependencies build-bundle build-types build-pack ## Builds the bundle by creating npm pack and placing it in browser/packs for local development

build: clean install-dependencies build-bundle build-types ## Builds the bundle of @mattermost/loadtest-browser-lib
	@echo "Successfully built @mattermost/loadtest-browser-lib"
