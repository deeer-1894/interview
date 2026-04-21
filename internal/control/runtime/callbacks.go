package runtime

import (
	"context"
	"time"
)

type Callback interface {
	OnSpanStart(ctx context.Context, span Span)
	OnSpanEnd(ctx context.Context, span Span, err error, duration time.Duration)
}

type Span struct {
	Scope string
	Name  string
}
