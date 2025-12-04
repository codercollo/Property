package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
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
		// Only carry out the check if rate limiting is enabled.
		if app.config.limiter.enabled {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
			mu.Lock()
			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					// Use the requests-per-second and burst values from the config
					// struct.
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

// authenticate is middleware{AUTHORIZATION} that verifies a Bearer token and adds the user to the request context
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Inform caches that the response depends on the Authorization header
		w.Header().Add("Vary", "Authorization")

		//Get Authorization header (empty if missing)
		authorizationHeader := r.Header.Get("Authorization")

		//If no token, treat as anonymous and continue
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		//Expect "Bearer <token>" format
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		//Extract token string
		token := headerParts[1]

		//Validate token format
		v := validator.New()
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		//Look up user for the token (auth scope)
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrUserNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		//Add user to context and continue
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
