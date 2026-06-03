#!/bin/bash -e
WORKSPACE="$(git rev-parse --show-toplevel)"
REGISTRY="docker.io"
REPO="davi17g/parcaprof-mcp"
TAG_LATEST=false
TAG=""
VERSION=""
CACHE_TO=""
CACHE_FROM=""
OUTPUT="type=image,push=true"
PLATFORMS="linux/amd64,linux/arm64"
BUILD_MODE="release"

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
	case $1 in
	--repo)        REPO="$2";        shift 2 ;;
	--tag)         TAG="$2";         shift 2 ;;
	--tag-latest)  TAG_LATEST="$2";  shift 2 ;;
	--version)     VERSION="$2";     shift 2 ;;
	--platforms)   PLATFORMS="$2";   shift 2 ;;
	--registry)    REGISTRY="$2";    shift 2 ;;
	--cache-to)    CACHE_TO="$2";    shift 2 ;;
	--cache-from)  CACHE_FROM="$2";  shift 2 ;;
	--output)      OUTPUT="$2";      shift 2 ;;
	--build-mode)  BUILD_MODE="$2";  shift 2 ;;
	-* | --*)
		echo "Unknown option $1"; exit 1 ;;
	*)
		POSITIONAL_ARGS+=("$1"); shift ;;
	esac
done

if [[ "$BUILD_MODE" != "debug" && "$BUILD_MODE" != "release" ]]; then
	echo "BUILD_MODE must be 'debug' or 'release', got '$BUILD_MODE'"
	exit 1
fi

set -- "${POSITIONAL_ARGS[@]}"

# Auto-derive latest patch of the Go minor used in go.mod.
GO_MINOR="$(go mod edit -json | jq -r '.Go' | cut -d. -f1-2)"
GO_VERSION="$(curl -s 'https://go.dev/dl/?mode=json&include=all' \
	| jq -r --arg ver "go${GO_MINOR}" '.[] | select(.version | startswith($ver)) | .version' \
	| sort -V | tail -n1 | cut -c3- | tr -d '\n')"
GO_VERSION="${GO_VERSION:-$GO_MINOR}"

PLATFORMS="$PLATFORMS" \
	TAG="$TAG" \
	REPO="$REPO" \
	CACHE_TO="$CACHE_TO" \
	CACHE_FROM="$CACHE_FROM" \
	OUTPUT="$OUTPUT" \
	REGISTRY="$REGISTRY" \
	GOPROXY="$GOPROXY" \
	LATEST="$TAG_LATEST" \
	BUILD_MODE="$BUILD_MODE" \
	GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD)" \
	GIT_COMMIT_SHA="$(git rev-parse HEAD)" \
	VERSION="$VERSION" \
	GO_VERSION="$GO_VERSION" \
	ISO8601="$(LC_TIME=en_US.UTF-8 date "+%Y-%m-%dT%H:%M:%S%z")" \
	CONTEXT="$WORKSPACE" \
	docker buildx bake \
	--allow=fs.read="$WORKSPACE" \
	default \
	--progress plain \
	--file "$WORKSPACE/scripts/docker-bake.hcl"
