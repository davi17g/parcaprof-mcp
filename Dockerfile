# syntax=docker/dockerfile:1.24.0

ARG GO_VERSION=1.26.4
ARG REGISTRY="docker.io"

FROM --platform=$BUILDPLATFORM ${REGISTRY}/tonistiigi/xx:1.9.0 AS xx
FROM --platform=$BUILDPLATFORM ${REGISTRY}/golang:${GO_VERSION} AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG BUILD_MODE=release

COPY --from=xx / /

WORKDIR /app/parcaprof-mcp
COPY . .

RUN xx-go --wrap

RUN --mount=type=secret,id=GOPROXY <<-EOF
    if [ -s /run/secrets/GOPROXY ]; then
        export GOPROXY="$(cat /run/secrets/GOPROXY)"
    fi
    go mod download
EOF

RUN --mount=type=secret,id=GOPROXY <<-EOF
    if [ -s /run/secrets/GOPROXY ]; then
        export GOPROXY="$(cat /run/secrets/GOPROXY)"
    fi
    OS="${TARGETOS}" ARCH="${TARGETARCH}" VERSION="${VERSION}" make build BUILD_MODE="${BUILD_MODE}"
    xx-verify /app/parcaprof-mcp/dist/parcaprof-mcp_${TARGETOS}_${TARGETARCH}
EOF

FROM ${REGISTRY}/alpine:3.23

ARG TARGETOS
ARG TARGETARCH

RUN apk update && \
    apk upgrade --no-cache && \
    apk add --no-cache ca-certificates

RUN addgroup -g 65532 -S parcagroup && \
    adduser -S -u 65532 -G parcagroup -h /home/parcauser parcauser

COPY --chown=parcauser:parcagroup --chmod=0755 --from=builder \
    /app/parcaprof-mcp/dist/parcaprof-mcp_${TARGETOS}_${TARGETARCH} \
    /usr/bin/parcaprof-mcp

USER parcauser
EXPOSE 8080
ENTRYPOINT ["/usr/bin/parcaprof-mcp"]
CMD ["--transport=http", "--http-addr=:8080"]
