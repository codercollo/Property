package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Defines and returns the application's route mappings
func (app *application) routes() http.Handler {
	//Initialize/create a new httprouter router instance
	router := httprouter.New()

	//Set custom 404 handler
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	//Set acustom 405 handler
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	//Register HEALTHCHECK endpoints
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	// =============================================================================
	// PROPERTIES ENDPOINTS - List and Create only
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/properties", app.requirePermission("properties:read", app.listPropertiesHandler))
	router.HandlerFunc(http.MethodPost, "/v1/properties", app.requirePermission("properties:write", app.createPropertyHandler))

	// Advanced property search endpoints
	router.HandlerFunc(http.MethodGet, "/v1/property-search", app.requirePermission("properties:read", app.advancedPropertySearchHandler))
	router.HandlerFunc(http.MethodGet, "/v1/property-filters", app.requirePermission("properties:read", app.getPropertyFiltersHandler))

	// =============================================================================
	// PROPERTY OPERATIONS - Separate namespace to avoid ALL conflicts
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/property/:id", app.requirePermission("properties:read", app.showPropertyHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/property/:id", app.requirePermission("properties:write", app.updatePropertyHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/property/:id", app.requirePermission("properties:delete", app.deletePropertyHandler))

	// Popular properties - different path entirely
	router.HandlerFunc(http.MethodGet, "/v1/popular-properties", app.listMostFavouritedPropertiesHandler)

	// =============================================================================
	// PROPERTY FEATURES
	// =============================================================================
	router.HandlerFunc(http.MethodPost, "/v1/property/:id/feature-payment", app.requireAuthenticatedUser(app.createFeaturePaymentHandler))
	router.HandlerFunc(http.MethodPost, "/v1/property/:id/feature", app.requirePermission("properties:feature", app.featurePropertyHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/property/:id/feature", app.requirePermission("properties:feature", app.unfeaturePropertyHandler))

	// =============================================================================
	// PROPERTY MEDIA
	// =============================================================================
	router.HandlerFunc(http.MethodPost, "/v1/property/:id/media", app.requirePermission("properties:write", app.uploadPropertyMediaHandler))
	router.HandlerFunc(http.MethodGet, "/v1/property/:id/media", app.requirePermission("properties:read", app.listPropertyMediaHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/property/:id/media", app.requirePermission("properties:write", app.updatePropertyMediaHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/property/:id/media", app.requirePermission("properties:write", app.deletePropertyMediaHandler))

	// =============================================================================
	// PROPERTY INQUIRIES
	// =============================================================================
	router.HandlerFunc(http.MethodPost, "/v1/property/:id/inquiries", app.requireAuthenticatedUser(app.createInquiryHandler))

	// =============================================================================
	// PROPERTY REVIEWS
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/property/:id/reviews", app.requirePermission("reviews:read", app.listReviewsForPropertyHandler))
	router.HandlerFunc(http.MethodPost, "/v1/property/:id/reviews", app.requirePermission("reviews:write", app.createReviewHandler))

	// =============================================================================
	// REVIEW MANAGEMENT
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/reviews/pending", app.requirePermission("reviews:moderate", app.listPendingReviewsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/reviews/approve/:id", app.requirePermission("reviews:moderate", app.approveReviewHandler))
	router.HandlerFunc(http.MethodPost, "/v1/reviews/reject/:id", app.requirePermission("reviews:moderate", app.rejectReviewHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/reviews/delete/:id", app.requireAuthenticatedUser(app.deleteReviewHandler))

	// =============================================================================
	// FAVOURITES
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/property/:id/favourite-count", app.getPropertyFavouriteCountHandler)

	// =============================================================================
	// USER FAVOURITES / SAVED PROPERTIES
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/users/me/favourites/stats", app.requireAuthenticatedUser(app.getUserFavouriteStatsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/me/favourites", app.requireAuthenticatedUser(app.listUserFavouritesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/me/favourite/:id/status", app.requireAuthenticatedUser(app.checkFavouriteStatusHandler))
	router.HandlerFunc(http.MethodPost, "/v1/users/me/favourite/:id", app.requireAuthenticatedUser(app.addFavouriteHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/users/me/favourite/:id", app.requireAuthenticatedUser(app.removeFavouriteHandler))

	// =============================================================================
	// INQUIRIES MANAGEMENT
	// =============================================================================
	// Agent endpoints
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/inquiry-stats", app.requireAgentRole(app.getAgentInquiryStatsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/inquiries", app.requireAgentRole(app.listAgentInquiriesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/inquiries/:id", app.requireAgentRole(app.getAgentInquiryHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/agents/me/inquiries/:id", app.requireAgentRole(app.updateInquiryHandler))

	// User endpoints
	router.HandlerFunc(http.MethodGet, "/v1/users/me/inquiries", app.requireAuthenticatedUser(app.listUserInquiriesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/me/inquiries/:id", app.requireAuthenticatedUser(app.getUserInquiryHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/inquiries/:id", app.requireAuthenticatedUser(app.deleteInquiryHandler))

	// =============================================================================
	// USER & AUTHENTICATION
	// =============================================================================
	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/users/:id/role", app.requirePermission("users:manage", app.updateUserRoleHandler))
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	// =============================================================================
	// AGENT ACCOUNT MANAGEMENT
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/agents/me", app.requireAuthenticatedUser(app.getAgentProfileHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/agents/me", app.requireAuthenticatedUser(app.updateAgentProfileHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/agents/me", app.requireAuthenticatedUser(app.deleteAgentAccountHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/agents/me/password", app.requireAuthenticatedUser(app.changeAgentPasswordHandler))

	// =============================================================================
	// AGENT PROPERTY MANAGEMENT
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/property-stats", app.requireAuthenticatedUser(app.getAgentPropertyStatsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/properties", app.requireAuthenticatedUser(app.listAgentPropertiesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/properties/:id", app.requireAuthenticatedUser(app.getAgentPropertyHandler))

	// =============================================================================
	// AGENT REVIEW MANAGEMENT
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/reviews/pending", app.requireAuthenticatedUser(app.listAgentPendingReviewsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/reviews", app.requireAuthenticatedUser(app.listAgentReviewsHandler))

	// =============================================================================
	// AGENT PAYMENTS & FEATURED LISTINGS
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/payments", app.requireAuthenticatedUser(app.listPaymentHistoryHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/payments/:id", app.requireAuthenticatedUser(app.getPaymentStatusHandler))

	// =============================================================================
	// AGENT DASHBOARD
	// =============================================================================
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
	router.HandlerFunc(http.MethodGet, "/v1/admin/stats/growth", app.requireAdminRole(app.getGrowthMetricsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/admin/stats", app.requireAdminRole(app.getPlatformStatsHandler))

	//Return configured router with middleware
	return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router))))
}
