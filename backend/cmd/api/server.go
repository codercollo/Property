package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Channel to receive shutdown errors.
	shutdownError := make(chan error)

	go func() {
		// Listen for interrupt/terminate signals.
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		//Log received shutdown signal
		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		// Graceful shutdown with timeout.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		//Shutdown server; report error if any
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		//Log that we're waiting for background goroutines
		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		//Wait for all background tasks to finish, then signal clean shutdown
		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	// //TLS cert and keys files
	// certFile := "tls/cert.pem"
	// keyFile := "tls/key.pem"

	// Ignore expected server-closed error during shutdown.
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Wait for graceful shutdown result.
	err = <-shutdownError
	if err != nil {
		return err
	}

	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
