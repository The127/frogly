package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/logging"
)

func TestJSONWritesOneObjectPerLine(t *testing.T) {
	var out bytes.Buffer

	logging.New(config.Log{Level: slog.LevelInfo, Format: config.LogFormatJSON}, &out).Info("planted", "idea", 42)

	var line map[string]any

	err := json.Unmarshal(out.Bytes(), &line)
	if err != nil {
		t.Fatalf("%q isn't JSON: %v", out.String(), err)
	}

	if line["msg"] != "planted" || line["idea"] != float64(42) {
		t.Errorf("line is %v, want msg planted and idea 42", line)
	}
}

func TestTextWritesKeyValuePairs(t *testing.T) {
	var out bytes.Buffer

	logging.New(config.Log{Level: slog.LevelInfo, Format: config.LogFormatText}, &out).Info("planted", "idea", 42)

	if !strings.Contains(out.String(), "msg=planted idea=42") {
		t.Errorf("line is %q, want msg=planted idea=42", out.String())
	}
}

func TestLevelHidesLowerLines(t *testing.T) {
	var out bytes.Buffer

	logger := logging.New(config.Log{Level: slog.LevelInfo, Format: config.LogFormatText}, &out)
	logger.Debug("hidden")
	logger.Info("shown")

	if strings.Contains(out.String(), "hidden") || !strings.Contains(out.String(), "shown") {
		t.Errorf("log is %q, want only the info line", out.String())
	}
}

func TestContextCarriesTheLogger(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	if logging.FromContext(logging.WithLogger(t.Context(), logger)) != logger {
		t.Error("the context lost the logger")
	}
}

func TestContextWithoutLoggerGivesTheDefault(t *testing.T) {
	if logging.FromContext(context.Background()) != slog.Default() {
		t.Error("an empty context didn't give slog.Default")
	}
}
