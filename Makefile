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

COORDINATOR=lt-coordinator
LOADTEST=lt-agent

# GOOS/GOARCH of the build host, used to determine whether we're cross-compiling or not
BUILDER_GOOS_GOARCH="$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)"

all: install

build-linux:
	@echo Build Linux amd64
	env GOOS=linux GOARCH=amd64 $(GO) build -o $(COORDINATOR) ./cmd/coordinator
	env GOOS=linux GOARCH=amd64 $(GO) build -o $(LOADTEST) ./cmd/loadtest

build-osx:
	@echo Build OSX amd64
	env GOOS=darwin GOARCH=amd64 $(GO) build -o $(COORDINATOR) ./cmd/coordinator
	env GOOS=darwin GOARCH=amd64 $(GO) build -o $(LOADTEST) ./cmd/loadtest

build-windows:
	@echo Build Windows amd64
	env GOOS=windows GOARCH=amd64 $(GO) build -o $(COORDINATOR) ./cmd/coordinator
	env GOOS=windows GOARCH=amd64 $(GO) build -o $(LOADTEST) ./cmd/loadtest

assets:
	go get github.com/kevinburke/go-bindata/go-bindata/...
	go generate ./...

build: assets build-linux build-windows build-osx

# Build and install for the current platform
install:
ifeq ($(BUILDER_GOOS_GOARCH),"darwin_amd64")
	@$(MAKE) build-osx
	@$(MAKE) install-gopath
endif
ifeq ($(BUILDER_GOOS_GOARCH),"windows_amd64")
	@$(MAKE) build-windows
	@$(MAKE) install-gopath
endif
ifeq ($(BUILDER_GOOS_GOARCH),"linux_amd64")
	@$(MAKE) build-linux
	@$(MAKE) install-gopath
endif

install-gopath: ; mv $(COORDINATOR) $(GOPATH)/bin; mv $(LOADTEST) $(GOPATH)/bin

# We only support Linux to package for now. Package manually for other targets.
package:
ifneq ($(git diff --shortstat 2> /dev/null | tail -n1),"")
	@echo Warning: Repository has uncommitted changes.
endif
	@$(MAKE) build-linux
	rm -rf $(DIST_ROOT)
	mkdir -p $(DIST_PATH)
	mkdir -p $(DIST_PATH)/config

	cp config/config.default.json $(DIST_PATH)/config/config.json
	cp config/coordinator.default.json $(DIST_PATH)/config/coordinator.json
	cp config/simplecontroller.default.json $(DIST_PATH)/config/simplecontroller.json
	cp README.md $(DIST_PATH)

	tar cf $(COORDINATOR)_$(DIST_VER)_linux_amd64.tar.gz $(COORDINATOR) && mv $(COORDINATOR)_$(DIST_VER)_linux_amd64.tar.gz $(DIST_PATH)/
	tar cf $(LOADTEST)_$(DIST_VER)_linux_amd64.tar.gz $(LOADTEST) && mv $(LOADTEST)_$(DIST_VER)_linux_amd64.tar.gz $(DIST_PATH)/
	rm $(COORDINATOR) $(LOADTEST)

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
	rm -f $(COORDINATOR)
	rm -f $(LOADTEST)
	rm -f .installdeps
	rm -f loadtest.log
	rm -rf $(DIST_ROOT)
