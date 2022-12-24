package gora

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func (r *Router) waitForGracefulShutdown(srv *http.Server) {
	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	r.Logger.Info().Msg("Shutdown server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		r.Logger.Debug().Msgf("Server shutdown error: %v", err)
	} else {
		r.Logger.Debug().Msg("Server shutdown gracefully")
	}
}

func (r *Router) Run(addr string) {
	srv := &http.Server{Addr: addr,
		Handler:        r,
		MaxHeaderBytes: 1 << 20,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.Logger.Info().Msgf("listen: %s", err)
		}
	}()

	r.waitForGracefulShutdown(srv)
}

func (r *Router) RunTLS(addr string, certFile, keyFile string) {
	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		MaxHeaderBytes: 1 << 20,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second}

	go func() {
		if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			r.Logger.Info().Msgf("listen: %s", err)
		}
	}()

	r.waitForGracefulShutdown(srv)
}
