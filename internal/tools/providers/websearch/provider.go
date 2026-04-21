package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"mockinterview/internal/protocol"
)

type Provider struct {
	client  *http.Client
	baseURL string
	ttl     time.Duration
	mu      sync.Mutex
	cache   map[string]cacheEntry
}

func New() *Provider {
	timeout := 8 * time.Second
	if raw := strings.TrimSpace(os.Getenv("WEB_SEARCH_TIMEOUT_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	baseURL := strings.TrimSpace(os.Getenv("WEB_SEARCH_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.duckduckgo.com/"
	}

	ttl := 30 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("WEB_SEARCH_CACHE_TTL_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			ttl = time.Duration(seconds) * time.Second
		}
	}

	return &Provider{
		client:  &http.Client{Timeout: timeout},
		baseURL: baseURL,
		ttl:     ttl,
		cache:   make(map[string]cacheEntry),
	}
}

func (p *Provider) Search(ctx context.Context, query string, limit int) ([]protocol.WebSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("web search query is required")
	}
	if limit <= 0 {
		limit = 5
	}
	key := cacheKey(query, limit)
	if cached, ok := p.loadCache(key); ok {
		return cached, nil
	}

	endpoint, err := url.Parse(p.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse web search base url: %w", err)
	}

	params := endpoint.Query()
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_redirect", "1")
	params.Set("no_html", "1")
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build web search request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		if cached, ok := p.loadCache(key); ok {
			return cached, nil
		}
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if cached, ok := p.loadCache(key); ok {
			return cached, nil
		}
		return nil, nil
	}

	var payload duckResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		if cached, ok := p.loadCache(key); ok {
			return cached, nil
		}
		return nil, nil
	}

	results := make([]protocol.WebSearchResult, 0, limit)
	if strings.TrimSpace(payload.Heading) != "" || strings.TrimSpace(payload.AbstractText) != "" {
		results = append(results, protocol.WebSearchResult{
			Title:   firstNonEmpty(payload.Heading, query),
			URL:     payload.AbstractURL,
			Snippet: payload.AbstractText,
		})
	}

	appendTopics(&results, payload.RelatedTopics, limit)
	if len(results) > limit {
		results = results[:limit]
	}
	p.saveCache(key, results)
	return results, nil
}

type cacheEntry struct {
	results   []protocol.WebSearchResult
	expiresAt time.Time
}

type duckResponse struct {
	Heading       string      `json:"Heading"`
	AbstractText  string      `json:"AbstractText"`
	AbstractURL   string      `json:"AbstractURL"`
	RelatedTopics []duckTopic `json:"RelatedTopics"`
}

type duckTopic struct {
	Text     string      `json:"Text"`
	FirstURL string      `json:"FirstURL"`
	Topics   []duckTopic `json:"Topics"`
	Result   string      `json:"Result"`
	Icon     any         `json:"Icon"`
	Name     string      `json:"Name"`
}

func appendTopics(results *[]protocol.WebSearchResult, topics []duckTopic, limit int) {
	for _, topic := range topics {
		if len(*results) >= limit {
			return
		}
		if len(topic.Topics) > 0 {
			appendTopics(results, topic.Topics, limit)
			continue
		}
		if strings.TrimSpace(topic.Text) == "" {
			continue
		}
		*results = append(*results, protocol.WebSearchResult{
			Title:   extractTopicTitle(topic.Text),
			URL:     topic.FirstURL,
			Snippet: topic.Text,
		})
	}
}

func extractTopicTitle(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, " - "); idx > 0 {
		return strings.TrimSpace(text[:idx])
	}
	return text
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cacheKey(query string, limit int) string {
	return fmt.Sprintf("%s::%d", strings.ToLower(strings.TrimSpace(query)), limit)
}

func (p *Provider) loadCache(key string) ([]protocol.WebSearchResult, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.cache[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(p.cache, key)
		return nil, false
	}
	return append([]protocol.WebSearchResult(nil), entry.results...), true
}

func (p *Provider) saveCache(key string, results []protocol.WebSearchResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache[key] = cacheEntry{
		results:   append([]protocol.WebSearchResult(nil), results...),
		expiresAt: time.Now().Add(p.ttl),
	}
}
