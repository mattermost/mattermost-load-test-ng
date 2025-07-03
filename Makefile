.PHONY: install clean

GO=go

DIST_ROOT=dist
# We specify version for the build; it is the branch-name by default, also we try
# to find if there is a tag pointed to the current commit. If so, we use the tag.
DIST_VER=$(shell git rev-parse --abbrev-ref HEAD)
ifeq ($(shell git describe --tags $(git rev-parse @) 2>&1 >/dev/null; echo $$?), 0)
	DIST_VER=$(shell git describe --tags $(git rev-parse @))
endif
DIST_PATH=$(DIST_ROOT)/$(DIST_VER)
STATUS=$(shell git diff-index --quiet HEAD --; echo $$?)

GOBIN=$(PWD)/bin
# We need to export GOBIN to allow it to be set
# for processes spawned from the Makefile
export GOBIN ?= $(PWD)/bin

PATH=$(shell printenv PATH):$(GOBIN)

AGENT=$(GOBIN)/ltagent
AGENT_ARGS=-mod=readonly -trimpath ./cmd/ltagent
API_SERVER=$(GOBIN)/ltapi
API_SERVER_ARGS=-mod=readonly -trimpath ./cmd/ltapi

# GOOS/GOARCH of the build host, used to determine whether we're cross-compiling or not
BUILDER_GOOS_GOARCH="$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)"

all: install ## Default: alias for 'install'.

build-linux: ## Build the binary (only for Linux on AMD64).
	@echo Build Linux amd64
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(AGENT) $(AGENT_ARGS)
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(API_SERVER) $(API_SERVER_ARGS)

build-osx: ## Build the binary (only for OSX on AMD64).
	@echo Build OSX amd64
	env GOOS=darwin GOARCH=amd64 $(GO) build -o $(AGENT) $(AGENT_ARGS)
	env GOOS=darwin GOARCH=amd64 $(GO) build -o $(API_SERVER) $(API_SERVER_ARGS)

assets: ## Generate the assets. Install go-bindata if needed.
	go install github.com/kevinburke/go-bindata/go-bindata@v3.23.0
	go generate ./...
	go fmt ./...

build-browser-api: ## Build the browser testing HTTP server.
	cd browser && npm run server:build

build: assets build-linux build-osx build-browser-api ## Generate the assets and build the binary for all platforms.

install: ## Build and install for the current platform.
	$(GO) install $(API_SERVER_ARGS)

package: ## Build and package (only available for Linux on AMD64).
ifneq ($(STATUS), 0)
	@echo Warning: Repository has uncommitted changes.
endif
	@$(MAKE) build-linux
	@$(MAKE) build-browser-api
	rm -rf $(DIST_ROOT)
	$(eval PLATFORM=linux-amd64)
	$(eval PLATFORM_DIST_PATH=$(DIST_PATH)/$(PLATFORM))
	mkdir -p $(PLATFORM_DIST_PATH)
	mkdir -p $(PLATFORM_DIST_PATH)/config
	mkdir -p $(PLATFORM_DIST_PATH)/bin

	cp config/config.sample.json $(PLATFORM_DIST_PATH)/config/config.json
	cp config/coordinator.sample.json $(PLATFORM_DIST_PATH)/config/coordinator.json
	cp config/simplecontroller.sample.json $(PLATFORM_DIST_PATH)/config/simplecontroller.json
	cp config/simulcontroller.sample.json $(PLATFORM_DIST_PATH)/config/simulcontroller.json
	cp LICENSE.txt $(PLATFORM_DIST_PATH)

	mv $(AGENT) $(PLATFORM_DIST_PATH)/bin
	mv $(API_SERVER) $(PLATFORM_DIST_PATH)/bin

	# Copy browser build directory if it exists
	if [ -d "browser/build" ]; then \
		cp -r browser/build $(PLATFORM_DIST_PATH)/browser; \
		cp -r browser/node_modules $(PLATFORM_DIST_PATH)/browser; \
		cp browser/package.json $(PLATFORM_DIST_PATH)/browser; \
		cp browser/.env $(PLATFORM_DIST_PATH)/; \
	fi

	$(eval PACKAGE_NAME=mattermost-load-test-ng-$(DIST_VER)-$(PLATFORM))
	cp -r $(PLATFORM_DIST_PATH) $(DIST_PATH)/$(PACKAGE_NAME)
	tar -C $(DIST_PATH) -czf $(DIST_PATH)/$(PACKAGE_NAME).tar.gz $(PACKAGE_NAME)
	rm -rf $(DIST_ROOT)/latest.tar.gz
	cp $(DIST_PATH)/$(PACKAGE_NAME).tar.gz $(DIST_ROOT)/latest.tar.gz
	rm -rf $(DIST_PATH)/$(PACKAGE_NAME)

