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

	//Register HEALTHCHECK endpoints
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	//Register PROPERTIES endpoints
	router.HandlerFunc(http.MethodGet, "/v1/properties", app.requirePermission("properties:read", app.listPropertiesHandler))
	router.HandlerFunc(http.MethodPost, "/v1/properties", app.requirePermission("properties:write", app.createPropertyHandler))
	router.HandlerFunc(http.MethodGet, "/v1/properties/:id", app.requirePermission("properties:read", app.showPropertyHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/properties/:id", app.requirePermission("properties:write", app.updatePropertyHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/properties/:id", app.requirePermission("properties:delete", app.deletePropertyHandler))
	router.HandlerFunc(http.MethodPost, "/v1/properties/:id/feature", app.requirePermission("properties:feature", app.featurePropertyHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/properties/:id/feature", app.requirePermission("properties:feature", app.unfeaturePropertyHandler))

	//Register REVIEWS endpoints
	router.HandlerFunc(http.MethodGet, "/v1/properties/:id/reviews", app.requirePermission("reviews:read", app.listReviewsForPropertyHandler))
	router.HandlerFunc(http.MethodPost, "/v1/properties/:id/reviews", app.requirePermission("reviews:write", app.createReviewHandler))
	router.HandlerFunc(http.MethodGet, "/v1/reviews/pending", app.requirePermission("reviews:moderate", app.listPendingReviewsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/reviews/:id/approve", app.requirePermission("reviews:moderate", app.approveReviewHandler))
	router.HandlerFunc(http.MethodPost, "/v1/reviews/:id/reject", app.requirePermission("reviews:moderate", app.rejectReviewHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/reviews/:id", app.requireAuthenticatedUser(app.deleteReviewHandler))

	//Register USER & Authentication endpoints
	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/users/:id/role", app.requirePermission("users:manage", app.updateUserRoleHandler))
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	//Return configures router
	return app.recoverPanic(app.rateLimit(app.authenticate(router)))

}
