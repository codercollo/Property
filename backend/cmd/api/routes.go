package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Defines and returns the application's route mappings
func (app *application) routes() http.Handler {
	router := httprouter.New()

	// Custom 404 & 405 handlers
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	// =============================================================================
	// HEALTHCHECK
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	// =============================================================================
	// PROPERTIES ENDPOINTS
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/properties", app.listPropertiesHandler)
	router.HandlerFunc(http.MethodPost, "/v1/properties", app.requirePermission("properties:write", app.createPropertyHandler))

	// Static routes BEFORE wildcards
	router.HandlerFunc(http.MethodGet, "/v1/popular-properties", app.listMostFavouritedPropertiesHandler)

	// Advanced property search
	router.HandlerFunc(http.MethodGet, "/v1/property-search", app.requirePermission("properties:read", app.advancedPropertySearchHandler))
	router.HandlerFunc(http.MethodGet, "/v1/property-filters", app.requirePermission("properties:read", app.getPropertyFiltersHandler))

	// =============================================================================
	// PROPERTY OPERATIONS (using /v1/property/:id to avoid conflicts)
	// =============================================================================
	// Longer paths first
	router.HandlerFunc(http.MethodPost, "/v1/property/:id/feature-payment", app.requireAuthenticatedUser(app.createFeaturePaymentHandler))
	router.HandlerFunc(http.MethodPost, "/v1/property/:id/feature", app.requirePermission("properties:feature", app.featurePropertyHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/property/:id/feature", app.requirePermission("properties:feature", app.unfeaturePropertyHandler))

	router.HandlerFunc(http.MethodGet, "/v1/property/:id/favourite-count", app.getPropertyFavouriteCountHandler)

	router.HandlerFunc(http.MethodPost, "/v1/property/:id/media", app.requirePermission("properties:write", app.uploadPropertyMediaHandler))
	router.HandlerFunc(http.MethodGet, "/v1/property/:id/media", app.requirePermission("properties:read", app.listPropertyMediaHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/property/:id/media", app.requirePermission("properties:write", app.updatePropertyMediaHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/property/:id/media", app.requirePermission("properties:write", app.deletePropertyMediaHandler))

	router.HandlerFunc(http.MethodPost, "/v1/property/:id/inquiries", app.requireAuthenticatedUser(app.createInquiryHandler))
	router.HandlerFunc(http.MethodPost, "/v1/property/:id/schedule", app.requireAuthenticatedUser(app.createScheduleHandler))

	router.HandlerFunc(http.MethodGet, "/v1/property/:id/reviews", app.requirePermission("reviews:read", app.listReviewsForPropertyHandler))
	router.HandlerFunc(http.MethodPost, "/v1/property/:id/reviews", app.requirePermission("reviews:write", app.createReviewHandler))

	// Base property routes (AFTER all sub-routes)
	router.HandlerFunc(http.MethodGet, "/v1/property/:id", app.showPropertyHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/property/:id", app.requirePermission("properties:write", app.updatePropertyHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/property/:id", app.requirePermission("properties:delete", app.deletePropertyHandler))

	// =============================================================================
	// REVIEWS
	// =============================================================================
	router.HandlerFunc(http.MethodGet, "/v1/reviews/pending", app.requirePermission("reviews:moderate", app.listPendingReviewsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/reviews/approve/:id", app.requirePermission("reviews:moderate", app.approveReviewHandler))
	router.HandlerFunc(http.MethodPost, "/v1/reviews/reject/:id", app.requirePermission("reviews:moderate", app.rejectReviewHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/reviews/delete/:id", app.requireAuthenticatedUser(app.deleteReviewHandler))

	// =============================================================================
	// USER ROUTES
	// =============================================================================
	// User favourites - static routes first
	router.HandlerFunc(http.MethodGet, "/v1/users/me/favourites/stats", app.requireAuthenticatedUser(app.getUserFavouriteStatsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/me/favourites", app.requireAuthenticatedUser(app.listUserFavouritesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/me/favourite/:id/status", app.requireAuthenticatedUser(app.checkFavouriteStatusHandler))
	router.HandlerFunc(http.MethodPost, "/v1/users/me/favourite/:id", app.requireAuthenticatedUser(app.addFavouriteHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/users/me/favourite/:id", app.requireAuthenticatedUser(app.removeFavouriteHandler))

	// User profile photo
	router.HandlerFunc(http.MethodPost, "/v1/users/me/photo", app.requireAuthenticatedUser(app.uploadProfilePhotoHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/me/photo", app.requireAuthenticatedUser(app.getProfilePhotoHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/users/me/photo", app.requireAuthenticatedUser(app.deleteProfilePhotoHandler))

	// User schedules (viewings and schedules are the same thing)
	router.HandlerFunc(http.MethodGet, "/v1/users/me/schedules", app.requireAuthenticatedUser(app.listUserSchedulesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/me/schedules/:id", app.requireAuthenticatedUser(app.getUserScheduleHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/users/me/schedules/:id", app.requireAuthenticatedUser(app.rescheduleUserScheduleHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/users/me/schedules/:id", app.requireAuthenticatedUser(app.cancelUserScheduleHandler))

	// User inquiries
	router.HandlerFunc(http.MethodGet, "/v1/users/me/inquiries", app.requireAuthenticatedUser(app.listUserInquiriesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/me/inquiries/:id", app.requireAuthenticatedUser(app.getUserInquiryHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/inquiries/:id", app.requireAuthenticatedUser(app.deleteInquiryHandler))

	// User authentication & registration
	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/password", app.updateUserPasswordHandler)

	// =============================================================================
	// TOKEN MANAGEMENT
	// =============================================================================
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/activation", app.createActivationTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/password-reset", app.createPasswordResetTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/revoke", app.requireAuthenticatedUser(app.revokeAuthenticationTokenHandler))
	router.HandlerFunc(http.MethodPost, "/v1/tokens/revoke-all", app.requireAuthenticatedUser(app.revokeAllTokensHandler))

	// =============================================================================
	// PAYMENT ROUTES
	// =============================================================================
	// Static payment routes BEFORE wildcard routes
	router.HandlerFunc(http.MethodPost, "/v1/payments/mpesa/callback", app.mpesaCallbackHandler)

	// General payment routes
	router.HandlerFunc(http.MethodPost, "/v1/payments", app.requireAuthenticatedUser(app.createPaymentHandler))

	// Payment status with wildcard (AFTER static routes)
	router.HandlerFunc(http.MethodGet, "/v1/payments/:id/status", app.requireAuthenticatedUser(app.queryPaymentStatusHandler))

	// =============================================================================
	// AGENT ROUTES
	// =============================================================================
	// Agent inquiries - static routes first
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/inquiry-stats", app.requireAuthenticatedUser(app.getAgentInquiryStatsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/inquiries", app.requireAuthenticatedUser(app.listAgentInquiriesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/inquiries/:id", app.requireAuthenticatedUser(app.getAgentInquiryHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/agents/me/inquiries/:id", app.requireAuthenticatedUser(app.updateInquiryHandler))

	// Agent schedules - static routes first
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/schedule-stats", app.requireAuthenticatedUser(app.getAgentScheduleStatsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/schedules", app.requireAuthenticatedUser(app.listAgentSchedulesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/schedules/:id", app.requireAuthenticatedUser(app.getAgentScheduleHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/agents/me/schedules/:id", app.requireAuthenticatedUser(app.updateAgentScheduleStatusHandler))

	// Agent profile
	router.HandlerFunc(http.MethodGet, "/v1/agents/me", app.requireAuthenticatedUser(app.getAgentProfileHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/agents/me", app.requireAuthenticatedUser(app.updateAgentProfileHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/agents/me", app.requireAuthenticatedUser(app.deleteAgentAccountHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/agents/me/password", app.requireAuthenticatedUser(app.changeAgentPasswordHandler))

	// Agent profile photo
	router.HandlerFunc(http.MethodPost, "/v1/agents/me/photo", app.requireAuthenticatedUser(app.uploadAgentProfilePhotoHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/photo", app.requireAuthenticatedUser(app.getAgentProfilePhotoHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/agents/me/photo", app.requireAuthenticatedUser(app.deleteAgentProfilePhotoHandler))

	// Agent properties - static routes first
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/property-stats", app.requireAuthenticatedUser(app.getAgentPropertyStatsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/properties", app.requireAuthenticatedUser(app.listAgentPropertiesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/properties/:id", app.requireAuthenticatedUser(app.getAgentPropertyHandler))

	// Agent reviews - static routes first
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/reviews/pending", app.requireAuthenticatedUser(app.listAgentPendingReviewsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/reviews", app.requireAuthenticatedUser(app.listAgentReviewsHandler))

	// Agent payments - static routes first
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/payments", app.requireAuthenticatedUser(app.listPaymentHistoryHandler))
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/payments/:id", app.requireAuthenticatedUser(app.getPaymentStatusHandler))

	// Agent dashboard
	router.HandlerFunc(http.MethodGet, "/v1/agents/me/stats", app.requireAuthenticatedUser(app.getAgentDashboardStatsHandler))

	// =============================================================================
	// ADMIN ROUTES
	// =============================================================================
	// Admin profile
	router.HandlerFunc(http.MethodGet, "/v1/admin/me", app.requireAdminRole(app.getAdminProfileHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/admin/me", app.requireAdminRole(app.updateAdminProfileHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/admin/me/password", app.requireAdminRole(app.changeAdminPasswordHandler))

	// Admin user management
	router.HandlerFunc(http.MethodGet, "/v1/admin/users", app.requireAdminRole(app.listAllUsersHandler))
	router.HandlerFunc(http.MethodGet, "/v1/admin/users/:id", app.requireAdminRole(app.viewUserHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/admin/users/:id", app.requireAdminRole(app.updateUserHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/admin/users/:id/role", app.requirePermission("users:manage", app.updateUserRoleHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/admin/users/:id", app.requireAdminRole(app.deleteUserHandler))

	// Admin agent management
	router.HandlerFunc(http.MethodGet, "/v1/admin/agents", app.requireAdminRole(app.listAllAgentsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/admin/agents/:id/verify", app.requireAdminRole(app.approveAgentVerificationHandler))
	router.HandlerFunc(http.MethodPost, "/v1/admin/agents/:id/reject", app.requireAdminRole(app.rejectAgentVerificationHandler))
	router.HandlerFunc(http.MethodPost, "/v1/admin/agents/:id/suspend", app.requireAdminRole(app.suspendAgentHandler))
	router.HandlerFunc(http.MethodPost, "/v1/admin/agents/:id/activate", app.requireAdminRole(app.activateAgentHandler))

	// Admin property management
	router.HandlerFunc(http.MethodGet, "/v1/admin/properties", app.requireAdminRole(app.listAllPropertiesHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/admin/properties/:id", app.requireAdminRole(app.adminDeletePropertyHandler))

	// Admin statistics - longer path first
	router.HandlerFunc(http.MethodGet, "/v1/admin/stats/growth", app.requireAdminRole(app.getGrowthMetricsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/admin/stats", app.requireAdminRole(app.getPlatformStatsHandler))

	// =============================================================================
	// DEBUG/METRICS
	// =============================================================================
	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	// Serve static files (profile photos)
	router.ServeFiles("/uploads/*filepath", http.Dir("./uploads"))

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}
