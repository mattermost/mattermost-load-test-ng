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

AGENT=ltagent
AGENT_ARGS=-mod=readonly -trimpath ./cmd/ltagent
API_SERVER=ltapi
API_SERVER_ARGS=-mod=readonly -trimpath ./cmd/ltapi

GOBIN=$(PWD)/bin
PATH=$(shell printenv PATH):$(GOBIN)

# GOOS/GOARCH of the build host, used to determine whether we're cross-compiling or not
BUILDER_GOOS_GOARCH="$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)"

all: install

build-linux:
	@echo Build Linux amd64
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(AGENT) $(AGENT_ARGS)
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(API_SERVER) $(API_SERVER_ARGS)

build-osx:
	@echo Build OSX amd64
	env GOOS=darwin GOARCH=amd64 $(GO) build -o $(AGENT) $(AGENT_ARGS)
	env GOOS=darwin GOARCH=amd64 $(GO) build -o $(API_SERVER) $(API_SERVER_ARGS)

build-windows:
	@echo Build Windows amd64
	env GOOS=windows GOARCH=amd64 $(GO) build -o $(AGENT) $(AGENT_ARGS)
	env GOOS=windows GOARCH=amd64 $(GO) build -o $(API_SERVER) $(API_SERVER_ARGS)

assets:
	go install github.com/kevinburke/go-bindata/go-bindata@v3.23.0
	go generate ./...
	go fmt ./...

build: assets build-linux build-windows build-osx

# Build and install for the current platform
install:
	$(GO) install $(API_SERVER_ARGS)

# We only support Linux to package for now. Package manually for other targets.
package:
ifneq ($(STATUS), 0)
	@echo Warning: Repository has uncommitted changes.
endif
	@$(MAKE) build-linux
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
	$(eval PACKAGE_NAME=mattermost-load-test-ng-$(DIST_VER)-$(PLATFORM))
	cp -r $(PLATFORM_DIST_PATH) $(DIST_PATH)/$(PACKAGE_NAME)
	tar -C $(DIST_PATH) -czf $(DIST_PATH)/$(PACKAGE_NAME).tar.gz $(PACKAGE_NAME)
	rm -rf $(DIST_PATH)/$(PACKAGE_NAME)

verify-gomod:
	$(GO) mod download
	$(GO) mod verify

check-style: golangci-lint

golangci-lint:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.2

	@echo Running golangci-lint
	golangci-lint run ./...

test:
	$(GO) test -v -mod=readonly -failfast -race ./...

MATCH=v.+\/mattermost-load-test-ng-v.+-linux-amd64.tar.gz
REPLACE=$(NEXT_VER)\/mattermost-load-test-ng-$(NEXT_VER)-linux-amd64.tar.gz
TAG_EXISTS=$(shell git rev-parse $(NEXT_VER) >/dev/null 2>&1; echo $$?)

release:
	@if [ -z "${NEXT_VER}" ]; then \
		echo "Error: NEXT_VER must be defined"; \
		exit 1; \
	else \
		if [ "${TAG_EXISTS}" -eq 0 ]; then \
		  echo "Error: tag ${NEXT_VER} already exists"; \
			exit 1; \
		else \
			if ! [ -x "$$(command -v goreleaser)" ]; then \
			echo "goreleaser is not installed, do you want to download it? [y/N] " && read ans && [ $${ans:-N} = y ]; \
				if [ $$ans = y ] || [ $$ans = Y ]  ; then \
					curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh; \
				else \
					echo "aborting make release."; \
					exit 1; \
				fi; \
			fi; \
			for file in $(shell grep -rPl --include="*.go" --include="*.json" $(MATCH)); do \
			sed -r -i 's/$(MATCH)/$(REPLACE)/g' $$file; \
			done; \
			git commit -a -m 'Releasing $(NEXT_VER)'; \
			git tag $(NEXT_VER); \
			goreleaser --rm-dist; \
		fi; \
	fi;\

clean:
	rm -f errors.log cache.db stats.log status.log
	rm -f .installdeps
	rm -f loadtest.log
	rm -rf $(DIST_ROOT)
