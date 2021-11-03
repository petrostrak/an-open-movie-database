package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

var (
	router *httprouter.Router
)

func init() {
	// Initialize a new httprouter router instance.
	router = httprouter.New()
}

// Update the routes() to return a http.Handler instead of a *httprouter.Router.
func (app *application) routes() http.Handler {

	// Convert the notFoundResponse() helper to a http.Handler using the
	// http.HandlerFunc() adapter, and then set it as the custom error
	// handler for 404 Not Found responses.
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	// Convert the methodNotAllowedResponse() helper to a http.Handler and see
	// it as the custom error handler  for 405 Method Not Allowed responses.
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	// Register the relevant methods, URL patterns and handler functions for our
	// endpoints using the HandlerFunc() method. Note that http.MethodGet and
	// http.MethodPost are constants which equate to the strings "GET" and "POST"
	// respectively.
	//
	// Use the requireActivatedUser() middleware on our /v1/movies** endpoints
	//
	// Movies:
	router.HandlerFunc(http.MethodGet, "/v1/movies", app.requireActivatedUser(app.listMoviesHandler))
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.requireActivatedUser(app.createMovieHandler))
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.requireActivatedUser(app.showMovieHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.requireActivatedUser(app.updateMovieHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.requireActivatedUser(app.deleteMovieHandler))

	// Users:
	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	// Authentication
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	// Wrap the router with the panic recovery middleware.
	//
	// Wrap the router with the rateLimit() middleware.
	//
	// Use the authenticate() middleware on all requests.
	return app.recoverPanic(app.rateLimit(app.authenticate(router)))
}
