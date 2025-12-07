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

	//Register AGENT ACCOUNT management
	router.HandlerFunc(http.MethodGet, "/v1/agents/me", app.requireAuthenticatedUser(app.getAgentProfileHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/agents/me", app.requireAuthenticatedUser(app.updateAgentProfileHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/agents/me", app.requireAuthenticatedUser(app.deleteAgentAccountHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/agents/me/password", app.requireAuthenticatedUser(app.changeAgentPasswordHandler))

	//Register AGENT PROPERTY management
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/properties", app.requireAuthenticatedUser(app.listAgentPropertiesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/properties/:id", app.requireAuthenticatedUser(app.getAgentPropertyHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/property-stats", app.requireAuthenticatedUser(app.getAgentPropertyStatsHandler))

	//Register AGENT REVIEW management
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/reviews", app.requireAuthenticatedUser(app.listAgentReviewsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/reviews/pending", app.requireAuthenticatedUser(app.listAgentPendingReviewsHandler))

	//Register AGENT PAYMENTS & FEATURED listings
	router.HandlerFunc(http.MethodPost, "/v1/properties/:id/feature/payment", app.requireAuthenticatedUser(app.createFeaturePaymentHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/payments", app.requireAuthenticatedUser(app.listPaymentHistoryHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/payments/:id", app.requireAuthenticatedUser(app.getPaymentStatusHandler))

	//Register AGENT DASHBOARD
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/stats", app.requireAuthenticatedUser(app.getAgentDashboardStatsHandler))

	// =============================================================================
	// ADMIN SELF-MANAGEMENT
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/admin/me", app.requireAdminRole(app.getAdminProfileHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/admin/me", app.requireAdminRole(app.updateAdminProfileHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/admin/me/password", app.requireAdminRole(app.changeAdminPasswordHandler))

	// =============================================================================
	// ADMIN USER MANAGEMENT
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/admin/users", app.requireAdminRole(app.listAllUsersHandler))
	router.HandlerFunc(http.MethodGet, "/v1/admin/users/:id", app.requireAdminRole(app.viewUserHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/admin/users/:id", app.requireAdminRole(app.updateUserHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/admin/users/:id", app.requireAdminRole(app.deleteUserHandler))

	// =============================================================================
	// ADMIN AGENT MANAGEMENT
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/admin/agents", app.requireAdminRole(app.listAllAgentsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/admin/agents/:id/verify", app.requireAdminRole(app.approveAgentVerificationHandler))
	router.HandlerFunc(http.MethodPost, "/v1/admin/agents/:id/suspend", app.requireAdminRole(app.suspendAgentHandler))
	router.HandlerFunc(http.MethodPost, "/v1/admin/agents/:id/activate", app.requireAdminRole(app.activateAgentHandler))

	// =============================================================================
	// ADMIN PROPERTY MANAGEMENT
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/admin/properties", app.requireAdminRole(app.listAllPropertiesHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/admin/properties/:id", app.requireAdminRole(app.adminDeletePropertyHandler))

	// =============================================================================
	// ADMIN PLATFORM STATISTICS
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/admin/stats", app.requireAdminRole(app.getPlatformStatsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/admin/stats/growth", app.requireAdminRole(app.getGrowthMetricsHandler))

	//Return configured router with middleware
	return app.recoverPanic(app.rateLimit(app.authenticate(router)))
}
