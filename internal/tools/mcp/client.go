package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

type Client struct {
	transport Transport
}

func NewClient(transport Transport) *Client {
	return &Client{transport: transport}
}

func (c *Client) ListTools(ctx context.Context, group string) ([]ToolDescriptor, error) {
	resp, err := c.transport.ListTools(ctx, ListToolsRequest{Group: group})
	if err != nil {
		return nil, err
	}
	return resp.Tools, nil
}

func (c *Client) Call(ctx context.Context, tool string, arguments map[string]any, out any, group string) error {
	resp, err := c.transport.CallTool(ctx, CallToolRequest{
		Tool:      tool,
		Arguments: arguments,
		Group:     group,
	})
	if err != nil {
		return err
	}
	if out == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil
	}
	if err := json.Unmarshal(resp.Result, out); err != nil {
		return fmt.Errorf("decode tool result %q: %w", tool, err)
	}
	return nil
}
