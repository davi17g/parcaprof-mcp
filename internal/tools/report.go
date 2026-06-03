package tools

import (
	"context"
	"fmt"
	"sort"

	parcaclient "github.com/davi17g/parcaprof-mcp/internal/parca"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
)

type reportInput struct {
	Query string `json:"query" jsonschema:"PromQL-style selector for the merged profile."`
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
	Limit int    `json:"limit,omitempty" jsonschema:"max rows to return. Default: 50."`
}

type topRow struct {
	Function   string  `json:"function"`
	SystemName string  `json:"system_name,omitempty"`
	File       string  `json:"file,omitempty"`
	Line       int64   `json:"line,omitempty"`
	Flat       int64   `json:"flat"`
	Cumulative int64   `json:"cumulative"`
	FlatPct    float64 `json:"flat_pct"`
	CumPct     float64 `json:"cumulative_pct"`
}

func registerReport(s *mcp.Server, c *parcaclient.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "parca_top",
		Description: "Top-N hottest functions for a merged profile in a time window, sorted by cumulative cost.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args reportInput) (*mcp.CallToolResult, any, error) {
		if args.Query == "" {
			return errResult(fmt.Errorf("query is required")), nil, nil
		}
		limit := args.Limit
		if limit <= 0 {
			limit = 50
		}
		start, end, err := Window(args.Start, args.End)
		if err != nil {
			return errResult(err), nil, nil
		}
		resp, err := mergedQuery(ctx, c, args.Query, start, end, querypb.QueryRequest_REPORT_TYPE_TOP)
		if err != nil {
			return errResult(err), nil, nil
		}
		top := resp.GetTop()
		if top == nil {
			return jsonResult(map[string]any{"rows": []topRow{}, "total": resp.GetTotal()}), nil, nil
		}
		nodes := top.GetList()
		sort.SliceStable(nodes, func(i, j int) bool {
			return nodes[i].GetCumulative() > nodes[j].GetCumulative()
		})
		total := resp.GetTotal()
		denom := float64(total)
		if denom == 0 {
			denom = 1
		}
		rows := make([]topRow, 0, min(limit, len(nodes)))
		for i, n := range nodes {
			if i >= limit {
				break
			}
			r := topRow{Flat: n.GetFlat(), Cumulative: n.GetCumulative()}
			r.FlatPct = float64(r.Flat) / denom * 100
			r.CumPct = float64(r.Cumulative) / denom * 100
			if m := n.GetMeta(); m != nil {
				if f := m.GetFunction(); f != nil {
					r.Function = f.GetName()
					r.SystemName = f.GetSystemName()
					r.File = f.GetFilename()
				}
				if l := m.GetLine(); l != nil {
					r.Line = l.GetLine()
				}
			}
			rows = append(rows, r)
		}
		return jsonResult(map[string]any{
			"unit":  top.GetUnit(),
			"total": total,
			"rows":  rows,
		}), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "parca_flamegraph",
		Description: "Flamegraph for a merged profile in a time window, flattened to a sorted table aggregated by function (top-N by cumulative).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args reportInput) (*mcp.CallToolResult, any, error) {
		if args.Query == "" {
			return errResult(fmt.Errorf("query is required")), nil, nil
		}
		limit := args.Limit
		if limit <= 0 {
			limit = 50
		}
		start, end, err := Window(args.Start, args.End)
		if err != nil {
			return errResult(err), nil, nil
		}
		resp, err := mergedQuery(ctx, c, args.Query, start, end, querypb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_TABLE)
		if err != nil {
			return errResult(err), nil, nil
		}
		fg := resp.GetFlamegraph()
		if fg == nil || fg.GetRoot() == nil {
			return jsonResult(map[string]any{"rows": []any{}}), nil, nil
		}
		agg := map[string]int64{}
		var walk func(nodes []*querypb.FlamegraphNode)
		walk = func(nodes []*querypb.FlamegraphNode) {
			for _, n := range nodes {
				name := "<unknown>"
				if m := n.GetMeta(); m != nil {
					if f := m.GetFunction(); f != nil && f.GetName() != "" {
						name = f.GetName()
					}
				}
				agg[name] += n.GetCumulative()
				walk(n.GetChildren())
			}
		}
		walk(fg.GetRoot().GetChildren())

		type row struct {
			Function   string  `json:"function"`
			Cumulative int64   `json:"cumulative"`
			Pct        float64 `json:"cumulative_pct"`
		}
		total := fg.GetRoot().GetCumulative()
		denom := float64(total)
		if denom == 0 {
			denom = 1
		}
		rows := make([]row, 0, len(agg))
		for name, c := range agg {
			rows = append(rows, row{Function: name, Cumulative: c, Pct: float64(c) / denom * 100})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].Cumulative > rows[j].Cumulative })
		if len(rows) > limit {
			rows = rows[:limit]
		}
		return jsonResult(map[string]any{
			"unit":   fg.GetUnit(),
			"total":  total,
			"height": fg.GetHeight(),
			"rows":   rows,
		}), nil, nil
	})
}
