package runtime

import (
	"context"

	"mockinterview/internal/interview/session"
)

type turnContextKey struct{}

func WithTurnContext(ctx context.Context, turn session.TurnContext) context.Context {
	return context.WithValue(ctx, turnContextKey{}, turn)
}

func TurnContextFrom(ctx context.Context) (session.TurnContext, bool) {
	turn, ok := ctx.Value(turnContextKey{}).(session.TurnContext)
	return turn, ok
}
