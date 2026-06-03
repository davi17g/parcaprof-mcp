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

> Note: the MCP spec defines only stdio and Streamable HTTP. There is no
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

## Run

```sh
go run ./cmd/parcaprof-mcp \
  --parca-address=localhost:7070 \
  --parca-insecure \
  --transport=http \
  --http-addr=:8080
```

Bearer-token auth to Parca: set `PARCA_BEARER_TOKEN`.

Stdio for Claude Desktop (`~/Library/Application Support/Claude/claude_desktop_config.json`):

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

## Verify

1. Start Parca locally: `docker run --rm -p 7070:7070 ghcr.io/parca-dev/parca:latest`.
2. Start this server with `--parca-insecure`.
3. `npx @modelcontextprotocol/inspector` → connect to `http://localhost:8080/mcp` → list tools → call `parca_profile_types`.
