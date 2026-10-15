package config_test

import (
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/the127/frogly/internal/config"
)

func TestDefaultsApplyWithoutVariables(t *testing.T) {
	loaded, err := config.Load([]string{"HOME=/home/frog"})
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Log.Level != slog.LevelInfo {
		t.Errorf("level is %v, want info", loaded.Log.Level)
	}

	if loaded.Log.Format != config.LogFormatJSON {
		t.Errorf("format is %q, want json", loaded.Log.Format)
	}
}

func TestVariablesOverrideDefaults(t *testing.T) {
	loaded, err := config.Load([]string{"FROGLY_LOG__LEVEL=debug", "FROGLY_LOG__FORMAT=text"})
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Log.Level != slog.LevelDebug {
		t.Errorf("level is %v, want debug", loaded.Log.Level)
	}

	if loaded.Log.Format != config.LogFormatText {
		t.Errorf("format is %q, want text", loaded.Log.Format)
	}
}

func TestErrorsNameTheVariable(t *testing.T) {
	tests := []struct {
		name     string
		variable string
	}{
		{name: "invalid log level", variable: "FROGLY_LOG__LEVEL=loud"},
		{name: "invalid log format", variable: "FROGLY_LOG__FORMAT=xml"},
		{name: "unknown variable", variable: "FROGLY_LOG__COLOUR=green"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := config.Load([]string{test.variable})
			if !errors.Is(err, config.ErrInvalid) {
				t.Fatalf("error is %v, want an invalid configuration", err)
			}

			name, _, _ := strings.Cut(test.variable, "=")
			if !strings.Contains(err.Error(), name) {
				t.Errorf("error %q doesn't name %s", err, name)
			}
		})
	}
}

func TestErrorsNameEveryInvalidVariable(t *testing.T) {
	_, err := config.Load([]string{"FROGLY_LOG__LEVEL=loud", "FROGLY_LOG__FORMAT=xml"})
	if err == nil {
		t.Fatal("loaded without an error")
	}

	for _, name := range []string{"FROGLY_LOG__LEVEL", "FROGLY_LOG__FORMAT"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q doesn't name %s", err, name)
		}
	}
}
