package mcp

import "context"

type Transport interface {
	ListTools(ctx context.Context, req ListToolsRequest) (ListToolsResponse, error)
	CallTool(ctx context.Context, req CallToolRequest) (CallToolResponse, error)
}
