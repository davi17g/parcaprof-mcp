package tools

import (
	"context"
	"encoding/json"

	parcaclient "github.com/davi17g/parcaprof-mcp/internal/parca"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
)

type profileTypesInput struct {
	Start string `json:"start,omitempty" jsonschema:"start time (RFC3339 or 'now-15m'). Default: now-15m."`
	End   string `json:"end,omitempty" jsonschema:"end time (RFC3339 or 'now'). Default: now."`
}

type labelsInput struct {
	Start       string   `json:"start,omitempty" jsonschema:"start time. Default: now-15m."`
	End         string   `json:"end,omitempty" jsonschema:"end time. Default: now."`
	Match       []string `json:"match,omitempty" jsonschema:"PromQL-style selector(s) to restrict label discovery."`
	ProfileType string   `json:"profile_type,omitempty" jsonschema:"restrict to a single profile type."`
}

type valuesInput struct {
	LabelName   string   `json:"label_name" jsonschema:"label name whose values to enumerate, e.g. 'job' or '__name__'."`
	Start       string   `json:"start,omitempty"`
	End         string   `json:"end,omitempty"`
	Match       []string `json:"match,omitempty"`
	ProfileType string   `json:"profile_type,omitempty"`
}

type seriesInput struct {
	Match []string `json:"match" jsonschema:"PromQL-style selector(s) to match series."`
	Start string   `json:"start,omitempty"`
	End   string   `json:"end,omitempty"`
}

func registerLabels(s *mcp.Server, c *parcaclient.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "parca_profile_types",
		Description: "List profile types (cpu, alloc_space, etc.) available in the Parca server for a time window.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args profileTypesInput) (*mcp.CallToolResult, any, error) {
		start, end, err := Window(args.Start, args.End)
		if err != nil {
			return errResult(err), nil, nil
		}
		resp, err := c.Query.ProfileTypes(ctx, &querypb.ProfileTypesRequest{Start: start, End: end})
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(resp.GetTypes()), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "parca_labels",
		Description: "List label names available for a time window, optionally filtered by selector or profile type.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args labelsInput) (*mcp.CallToolResult, any, error) {
		start, end, err := Window(args.Start, args.End)
		if err != nil {
			return errResult(err), nil, nil
		}
		resp, err := c.Query.Labels(ctx, &querypb.LabelsRequest{
			Start: start, End: end, Match: args.Match, ProfileType: optStr(args.ProfileType),
		})
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(map[string]any{
			"label_names": resp.GetLabelNames(),
			"warnings":    resp.GetWarnings(),
		}), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "parca_label_values",
		Description: "List values for a given label name in a time window.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args valuesInput) (*mcp.CallToolResult, any, error) {
		start, end, err := Window(args.Start, args.End)
		if err != nil {
			return errResult(err), nil, nil
		}
		resp, err := c.Query.Values(ctx, &querypb.ValuesRequest{
			LabelName: args.LabelName, Start: start, End: end,
			Match: args.Match, ProfileType: optStr(args.ProfileType),
		})
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(map[string]any{
			"label_values": resp.GetLabelValues(),
			"warnings":     resp.GetWarnings(),
		}), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "parca_series",
		Description: "Discover series matching one or more PromQL-style selectors over a time window.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args seriesInput) (*mcp.CallToolResult, any, error) {
		start, end, err := Window(args.Start, args.End)
		if err != nil {
			return errResult(err), nil, nil
		}
		resp, err := c.Query.Series(ctx, &querypb.SeriesRequest{
			Start: start, End: end, Match: args.Match,
		})
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(resp), nil, nil
	})
}

func jsonResult(v any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}

func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func errResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}
