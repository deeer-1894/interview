package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HTTPTransportConfig struct {
	BaseURL   string
	Token     string
	Group     string
	Timeout   time.Duration
	ListPath  string
	CallPath  string
	UserAgent string
}

type HTTPTransport struct {
	baseURL   string
	group     string
	token     string
	listURL   string
	callURL   string
	userAgent string
	client    *http.Client
}

func NewHTTPTransport(cfg HTTPTransportConfig) (*HTTPTransport, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("mcp remote base url is required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	listPath := firstNonEmpty(cfg.ListPath, "/tools/list")
	callPath := firstNonEmpty(cfg.CallPath, "/tools/call")
	userAgent := firstNonEmpty(cfg.UserAgent, "mockinterview-mcp-client/1.0")
	return &HTTPTransport{
		baseURL:   baseURL,
		group:     strings.TrimSpace(cfg.Group),
		token:     strings.TrimSpace(cfg.Token),
		listURL:   joinURL(baseURL, listPath),
		callURL:   joinURL(baseURL, callPath),
		userAgent: userAgent,
		client:    &http.Client{Timeout: timeout},
	}, nil
}

func (t *HTTPTransport) ListTools(ctx context.Context, req ListToolsRequest) (ListToolsResponse, error) {
	var response ListToolsResponse
	payload := req
	if payload.Group == "" {
		payload.Group = t.group
	}
	if err := t.post(ctx, t.listURL, payload, &response); err != nil {
		return ListToolsResponse{}, err
	}
	return response, nil
}

func (t *HTTPTransport) CallTool(ctx context.Context, req CallToolRequest) (CallToolResponse, error) {
	var response CallToolResponse
	payload := req
	if payload.Group == "" {
		payload.Group = t.group
	}
	if err := t.post(ctx, t.callURL, payload, &response); err != nil {
		return CallToolResponse{}, err
	}
	return response, nil
}

func (t *HTTPTransport) post(ctx context.Context, target string, reqBody any, out any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal mcp request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build mcp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", t.userAgent)
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("send mcp request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mcp remote returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode mcp response: %w", err)
	}
	return nil
}

func joinURL(baseURL, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return baseURL + "/" + strings.TrimLeft(path, "/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
