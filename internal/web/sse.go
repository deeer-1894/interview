package web

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// sseWriter handles Server-Sent Events output
type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// newSSEWriter creates a new SSE writer. Returns nil if streaming is not supported.
func newSSEWriter(w http.ResponseWriter) *sseWriter {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	return &sseWriter{
		w:       w,
		flusher: flusher,
	}
}

// SendChunk sends a content chunk to the client
func (s *sseWriter) SendChunk(chunk string) {
	s.sendEvent("message", map[string]string{
		"type":    "chunk",
		"content": chunk,
	})
}

// SendDone sends the completion event with full content
func (s *sseWriter) SendDone(content string) {
	s.sendEvent("message", map[string]string{
		"type":    "done",
		"content": content,
	})
}

// SendJSON sends a structured SSE event payload.
func (s *sseWriter) SendJSON(eventType string, data any) {
	s.sendEvent(eventType, data)
}

// SendError sends an error event to the client
func (s *sseWriter) SendError(errMsg string) {
	s.sendEvent("error", map[string]string{
		"error": errMsg,
	})
}

// sendEvent sends a single SSE event
func (s *sseWriter) sendEvent(eventType string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventType, jsonData)
	s.flusher.Flush()
}
