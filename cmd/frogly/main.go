// Command frogly is a task planner for self-organized teams.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/logging"
	"github.com/the127/frogly/internal/version"
)

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

	return nil
}