verify-gomod: ## Run go mod verify.
	$(GO) mod download
	$(GO) mod verify

check-style: golangci-lint validate-json-configs ## Check the style of the code.

golangci-lint:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64

	@echo Running golangci-lint
	$(GOBIN)/golangci-lint run ./...

validate-json-configs:
	$(GO) run ./scripts/json_validator.go config/config.sample.json
	$(GO) run ./scripts/json_validator.go config/coordinator.sample.json
	$(GO) run ./scripts/json_validator.go config/deployer.sample.json
	$(GO) run ./scripts/json_validator.go config/comparison.sample.json
	$(GO) run ./scripts/json_validator.go config/gencontroller.sample.json
	$(GO) run ./scripts/json_validator.go config/simplecontroller.sample.json
	$(GO) run ./scripts/json_validator.go config/simulcontroller.sample.json

test: ## Run all tests.
	$(GO) test -v -mod=readonly -failfast -race -tags=integration ./...

MATCH=v.+\/mattermost-load-test-ng-v.+-linux-amd64.tar.gz
REPLACE=$(NEXT_VER)\/mattermost-load-test-ng-$(NEXT_VER)-linux-amd64.tar.gz
TAG_EXISTS=$(shell git rev-parse $(NEXT_VER) >/dev/null 2>&1; echo $$?)
BRANCH_NAME=bump-$(NEXT_VER)
BRANCH_EXISTS=$(shell git rev-parse $(BRANCH_NAME) >/dev/null 2>&1; echo $$?)
PR_URL=https://github.com/mattermost/mattermost-load-test-ng/compare/master...$(BRANCH_NAME)?quick_pull=1&labels=2:+Dev+Review
CURR_BRANCH=$(shell git branch --show-current)

release: ## Perform the release of a new version, identified by the envvar NEXT_VER. Install goreleaser if needed.
ifndef NEXT_VER
	@echo "Error: NEXT_VER must be defined"
else
ifeq ($(TAG_EXISTS), 0)
	@echo "Error: tag ${NEXT_VER} already exists"
else
	go install github.com/goreleaser/goreleaser@latest
	@echo -n "Release will be created from branch $(CURR_BRANCH). "
	@echo -n "Do you want to continue? [y/N] " && read ans && if [ $${ans:-'N'} != 'y' ]; then exit 1; fi
	git pull
	git tag $(NEXT_VER)
	git push origin $(NEXT_VER)
	goreleaser --clean
endif
endif

update-dependencies: ## Uses go get -u to update all dependencies and go mod tidy to clean up after itself.
	@echo Updating dependencies
	$(GO) get -u ./... # Update all dependencies (does not update across major versions)
	$(GO) get -u github.com/mattermost/mattermost/server/v8@master
	$(GO) get -u github.com/mattermost/mattermost/server/public@master
	$(GO) mod tidy # Tidy up

clean: ## Remove all generated files to start from scratch.
	rm -f errors.log cache.db stats.log status.log
	rm -f .installdeps
	rm -f loadtest.log
	rm -rf $(DIST_ROOT)

## Help documentation à la https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Print this help text.
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' ./Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-19s\033[0m %s\n", $$1, $$2}'
	@echo
