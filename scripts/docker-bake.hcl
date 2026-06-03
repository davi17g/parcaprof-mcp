group "default" {
  targets = ["parcaprof-mcp"]
}

variable "CONTEXT"        { default = null }
variable "LATEST"         { default = false }
variable "TAG"            { default = "" }
variable "GIT_BRANCH"     { default = null }
variable "GIT_COMMIT_SHA" { default = null }
variable "VERSION"        { default = null }
variable "ISO8601"        { default = null }
variable "REPO"           { default = "davi17g/parcaprof-mcp" }
variable "PLATFORMS"      { default = "linux/amd64,linux/arm64" }
variable "REGISTRY"       { default = "docker.io" }
variable "GO_VERSION"     { default = "1.26.4" }
variable "CACHE_FROM"     { default = "" }
variable "CACHE_TO"       { default = "" }
variable "OUTPUT"         { default = "type=image,push=true" }
variable "BUILD_MODE"     { default = "release" }

function "norm" {
  params = [value]
  result = value == null || value == "" ? [] : length(regexall(" ", value)) > 0 ? split(" ", value) : [value]
}

function "tags" {
  params = [repo]
  result = LATEST == true ? [
    "${repo}:${TAG}",
    "${repo}:latest",
  ] : ["${repo}:${TAG}"]
}

target "parcaprof-mcp" {
  labels = {
    "org.opencontainers.image.title"         = "parcaprof-mcp"
    "org.opencontainers.image.description"   = "MCP server exposing a Parca continuous-profiling backend"
    "org.opencontainers.image.documentation" = "https://github.com/davi17g/parcaprof-mcp#readme"
    "org.opencontainers.image.base.name"     = "docker.io/alpine:3.23"
    "org.opencontainers.image.source"        = "https://github.com/davi17g/parcaprof-mcp/tree/${GIT_BRANCH}"
    "org.opencontainers.image.vendor"        = "davi17g"
    "org.opencontainers.image.version"       = "${VERSION}"
    "org.opencontainers.image.url"           = "https://github.com/davi17g/parcaprof-mcp"
    "org.opencontainers.image.licenses"      = "Apache-2.0"
    "org.opencontainers.image.revision"      = "${GIT_COMMIT_SHA}"
    "org.opencontainers.image.created"       = "${ISO8601}"
  }

  args = {
    GO_VERSION = "${GO_VERSION}"
    REGISTRY   = "${REGISTRY}"
    VERSION    = "${VERSION}"
    BUILD_MODE = "${BUILD_MODE}"
  }

  secret     = ["id=GOPROXY,env=GOPROXY"]
  context    = "${CONTEXT}"
  dockerfile = "Dockerfile"
  platforms  = split(",", replace("${PLATFORMS}", " ", ","))
  cache-to   = norm("${CACHE_TO}")
  cache-from = norm("${CACHE_FROM}")

  tags   = tags("${REPO}")
  output = norm("${OUTPUT}")
}
