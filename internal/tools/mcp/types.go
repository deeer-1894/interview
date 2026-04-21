package mcp

import "encoding/json"

type TransportKind string

const (
	TransportHTTP           TransportKind = "http"
	TransportStreamableHTTP TransportKind = "streamable_http"
	TransportStdio          TransportKind = "stdio"
)

type ToolDescriptor struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
}

type ListToolsRequest struct {
	Group string `json:"group,omitempty"`
}

type ListToolsResponse struct {
	Tools []ToolDescriptor `json:"tools"`
}

type CallToolRequest struct {
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Group     string         `json:"group,omitempty"`
}

type CallToolResponse struct {
	Result json.RawMessage `json:"result"`
}
