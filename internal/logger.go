package internal

import (
	"os"

	"log/slog"

	"go.uber.org/fx"
)

// ProvideLogger sets up the slog configuration
func ProvideLogger() *slog.Logger {
	slogHandlerOpts := &slog.HandlerOptions{}

	if os.Getenv("HELMPER_LOG_LEVEL") == "DEBUG" {
		slogHandlerOpts.Level = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, slogHandlerOpts))

	// Set this logger as the default
	slog.SetDefault(logger)

	// Example log entries
	slog.Info("Application started")
	slog.Debug("Debugging application")

	return logger
}

var LoggerModule = fx.Provide(ProvideLogger)
