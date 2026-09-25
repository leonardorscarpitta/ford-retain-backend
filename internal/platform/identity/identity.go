// Package identity carries the authenticated user's ID through a request
// context, so modules can scope data without depending on the auth module.
package identity

import "context"

type key struct{}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key{}, id)
}

// UserID returns the authenticated user's ID, or "" if there is none.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(key{}).(string)
	return id
}
