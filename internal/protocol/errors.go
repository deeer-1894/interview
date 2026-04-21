package protocol

import "errors"

type ErrorKind string

const (
	ErrorKindUnknown   ErrorKind = "unknown"
	ErrorKindInterview ErrorKind = "interview"
	ErrorKindTool      ErrorKind = "tool"
	ErrorKindModel     ErrorKind = "model"
)

type RunError struct {
	Kind      ErrorKind `json:"kind"`
	Stage     string    `json:"stage,omitempty"`
	Operation string    `json:"operation,omitempty"`
	Message   string    `json:"message"`
	Retryable bool      `json:"retryable,omitempty"`
	Err       error     `json:"-"`
}

func (e *RunError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Kind)
}

func (e *RunError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type ErrorPayload struct {
	Kind      ErrorKind `json:"kind"`
	Stage     string    `json:"stage,omitempty"`
	Operation string    `json:"operation,omitempty"`
	Message   string    `json:"message"`
	Retryable bool      `json:"retryable,omitempty"`
}

func WrapInterviewError(stage, operation string, retryable bool, err error) error {
	return wrapRunError(ErrorKindInterview, stage, operation, retryable, err)
}

func WrapToolError(stage, operation string, retryable bool, err error) error {
	return wrapRunError(ErrorKindTool, stage, operation, retryable, err)
}

func WrapModelError(stage, operation string, retryable bool, err error) error {
	return wrapRunError(ErrorKindModel, stage, operation, retryable, err)
}

func ErrorPayloadFromError(err error) *ErrorPayload {
	if err == nil {
		return nil
	}
	var runErr *RunError
	if errors.As(err, &runErr) {
		return &ErrorPayload{
			Kind:      runErr.Kind,
			Stage:     runErr.Stage,
			Operation: runErr.Operation,
			Message:   runErr.Error(),
			Retryable: runErr.Retryable,
		}
	}
	return &ErrorPayload{
		Kind:    ErrorKindUnknown,
		Message: err.Error(),
	}
}

func wrapRunError(kind ErrorKind, stage, operation string, retryable bool, err error) error {
	if err == nil {
		return nil
	}
	var existing *RunError
	if errors.As(err, &existing) {
		return err
	}
	return &RunError{
		Kind:      kind,
		Stage:     stage,
		Operation: operation,
		Message:   err.Error(),
		Retryable: retryable,
		Err:       err,
	}
}
