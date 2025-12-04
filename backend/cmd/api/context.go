package main

import (
	"context"
	"net/http"

	"github.com/codercollo/property/backend/internal/data"
)

// contextKey is a custom type for context keys
type contextKey string

// userContextKey is the key used to store/retrieve the User from the context
const userContextKey = contextKey("user")

// contextSetUser  adds the User to the request context and returns the updated request
func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// contextGetUser retrieves the User to the request context and panics if missing
func (app *application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}
