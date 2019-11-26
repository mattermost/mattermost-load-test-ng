.PHONY: install clean

GOFLAGS ?= $(GOFLAGS:)
GO=go

DIST_ROOT=dist
DIST_FOLDER_NAME=mattermost-load-test
DIST_PATH=$(DIST_ROOT)/$(DIST_FOLDER_NAME)

# GOOS/GOARCH of the build host, used to determine whether we're cross-compiling or not
BUILDER_GOOS_GOARCH="$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)"

all: install

build-linux:
	@echo Build Linux amd64
	env GOOS=linux GOARCH=amd64 $(GO) install -i $(GOFLAGS) $(GO_LINKER_FLAGS) ./...

build-osx:
	@echo Build OSX amd64
	env GOOS=darwin GOARCH=amd64 $(GO) install -i $(GOFLAGS) $(GO_LINKER_FLAGS) ./...

build-windows:
	@echo Build Windows amd64
	env GOOS=windows GOARCH=amd64 $(GO) install -i $(GOFLAGS) $(GO_LINKER_FLAGS) ./...

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

	cp loadtestconfig.default.json $(DIST_PATH)/loadtestconfig.json
	cp README.md $(DIST_PATH)
	cp -r testfiles $(DIST_PATH)

	@# ----- PLATFORM SPECIFIC -----

	@# Linux, the only supported package version for now. Build manually for other targets.
ifeq ($(BUILDER_GOOS_GOARCH),"linux_amd64")
	cp $(GOPATH)/bin/loadtest $(DIST_PATH)/bin # from native bin dir, not cross-compiled
else
	cp $(GOPATH)/bin/linux_amd64/loadtest $(DIST_PATH)/bin # from cross-compiled bin dir
endif
	tar -C $(DIST_ROOT) -czf $(DIST_PATH).tar.gz $(DIST_FOLDER_NAME)

clean:
	rm -f errors.log cache.db stats.log status.log
	rm -f ./cmd/loadtest/loadtest
	rm -f .installdeps
	rm -f loadtest.log
	rm -rf $(DIST_ROOT)
