// Command frogly is a task planner for self-organized teams.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/The127/ioc"

	"github.com/the127/frogly/internal/authn"
	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/postgres"
	"github.com/the127/frogly/internal/db/sqlite"
	"github.com/the127/frogly/internal/logging"
	"github.com/the127/frogly/internal/server"
	"github.com/the127/frogly/internal/version"
	"github.com/the127/frogly/web"
)

// a slow client mustn't hold a connection open forever before it has even
// sent its headers
const readHeaderTimeout = 10 * time.Second

// a keep-alive connection nobody uses only costs memory
const idleTimeout = 60 * time.Second

// the default HTTP client has no timeout, so an issuer that accepts the
// connection but never answers would hang the start
const discoveryTimeout = 10 * time.Second

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

	discoveryCtx, cancelDiscovery := context.WithTimeout(ctx, discoveryTimeout)
	verifier, err := authn.NewVerifier(discoveryCtx, cfg.OIDC.Issuer, cfg.OIDC.ClientID)

	cancelDiscovery()

	if err != nil {
		return err
	}

	database, err := openDatabase(cfg.Database)
	if err != nil {
		return err
	}

	defer func() {
		err := database.Close()
		if err != nil {
			logger.Error("closing the database failed", "error", err)
		}
	}()

	err = database.Migrate(ctx)
	if err != nil {
		return err
	}

	dependencies := ioc.NewDependencyCollection()
	ioc.RegisterSingleton(dependencies, func(*ioc.DependencyProvider) db.Database {
		return database
	})
	ioc.RegisterScoped(dependencies, func(services *ioc.DependencyProvider) db.Context {
		return ioc.GetDependency[db.Database](services).NewContext()
	})

	httpServer := &http.Server{
		Handler:           server.New(cfg.OIDC, verifier, logger, web.Files(), dependencies.BuildProvider()),
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// binding before serving makes a taken port fail the start, and logs the
	// real address when the config asks for port 0
	listener, err := net.Listen("tcp", cfg.Server.ListenAddress)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", cfg.Server.ListenAddress, err)
	}

	logger.Info("listening", "address", listener.Addr().String())

	served := make(chan error, 1)

	go func() {
		served <- httpServer.Serve(listener)
	}()

	select {
	case err := <-served:
		return fmt.Errorf("serving: %w", err)

	case <-ctx.Done():
	}

	// a second Ctrl-C then stops frogly at once, without waiting
	stop()

	logger.Info("stopping, waiting for running requests")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err = httpServer.Shutdown(shutdownCtx)
	if err != nil {
		return fmt.Errorf("stopping: %w", err)
	}

	return nil
}

func openDatabase(settings config.Database) (db.Database, error) {
	switch settings.Driver {
	case config.DriverPostgres:
		return postgres.Open(string(settings.Postgres.URL))

	default:
		return sqlite.Open(settings.SQLite.Path)
	}
}
