# syntax=docker/dockerfile:1.7
#
# Multi-stage build for mattermost-load-test-ng.
# Produces a single image containing ltagent, ltapi, ltcoordinator, and ltctl.
# The desired binary is selected per container via the command (e.g. `docker run ... ltapi`).
#
# Build (single arch, amd64):
#   docker build -t mattermost/mattermost-load-test-ng:vX.Y.Z .
#
# Build (multi-arch via buildx):
#   docker buildx build --platform linux/amd64,linux/arm64 \
#     -t mattermost/mattermost-load-test-ng:vX.Y.Z --push .
#
# Run a manual load test (most common usage):
#   docker run --rm \
#     -v $(pwd)/config.json:/mattermost-load-test/config/config.json:ro \
#     mattermost/mattermost-load-test-ng:vX.Y.Z ltagent -n 100 -d 600

# ---------- Stage 1: build ----------
# BUILDPLATFORM runs the toolchain natively on the build host while cross-compiling
# to TARGETOS/TARGETARCH. Single-arch builds work too — TARGETOS/TARGETARCH default
# to the build host's values when --platform isn't supplied.
FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS build
# Verify latest 1.23.x patch before each release:
#   curl -s 'https://go.dev/dl/?mode=json' | jq -r '.[0].version'

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# Module download as a cacheable layer so source changes don't re-download dependencies.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/ltagent       ./cmd/ltagent && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/ltapi         ./cmd/ltapi && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/ltcoordinator ./cmd/ltcoordinator && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/ltctl         ./cmd/ltctl

# ---------- Stage 2: runtime ----------
FROM debian:12-slim
# Verify latest patch tag before each release: docker pull debian:12-slim
# CVE scan: trivy image debian:12-slim

# Debug tooling expected when operators shell into a running container to investigate.
# For a hardened minimal image, swap base to gcr.io/distroless/static-debian12:nonroot
# and drop this apt-get block — at the cost of losing curl/jq/dig for troubleshooting.
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      jq \
      netcat-openbsd \
      net-tools \
      iproute2 \
      dnsutils \
      tini \
    && rm -rf /var/lib/apt/lists/*

# Non-root user so the image satisfies common admission policies out of the box.
RUN groupadd --system --gid 65532 mmlt \
    && useradd --system --uid 65532 --gid mmlt --home-dir /mattermost-load-test --shell /sbin/nologin mmlt

# Copy all four binaries into PATH.
COPY --from=build /out/ltagent /out/ltapi /out/ltcoordinator /out/ltctl /usr/local/bin/

# Sample configs baked as a fallback baseline. Real configs come from a runtime mount:
#   -v /path/to/config.json:/mattermost-load-test/config/config.json:ro
RUN mkdir -p /mattermost-load-test/config /mattermost-load-test/logs
COPY config/config.sample.json           /mattermost-load-test/config/config.json
COPY config/coordinator.sample.json      /mattermost-load-test/config/coordinator.json
COPY config/simulcontroller.sample.json  /mattermost-load-test/config/simulcontroller.json
COPY config/simplecontroller.sample.json /mattermost-load-test/config/simplecontroller.json
RUN chown -R mmlt:mmlt /mattermost-load-test

USER mmlt
WORKDIR /mattermost-load-test

# ltapi listens here when used as the entrypoint. Override per role via the CMD arg.
EXPOSE 4000

# tini reaps zombies and forwards SIGTERM/SIGINT so `docker stop` exits cleanly for
# whichever binary is chosen at runtime.
ENTRYPOINT ["/usr/bin/tini", "--"]
# Default to ltapi; operators override per role: `docker run ... <image> ltagent ...`
CMD ["ltapi", "--port=4000"]

# OCI labels — version/revision filled in by the release workflow via build args.
LABEL org.opencontainers.image.title="mattermost-load-test-ng"
LABEL org.opencontainers.image.description="Load testing toolkit for Mattermost"
LABEL org.opencontainers.image.source="https://github.com/mattermost/mattermost-load-test-ng"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.vendor="Mattermost, Inc."
