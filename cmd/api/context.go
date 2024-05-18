package main

import (
	"context"
	"net/http"

	"github.com/yousifsabah0/blackbox/internal/data"
)

type contextKey string

const (
	userCtxKey = contextKey("user")
)

func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userCtxKey, user)
	return r.WithContext(ctx)
}

func (app *application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userCtxKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}

	return user
}
