# parcaprof-mcp

MCP server that exposes a [Parca](https://www.parca.dev/) continuous-profiling
backend to MCP clients (Claude Desktop, Claude Code, MCP Inspector, …) so an
LLM can explore profile types, label sets, time-range series, and per-function
hotspots (top / flamegraph) over a Parca server's gRPC API.

Built with the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

## Transports

| MCP transport | Why |
|---|---|
| **Streamable HTTP** (`/mcp`) | Current MCP spec; preferred for remote clients |
| **SSE** (`/sse`) | Legacy spec; kept for older clients |
| **stdio** | For local install in Claude Desktop / Claude Code |

> The MCP spec defines only stdio and Streamable HTTP. There is no
> MCP-over-gRPC transport. gRPC here is used only to talk **to Parca**.

## Tools

| Tool | Parca RPC |
|---|---|
| `parca_profile_types` | `ProfileTypes` |
| `parca_labels` | `Labels` |
| `parca_label_values` | `Values` |
| `parca_series` | `Series` |
| `parca_query_range` | `QueryRange` |
| `parca_query_single` | `Query` (compact summary) |
| `parca_top` | `Query` with `REPORT_TYPE_TOP`, sorted top-N rows |
| `parca_flamegraph` | `Query` with `REPORT_TYPE_FLAMEGRAPH_TABLE`, flattened & aggregated |

Times accept RFC3339 or relative `now`, `now-15m`, `now-1h30m`, `now+5m`. The
default window is the last 15 minutes.

## Flags

| Flag | Default | Description |
|---|---|---|
| `--parca-address` | `localhost:7070` | Parca gRPC `host:port` |
| `--parca-insecure` | `false` | Use plaintext (no TLS) to Parca |
| `--transport` | `http` | `http` (Streamable HTTP + SSE) or `stdio` |
| `--http-addr` | `:8080` | HTTP listen address (when `--transport=http`) |
| `--version` | — | Print version, commit, and build time, then exit |

Env: `PARCA_BEARER_TOKEN` is sent as `authorization: Bearer …` on every gRPC call.

## Build

The Makefile injects version / commit / build-time via `-ldflags -X`, supports
debug and release modes, and cross-compiles for `linux/amd64,linux/arm64`.

```sh
make build                              # native binary into dist/
make buildx                             # cross-compile for all ARCHS
make build BUILD_MODE=debug             # no inlining/optimization (for dlv)
make test                               # go test ./...
make coverage                           # go test + filtered coverage.cov report
make install PREFIX=/usr/local          # install dist/<binary> to $PREFIX/bin
```

Versioning: `VERSION` is derived from `git describe --tags --exact-match`,
falling back to the current branch, then to `dev`. Override with
`make build VERSION=1.2.3`.

## Docker

Single-arch local build:

```sh
make docker-build IMAGE_REPO=davi17g/parcaprof-mcp IMAGE_TAG=dev
docker run --rm -p 8080:8080 davi17g/parcaprof-mcp:dev \
  --parca-address=host.docker.internal:7070 --parca-insecure
```

Multi-arch via `buildx bake` (`linux/amd64,linux/arm64`, OCI labels, GOPROXY
secret mount, optional registry cache):

```sh
make docker-buildx \
  IMAGE_REPO=davi17g/parcaprof-mcp \
  IMAGE_TAG=v0.1.0 \
  IMAGE_OUTPUT=type=image,push=true
```

The Dockerfile uses `tonistiigi/xx` for cross-compilation on a native
`$BUILDPLATFORM` Go builder, then ships the binary on `alpine` as non-root
`parcauser` (UID 65532). `GOPROXY` is consumed via a BuildKit
`--mount=type=secret` so private proxy credentials never land in image layers.

## Run

```sh
make run                                # build + run against localhost:7070 (insecure)
```

Or directly:

```sh
go run ./cmd/parcaprof-mcp \
  --parca-address=localhost:7070 \
  --parca-insecure \
  --transport=http \
  --http-addr=:8080
```

### Claude Desktop (stdio)

`~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "parca": {
      "command": "parcaprof-mcp",
      "args": ["--transport=stdio", "--parca-address=parca.example:7070"]
    }
  }
}
```

## Verify end-to-end

1. Start Parca: `docker run --rm -p 7070:7070 ghcr.io/parca-dev/parca:v0.28.0`.
2. Start this server with `--parca-insecure`.
3. `npx @modelcontextprotocol/inspector` → connect to `http://localhost:8080/mcp`
   → list tools → call `parca_profile_types`.

The HTTP transport also exposes `/healthz` (returns 200) for liveness probes.

## Repository layout

```
cmd/parcaprof-mcp/      # entrypoint: flags, server bootstrap, transport mux
internal/parca/         # gRPC client wrapper (TLS, bearer-token auth)
internal/tools/         # MCP tool handlers (labels, query, top, flamegraph)
scripts/                # docker-buildx.sh + docker-bake.hcl (multi-arch build)
Dockerfile              # xx cross-compile builder -> alpine runtime
Makefile                # build / test / docker targets
```

## CI

- `.github/workflows/tests.yml` — `make build` + `make coverage` on every push and PR.
- `.github/workflows/golangci-lint.yaml` — `golangci-lint` with the config in `.golangci.yaml`.
- `.github/workflows/docker-publish.yml` — multi-arch image push to **Docker Hub only**
  (`docker.io/davi17g/parcaprof-mcp`) on every `v*` git tag, plus a manual
  `workflow_dispatch` trigger. Tag pushes also publish `:latest`. Requires repo
  secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`.
- `.github/dependabot.yml` — weekly updates for Go modules, GitHub Actions, and Docker base images.

## License

[Apache-2.0](./LICENSE)
