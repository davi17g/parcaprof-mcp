go through the SHELL = bash
NAME = parcaprof-mcp
WORKSPACE = $(shell pwd)
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git rev-parse --abbrev-ref HEAD 2>/dev/null || echo dev)

GO ?= $(shell which go || echo /usr/local/go/bin/go)
OS ?= $(shell $(GO) env GOOS)
ARCH ?= $(shell $(GO) env GOARCH)
REGISTRY ?= docker.io
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_MODE ?= release

ifeq ($(filter debug release,$(BUILD_MODE)),)
$(error BUILD_MODE must be 'debug' or 'release', got '$(BUILD_MODE)')
endif

LDFLAGS_COMMON = \
-X 'main.appVersion=$(VERSION)' \
-X 'main.commitHash=$(GIT_COMMIT)' \
-X 'main.buildTime=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')'

GOBUILD_debug   = GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 $(GO) build -tags debug -gcflags "all=-N -l" -ldflags="$(LDFLAGS_COMMON)"
GOBUILD_release = GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w $(LDFLAGS_COMMON)"
GOBUILD = $(GOBUILD_$(BUILD_MODE))
GOTEST = $(GO) test
NPROC := $(shell nproc 2>/dev/null || getconf _NPROCESSORS_ONLN)

ARCHS ?= linux/amd64 linux/arm64
IMAGE_TAG ?= test
IMAGE_TAG_LATEST ?= false
IMAGE_REPO ?= davi17g/parcaprof-mcp
IMAGE_CACHE_FROM ?=
IMAGE_CACHE_TO ?=
IMAGE_OUTPUT ?= type=image,push=true

BINARY_NAME = parcaprof-mcp
TARGET_DIR = $(WORKSPACE)/dist
BIN_DIR = $(WORKSPACE)/bin
CMD_DIR = $(WORKSPACE)/cmd/parcaprof-mcp

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=

.PHONY: test
test:
	$(GOTEST) -parallel $(NPROC) -timeout=5m -count=1 -v ./...

.PHONY: coverage
coverage:
	$(GOTEST) -parallel $(NPROC) -timeout=5m -count=1 ./... -coverprofile to_filter.cov -coverpkg ./...
	grep -v "test\|mocks" to_filter.cov > coverage.cov
	rm -f to_filter.cov
	$(GO) tool cover -func coverage.cov

.PHONY: clean
clean:
	rm -Rf $(BIN_DIR) $(TARGET_DIR)

.PHONY: build
build:
	mkdir -p "$(TARGET_DIR)"
	@echo "Building $(BINARY_NAME) $(VERSION) ($(BUILD_MODE)) for $(OS)/$(ARCH)..."
	$(GOBUILD) -o $(TARGET_DIR)/$(BINARY_NAME)_$(OS)_$(ARCH) $(CMD_DIR)

.PHONY: buildx
buildx:
	@for arch in $(ARCHS); do \
		OS=$$(echo $$arch | cut -d/ -f1); \
		ARCH=$$(echo $$arch | cut -d/ -f2); \
		OS=$$OS ARCH=$$ARCH $(MAKE) build; \
	done

.PHONY: install
install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(TARGET_DIR)/$(BINARY_NAME)_$(OS)_$(ARCH) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY_NAME)

.PHONY: docker-build
docker-build:
	DOCKER_BUILDKIT=1 docker build \
		--progress=plain \
		--tag $(IMAGE_REPO):$(IMAGE_TAG) \
		--build-arg REGISTRY=$(REGISTRY) \
		--build-arg BUILD_MODE=$(BUILD_MODE) \
		--build-arg VERSION=$(VERSION) \
		--file $(WORKSPACE)/Dockerfile .

.PHONY: docker-buildx
docker-buildx:
	cd ./scripts && ./docker-buildx.sh \
		--repo $(IMAGE_REPO) \
		--tag $(IMAGE_TAG) \
		--tag-latest $(IMAGE_TAG_LATEST) \
		--registry $(REGISTRY) \
		--version $(VERSION) \
		--platforms "$(ARCHS)" \
		--cache-to "$(IMAGE_CACHE_TO)" \
		--cache-from "$(IMAGE_CACHE_FROM)" \
		--output "$(IMAGE_OUTPUT)" \
		--build-mode "$(BUILD_MODE)"

.PHONY: run
run: build
	$(TARGET_DIR)/$(BINARY_NAME)_$(OS)_$(ARCH) --parca-address=localhost:7070 --parca-insecure --transport=http
