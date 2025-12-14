package main

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/felixge/httpsnoop"
	"github.com/pascaldekloe/jwt"
	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
)

// recoverPanic is middleware that recovers from any panics, logs the error,
// and returns a 500 status error
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Recover from panics to prevent server crash
		defer func() {
			if err := recover(); err != nil {
				//Close the connection after responding
				w.Header().Set("Connection", "close")
				//Log the error and send a 500 response
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// rateLimit limits requests rate using a token bucket
func (app *application) rateLimit(next http.Handler) http.Handler {
	// Client holds limiter and last seen time
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client) // Map of IPs to clients
	)

	// Cleanup old clients every minute
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.config.limiter.enabled {
			// Use the realip.FromRequest() function to get the client's real IP address.
			ip := realip.FromRequest(r)
			mu.Lock()
			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps),
						app.config.limiter.burst),
				}
			}
			clients[ip].lastSeen = time.Now()
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}
			mu.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}

// / Middleware to authenticate requests using a JWT token.
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set Vary header for caching proxies
		w.Header().Add("Vary", "Authorization")

		// Get Authorization header
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// Parse and validate Bearer token format
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}
		token := headerParts[1]

		// Verify JWT signature and extract claims
		claims, err := jwt.HMACCheck([]byte(token), []byte(app.config.jwt.secret))
		if err != nil {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Validate token timing and metadata
		if !claims.Valid(time.Now()) ||
			claims.Issuer != "propertyown.api" ||
			!claims.AcceptAudience("propertyown.api") {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Extract user ID from claims
		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// Lookup user record
		user, err := app.models.Users.Get(userID)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrUserNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		// Add user to request context and continue
		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

// requireAuthenticatedUser used to block anonymous users
func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Get user from context
		user := app.contextGetUser(r)

		//Block anonymous users
		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		//Proceed to the next handler
		next.ServeHTTP(w, r)
	})
}

// requireActivatedUser allows only authenticated and activated users
func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	//Inner handler that checks user activation before calling the nex handler
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		//Block inactive users
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})

	//Run activation check after ensuring authentication
	return app.requireAuthenticatedUser(fn)
}

// requirePermission ensures the user has the specified permission
func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		//Get user from context
		user := app.contextGetUser(r)

		//Fetch user's permissions
		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		//Deny access if the user lacks the permission
		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}

		//User has permission - proceeed
		next.ServeHTTP(w, r)
	}

	//Ensure user is activated before checking permissions
	return app.requireActivatedUser(fn)
}

// requireAgentRole ensures the user is an agent
func (app *application) requireAgentRole(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		if user.Role != "agent" {
			app.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireAdminRole ensures the user is an admin
func (app *application) requireAdminRole(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		if user.Role != "admin" {
			app.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireAgentOrAdmin ensures the user is either an agent or admin
func (app *application) requireAgentOrAdmin(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		if user.Role != "agent" && user.Role != "admin" {
			app.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireAdminOrSelf allows admins or the user themselves to access
func (app *application) requireAdminOrSelf(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		// Extract user ID from URL if present
		id, err := app.readIDParam(r)
		if err == nil {
			// If user is admin or accessing their own resource
			if user.Role == "admin" || user.ID == id {
				next.ServeHTTP(w, r)
				return
			}
		} else if user.Role == "admin" {
			// Admin can access even without specific ID
			next.ServeHTTP(w, r)
			return
		}

		app.notPermittedResponse(w, r)
	})
}

// Middleware to handle CORS
func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")

		w.Header().Add("Vary", "Access-Control-Request-Method")
		origin := r.Header.Get("Origin")

		if origin != "" && len(app.config.cors.trustedOrigins) != 0 {
			for i := range app.config.cors.trustedOrigins {
				if origin == app.config.cors.trustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
						w.WriteHeader(http.StatusOK)
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// Metrics middleware collects request count, response count, status codes, and processing time.
func (app *application) metrics(next http.Handler) http.Handler {
	// Global metrics counters
	totalRequestsReceived := expvar.NewInt("total_requests_received")
	totalResponsesSent := expvar.NewInt("total_responses_sent")
	totalProcessingTimeMicroseconds := expvar.NewInt("total_processing_time_μs")

	// Response status code metrics
	totalResponsesSentByStatus := expvar.NewMap("total_responses_sent_by_status")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track incoming request
		totalRequestsReceived.Add(1)

		// Capture request metrics while executing the next handler
		metrics := httpsnoop.CaptureMetrics(next, w, r)

		// Track response count
		totalResponsesSent.Add(1)

		// Track cumulative request processing time
		totalProcessingTimeMicroseconds.Add(metrics.Duration.Microseconds())

		// Track response status codes
		totalResponsesSentByStatus.Add(strconv.Itoa(metrics.Code), 1)
	})
}
