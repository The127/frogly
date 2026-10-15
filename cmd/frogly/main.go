// Command frogly is a task planner for self-organized teams.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/the127/frogly/internal/authn"
	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/logging"
	"github.com/the127/frogly/internal/server"
	"github.com/the127/frogly/internal/version"
)

// a slow client mustn't hold a connection open forever before it has even
// sent its headers
const readHeaderTimeout = 10 * time.Second

// long enough for running requests to finish, short enough that a
// container runtime doesn't kill frogly first
const shutdownTimeout = 10 * time.Second

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "frogly:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Environ())
	if err != nil {
		return err
	}

	logger := logging.New(cfg.Log, os.Stderr)
	slog.SetDefault(logger)

	logger.Info("starting frogly", "version", version.Get())
	logger.Debug("configuration loaded", "log_level", cfg.Log.Level, "log_format", cfg.Log.Format)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	verifier, err := authn.NewVerifier(ctx, cfg.OIDC.Issuer, cfg.OIDC.ClientID)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              cfg.Server.ListenAddress,
		Handler:           server.New(cfg.OIDC, verifier, logger),
		ReadHeaderTimeout: readHeaderTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	served := make(chan error, 1)

	go func() {
		served <- httpServer.ListenAndServe()
	}()

	logger.Info("listening", "address", cfg.Server.ListenAddress)

	select {
	case err := <-served:
		return fmt.Errorf("serving: %w", err)

	case <-ctx.Done():
	}

	logger.Info("stopping, waiting for running requests")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err = httpServer.Shutdown(shutdownCtx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("stopping: %w", err)
	}

	return nil
}
