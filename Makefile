.PHONY: install clean

GO=go

DIST_ROOT=dist
DIST_FOLDER_NAME=mattermost-load-test-ng
DIST_PATH=$(DIST_ROOT)/$(DIST_FOLDER_NAME)

# GOOS/GOARCH of the build host, used to determine whether we're cross-compiling or not
BUILDER_GOOS_GOARCH="$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)"

all: install

build-linux:
	@echo Build Linux amd64
	env GOOS=linux GOARCH=amd64 $(GO) install -mod=readonly -trimpath ./...

build-osx:
	@echo Build OSX amd64
	env GOOS=darwin GOARCH=amd64 $(GO) install -mod=readonly -trimpath ./...

build-windows:
	@echo Build Windows amd64
	env GOOS=windows GOARCH=amd64 $(GO) install -mod=readonly -trimpath ./...

build: build-linux build-windows build-osx

# Build and install for the current platform
install:
ifeq ($(BUILDER_GOOS_GOARCH),"darwin_amd64")
	@$(MAKE) build-osx
endif
ifeq ($(BUILDER_GOOS_GOARCH),"windows_amd64")
	@$(MAKE) build-windows
endif
ifeq ($(BUILDER_GOOS_GOARCH),"linux_amd64")
	@$(MAKE) build-linux
endif

package: build-linux
	rm -rf $(DIST_ROOT)
	mkdir -p $(DIST_PATH)/bin

	cp config.default.json $(DIST_PATH)/config/config.json
	cp README.md $(DIST_PATH)
	#cp -r testfiles $(DIST_PATH)

	@# ----- PLATFORM SPECIFIC -----

	@# Linux, the only supported package version for now. Build manually for other targets.
ifeq ($(BUILDER_GOOS_GOARCH),"linux_amd64")
	cp $(GOPATH)/bin/loadtest-ng $(DIST_PATH)/bin # from native bin dir, not cross-compiled
else
	cp $(GOPATH)/bin/linux_amd64/loadtest-ng $(DIST_PATH)/bin # from cross-compiled bin dir
endif
	tar -C $(DIST_ROOT) -czf $(DIST_PATH).tar.gz $(DIST_FOLDER_NAME)

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
	@if ! [ -a config/simplecontroller.json ]; then \
    	cp config/simplecontroller.default.json config/simplecontroller.json; \
 	fi;\
	$(GO) test -v -mod=readonly -failfast ./...

clean:
	rm -f errors.log cache.db stats.log status.log
	rm -f ./cmd/loadtest/loadtest-ng
	rm -f .installdeps
	rm -f loadtest.log
	rm -rf $(DIST_ROOT)
