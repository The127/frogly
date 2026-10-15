// Package logging builds frogly's structured logger from the configuration
// and carries it in a context.Context. It uses log/slog as it is. Where and
// how logs are written changes through the slog.Handler, not through an
// interface of our own.
package logging

import (
	"context"
	"io"
	"log/slog"

	"github.com/the127/frogly/internal/config"
)

// New builds the logger the configuration asks for, writing to out.
func New(log config.Log, out io.Writer) *slog.Logger {
	options := &slog.HandlerOptions{
		Level: log.Level,
	}

	var handler slog.Handler = slog.NewJSONHandler(out, options)
	if log.Format == config.LogFormatText {
		handler = slog.NewTextHandler(out, options)
	}

	return slog.New(handler)
}

type contextKey struct{}

// WithLogger returns a context that carries logger, so a request handler
// can add fields like a request ID and pass the logger down.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext is the logger ctx carries, or slog.Default when it carries
// none.
func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(contextKey{}).(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return logger
}
