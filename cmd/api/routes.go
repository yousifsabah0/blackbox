package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowed)

	// Health route
	router.HandlerFunc(http.MethodGet, "/api/v1/health", app.handleHealth)

	// Movies routes
	router.HandlerFunc(http.MethodGet, "/api/v1/movies", app.requirePermission("movies:read", app.handleShowAllMovies))
	router.HandlerFunc(http.MethodGet, "/api/v1/movies/:id", app.requirePermission("movies:read", app.handleShowMovie))

	router.HandlerFunc(http.MethodPost, "/api/v1/movies", app.requirePermission("movies:write", app.handleCreateMovie))

	router.HandlerFunc(http.MethodPatch, "/api/v1/movies/:id/edit", app.requirePermission("movies:write", app.handleUpdateMovie))
	router.HandlerFunc(http.MethodDelete, "/api/v1/movies/:id/delete", app.requirePermission("movies:write", app.handleDeleteMovie))

	// Users routes
	router.HandlerFunc(http.MethodPost, "/api/v1/users/register", app.handleUserRegister)

	router.HandlerFunc(http.MethodPut, "/api/v1/users/activate", app.handleActivateUser)

	// Tokens routes
	router.HandlerFunc(http.MethodPost, "/api/v1/tokens/auth", app.handleCreateAuthenticationTokenHandler)

	router.HandlerFunc(http.MethodPost, "/api/v1/tokens/password-reset/request", app.handleRequestPasswordReset)

	router.HandlerFunc(http.MethodPut, "/api/v1/tokens/password-reset", app.handleUpdatePassword)

	return app.recoverPanic(app.rateLimit(app.authenticate(router)))
}
