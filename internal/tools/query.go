package tools

import (
	"context"
	"fmt"
	"time"

	parcaclient "github.com/davi17g/parcaprof-mcp/internal/parca"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type queryRangeInput struct {
	Query   string   `json:"query" jsonschema:"PromQL-style selector, e.g. 'parca_agent:samples:count:cpu:nanoseconds:delta{job=\"api\"}'."`
	Start   string   `json:"start,omitempty"`
	End     string   `json:"end,omitempty"`
	StepSec int32    `json:"step_seconds,omitempty" jsonschema:"step in seconds (optional)."`
	Limit   uint32   `json:"limit,omitempty"`
	SumBy   []string `json:"sum_by,omitempty"`
}

type querySingleInput struct {
	Query string `json:"query" jsonschema:"PromQL-style selector for the merged profile."`
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

func registerQuery(s *mcp.Server, c *parcaclient.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "parca_query_range",
		Description: "Query a profile metric over a time range, returning one or more series of cumulative samples.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args queryRangeInput) (*mcp.CallToolResult, any, error) {
		if args.Query == "" {
			return errResult(fmt.Errorf("query is required")), nil, nil
		}
		start, end, err := Window(args.Start, args.End)
		if err != nil {
			return errResult(err), nil, nil
		}
		req := &querypb.QueryRangeRequest{
			Query: args.Query, Start: start, End: end,
			Limit: args.Limit, SumBy: args.SumBy,
		}
		if args.StepSec > 0 {
			req.Step = durationpb.New(time.Duration(args.StepSec) * time.Second)
		}
		resp, err := c.Query.QueryRange(ctx, req)
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(resp), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "parca_query_single",
		Description: "Run a merged profile query and return a compact summary (total, filtered, unit, reported node count). Use parca_top or parca_flamegraph for richer output.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args querySingleInput) (*mcp.CallToolResult, any, error) {
		if args.Query == "" {
			return errResult(fmt.Errorf("query is required")), nil, nil
		}
		start, end, err := Window(args.Start, args.End)
		if err != nil {
			return errResult(err), nil, nil
		}
		resp, err := mergedQuery(ctx, c, args.Query, start, end, querypb.QueryRequest_REPORT_TYPE_TOP)
		if err != nil {
			return errResult(err), nil, nil
		}
		out := map[string]any{
			"total":    resp.GetTotal(),
			"filtered": resp.GetFiltered(),
		}
		if top := resp.GetTop(); top != nil {
			out["unit"] = top.GetUnit()
			out["reported_nodes"] = top.GetReported()
		}
		return jsonResult(out), nil, nil
	})
}

func mergedQuery(ctx context.Context, c *parcaclient.Client, q string, start, end *timestamppb.Timestamp, rt querypb.QueryRequest_ReportType) (*querypb.QueryResponse, error) {
	return c.Query.Query(ctx, &querypb.QueryRequest{
		Mode:       querypb.QueryRequest_MODE_MERGE,
		ReportType: rt,
		Options: &querypb.QueryRequest_Merge{
			Merge: &querypb.MergeProfile{Query: q, Start: start, End: end},
		},
	})
}
