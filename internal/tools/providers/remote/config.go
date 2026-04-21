package remote

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Enabled   bool
	BaseURL   string
	Token     string
	Group     string
	Timeout   time.Duration
	ListPath  string
	CallPath  string
	Transport string
}

func ConfigFromEnv() Config {
	timeout := 8 * time.Second
	if raw := strings.TrimSpace(os.Getenv("MCP_REMOTE_TIMEOUT_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}
	return Config{
		Enabled:   strings.EqualFold(strings.TrimSpace(os.Getenv("MCP_REMOTE_ENABLED")), "true"),
		BaseURL:   strings.TrimSpace(os.Getenv("MCP_REMOTE_BASE_URL")),
		Token:     strings.TrimSpace(os.Getenv("MCP_REMOTE_TOKEN")),
		Group:     strings.TrimSpace(os.Getenv("MCP_REMOTE_GROUP")),
		Timeout:   timeout,
		ListPath:  strings.TrimSpace(os.Getenv("MCP_REMOTE_LIST_PATH")),
		CallPath:  strings.TrimSpace(os.Getenv("MCP_REMOTE_CALL_PATH")),
		Transport: strings.TrimSpace(os.Getenv("MCP_REMOTE_TRANSPORT")),
	}
}
