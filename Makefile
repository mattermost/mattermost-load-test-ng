.PHONY: install clean

GO=go

DIST_ROOT=dist
# We specify version for the build; it is the branch-name by default, also we try
# to find if there is a tag pointed to the current commit. If so, we use the tag.
DIST_VER=$(shell git rev-parse --abbrev-ref HEAD)
ifeq ($(shell git describe --tags $(git rev-parse @) >&/dev/null; echo $$?), 0)
	DIST_VER=$(shell git describe --tags $(git rev-parse @))
endif
DIST_PATH=$(DIST_ROOT)/$(DIST_VER)
STATUS=$(shell git diff-index --quiet HEAD --; echo $$?)

COORDINATOR=lt-coordinator
COORDINATOR_ARGS=-mod=readonly -trimpath ./cmd/coordinator
LOADTEST=lt-agent
LOADTEST_ARGS=-mod=readonly -trimpath ./cmd/loadtest

# GOOS/GOARCH of the build host, used to determine whether we're cross-compiling or not
BUILDER_GOOS_GOARCH="$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)"

all: install

build-linux:
	@echo Build Linux amd64
	env GOOS=linux GOARCH=amd64 $(GO) build -o $(COORDINATOR) $(COORDINATOR_ARGS)
	env GOOS=linux GOARCH=amd64 $(GO) build -o $(LOADTEST) $(LOADTEST_ARGS)

build-osx:
	@echo Build OSX amd64
	env GOOS=darwin GOARCH=amd64 $(GO) build -o $(COORDINATOR) $(COORDINATOR_ARGS)
	env GOOS=darwin GOARCH=amd64 $(GO) build -o $(LOADTEST) $(LOADTEST_ARGS)

build-windows:
	@echo Build Windows amd64
	env GOOS=windows GOARCH=amd64 $(GO) build -o $(COORDINATOR) $(COORDINATOR_ARGS)
	env GOOS=windows GOARCH=amd64 $(GO) build -o $(LOADTEST) $(LOADTEST_ARGS)

assets:
	go get github.com/kevinburke/go-bindata/go-bindata/...
	go generate ./...

build: assets build-linux build-windows build-osx

# Build and install for the current platform
install:
	$(GO) install $(COORDINATOR_ARGS)
	$(GO) install $(LOADTEST_ARGS)

# We only support Linux to package for now. Package manually for other targets.
package:
ifneq ($(STATUS), 0)
	@echo Warning: Repository has uncommitted changes.
endif
	@$(MAKE) build-linux
	rm -rf $(DIST_ROOT)
	$(eval PLATFORM=linux_amd64)
	$(eval PLATFORM_DIST_PATH=$(DIST_PATH)/$(PLATFORM))
	mkdir -p $(PLATFORM_DIST_PATH)
	mkdir -p $(PLATFORM_DIST_PATH)/config
	mkdir -p $(PLATFORM_DIST_PATH)/bin

	cp config/config.default.json $(PLATFORM_DIST_PATH)/config/config.json
	cp config/coordinator.default.json $(PLATFORM_DIST_PATH)/config/coordinator.json
	cp config/simplecontroller.default.json $(PLATFORM_DIST_PATH)/config/simplecontroller.json
	cp README.md $(PLATFORM_DIST_PATH)

	mv $(COORDINATOR) $(PLATFORM_DIST_PATH)/bin
	mv $(LOADTEST) $(PLATFORM_DIST_PATH)/bin
	tar -C $(PLATFORM_DIST_PATH) -czf $(DIST_PATH)/mattermost-load-test-ng_$(DIST_VER)_$(PLATFORM).tar.gz ./

verify-gomod:
	$(GO) mod download
	$(GO) mod verify

check-style: golangci-lint

golangci-lint:
# https://stackoverflow.com/a/677212/1027058 (check if a command exists or not)
	@if ! [ -x "$$(command -v golangci-lint)" ]; then \
		echo "golangci-lint is not installed. Please see https://github.com/golangci/golangci-lint#install for installation instructions."; \
		exit 1; \
	fi; \

	@echo Running golangci-lint
	golangci-lint run ./...

test:
	@if ! [ -e config/simplecontroller.json ]; then \
    	cp config/simplecontroller.default.json config/simplecontroller.json; \
 	fi;\
	$(GO) test -v -mod=readonly -failfast ./...

clean:
	rm -f errors.log cache.db stats.log status.log
	rm -f .installdeps
	rm -f loadtest.log
	rm -rf $(DIST_ROOT)
