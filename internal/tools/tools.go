package tools

import (
	parcaclient "github.com/davi17g/parcaprof-mcp/internal/parca"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register attaches all Parca tools to the MCP server.
func Register(s *mcp.Server, c *parcaclient.Client) {
	registerLabels(s, c)
	registerQuery(s, c)
	registerReport(s, c)
}
