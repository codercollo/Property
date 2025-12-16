package main

import (
	"strconv"
	"time"
)

// startBackgroundJobs starts all background maintenance jobs
func (app *application) startBackgroundJobs() {
	app.logger.PrintInfo("starting background jobs", nil)

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Run once on startup
		app.cleanupExpiredRevokedTokens()

		for range ticker.C {
			app.cleanupExpiredRevokedTokens()
		}
	}()
}

// cleanupExpiredRevokedTokens removes expired revoked tokens from the database
func (app *application) cleanupExpiredRevokedTokens() {
	app.logger.PrintInfo("starting cleanup of expired revoked tokens", nil)

	count, err := app.models.RevokedTokens.DeleteExpired()
	if err != nil {
		app.logger.PrintError(err, map[string]string{
			"job": "cleanup_expired_revoked_tokens",
		})
		return
	}

	app.logger.PrintInfo("cleanup complete", map[string]string{
		"job":            "cleanup_expired_revoked_tokens",
		"tokens_removed": strconv.FormatInt(count, 10),
	})
}
