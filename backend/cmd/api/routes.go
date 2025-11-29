package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Defines and returns the application's route mappings
func (app *application) routes() http.Handler {
	//Initialize/create a new httpprouter router instance
	router := httprouter.New()

	//Set custom 404 handler
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	//Set acustom 405 handler
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	//Register GET and POST endpoints
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodGet, "/v1/properties", app.listPropertiesHandler)
	router.HandlerFunc(http.MethodPost, "/v1/properties", app.createPropertyHandler)
	router.HandlerFunc(http.MethodGet, "/v1/properties/:id", app.showPropertyHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/properties/:id", app.updatePropertyHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/properties/:id", app.deletePropertyHandler)

	//Register USER endpoints
	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)

	//Return configures router
	return app.recoverPanic(app.rateLimit(router))

}
