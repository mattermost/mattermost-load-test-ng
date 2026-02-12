.PHONY: clean install-dependencies build-bundle build-types build-package build all

all: build

## Clean build artifacts, node_modules, and test results
clean:
	@echo "Cleaning build artifacts, node_modules, and test results of @mattermost/loadtest-browser-lib"
	rm -rf dist
	rm -rf node_modules
	rm -rf *.tsbuildinfo

## Install npm dependencies of @mattermost/loadtest-browser-lib
install-dependencies:
	@echo "Installing node_modules of @mattermost/loadtest-browser-lib"
	npm install
	@echo "Successfully installed node_modules of @mattermost/loadtest-browser-lib"

## Build the bundle of @mattermost/loadtest-browser-lib with types
build-bundle:
	@echo "Building bundle of @mattermost/loadtest-browser-lib"
	node scripts/generate_bundle.mjs
	@echo "Successfully built bundle of @mattermost/loadtest-browser-lib"

## Build the type declarations of @mattermost/loadtest-browser-lib
build-types:
	@echo "Building types of @mattermost/loadtest-browser-lib"
	node scripts/generate_types.mjs
	@echo "Successfully built types of @mattermost/loadtest-browser-lib"

## Build the package of @mattermost/loadtest-browser-lib and put it in the browser/packs directory
build-package:
	@echo "Building package of @mattermost/loadtest-browser-lib"
	node scripts/generate_pack.mjs
	@echo "Successfully built package of @mattermost/loadtest-browser-lib"

build: clean install-dependencies build-bundle build-types build-package
