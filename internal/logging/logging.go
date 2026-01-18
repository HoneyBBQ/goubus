// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package logging

import (
	"context"
	"log/slog"
)

type contextKey struct{}

var loggerKey = contextKey{}

// FromContext returns the logger from the context.
// If no logger is found, it returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}

	return slog.Default()
}

// WithContext returns a new context with the given logger.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// Discard returns a logger that discards all log entries.
func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
