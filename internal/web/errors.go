package web

import "errors"

// Sentinel errors for the web package
var (
	ErrSessionNotFound    = errors.New("session not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrStreamingNotSupported = errors.New("streaming not supported")
	ErrMethodNotAllowed   = errors.New("method not allowed")
)
