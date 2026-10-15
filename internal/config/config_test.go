package config_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/config"
)

func TestLoadsTheLogSettings(t *testing.T) {
	tests := []struct {
		name       string
		environ    []string
		wantLevel  slog.Level
		wantFormat config.LogFormat
	}{
		{
			name:       "defaults without variables",
			environ:    []string{"HOME=/home/frog"},
			wantLevel:  slog.LevelInfo,
			wantFormat: config.LogFormatJSON,
		},
		{
			name:       "variables override the defaults",
			environ:    []string{"FROGLY_LOG__LEVEL=debug", "FROGLY_LOG__FORMAT=text"},
			wantLevel:  slog.LevelDebug,
			wantFormat: config.LogFormatText,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			loaded, err := config.Load(test.environ)

			// assert
			require.NoError(t, err)
			assert.Equal(t, test.wantLevel, loaded.Log.Level)
			assert.Equal(t, test.wantFormat, loaded.Log.Format)
		})
	}
}

func TestErrorsNameTheVariable(t *testing.T) {
	tests := []struct {
		name     string
		variable string
		value    string
	}{
		{name: "invalid log level", variable: "FROGLY_LOG__LEVEL", value: "loud"},
		{name: "invalid log format", variable: "FROGLY_LOG__FORMAT", value: "xml"},
		{name: "unknown variable", variable: "FROGLY_LOG__COLOUR", value: "green"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			environ := []string{test.variable + "=" + test.value}

			// act
			_, err := config.Load(environ)

			// assert
			require.ErrorIs(t, err, config.ErrInvalid)
			require.ErrorContains(t, err, test.variable)
		})
	}
}

func TestErrorsNameEveryInvalidVariable(t *testing.T) {
	// arrange
	environ := []string{"FROGLY_LOG__LEVEL=loud", "FROGLY_LOG__FORMAT=xml"}

	// act
	_, err := config.Load(environ)

	// assert
	require.ErrorIs(t, err, config.ErrInvalid)
	require.ErrorContains(t, err, "FROGLY_LOG__LEVEL")
	require.ErrorContains(t, err, "FROGLY_LOG__FORMAT")
}
