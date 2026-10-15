package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/logging"
)

type line struct {
	Msg  string `json:"msg"`
	Idea int    `json:"idea"`
}

func readLines(t *testing.T, out *bytes.Buffer) []line {
	t.Helper()

	var lines []line

	decoder := json.NewDecoder(out)
	for decoder.More() {
		var next line

		require.NoError(t, decoder.Decode(&next))

		lines = append(lines, next)
	}

	return lines
}

func TestJSONWritesTheFields(t *testing.T) {
	// arrange
	var out bytes.Buffer

	logger := logging.New(config.Log{Level: slog.LevelInfo, Format: config.LogFormatJSON}, &out)

	// act
	logger.Info("planted", "idea", 42)

	// assert
	assert.Equal(t, []line{{Msg: "planted", Idea: 42}}, readLines(t, &out))
}

func TestTextIsNotJSON(t *testing.T) {
	// arrange
	var out bytes.Buffer

	logger := logging.New(config.Log{Level: slog.LevelInfo, Format: config.LogFormatText}, &out)

	// act
	logger.Info("planted")

	// assert
	assert.False(t, json.Valid(out.Bytes()), "%q is JSON", out.String())
	assert.Contains(t, out.String(), "msg=planted")
}

func TestLevelHidesLowerLines(t *testing.T) {
	// arrange
	var out bytes.Buffer

	logger := logging.New(config.Log{Level: slog.LevelInfo, Format: config.LogFormatJSON}, &out)

	// act
	logger.Debug("hidden")
	logger.Info("shown")

	// assert
	assert.Equal(t, []line{{Msg: "shown"}}, readLines(t, &out))
}

func TestContextCarriesTheLogger(t *testing.T) {
	// arrange
	logger := slog.New(slog.DiscardHandler)
	ctx := logging.WithLogger(t.Context(), logger)

	// act
	got := logging.FromContext(ctx)

	// assert
	assert.Same(t, logger, got)
}

func TestContextWithoutLoggerGivesTheDefault(t *testing.T) {
	// act
	got := logging.FromContext(context.Background())

	// assert
	assert.Same(t, slog.Default(), got)
}
