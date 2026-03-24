package ctxkeys

import "context"

type key string

const (
	playerID    key = "playerID"
	playerEmail key = "playerEmail"
)

func WithPlayerID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, playerID, id)
}

func WithPlayerEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, playerEmail, email)
}

func PlayerID(ctx context.Context) string {
	v, _ := ctx.Value(playerID).(string)
	return v
}

func PlayerEmail(ctx context.Context) string {
	v, _ := ctx.Value(playerEmail).(string)
	return v
}
