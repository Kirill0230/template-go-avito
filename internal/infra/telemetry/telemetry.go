package telemetry

import (
	"fmt"
	"log/slog"
	"os"
)

func NewLogger(serviceName, levelText string) (*slog.Logger, error) {
	var level slog.Level

	if err := level.UnmarshalText([]byte(levelText)); err != nil {
		return nil, fmt.Errorf(
			"parse log level %q: %w",
			levelText,
			err,
		)
	}

	handler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: level,
		},
	)

	logger := slog.New(handler).With(
		slog.String("service", serviceName),
	)

	return logger, nil
}
