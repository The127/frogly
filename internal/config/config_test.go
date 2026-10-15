package config_test

import (
	"log/slog"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/config"
)

// without these, every configuration is invalid
var required = []string{
	"FROGLY_OIDC__ISSUER=http://localhost:8082/oidc/frogly",
	"FROGLY_OIDC__CLIENT_ID=frogly",
}

func environment(variables ...string) []string {
	return append(slices.Clone(required), variables...)
}

func TestLoadsTheLogSettings(t *testing.T) {
	tests := []struct {
		name       string
		environ    []string
		wantLevel  slog.Level
		wantFormat config.LogFormat
	}{
		{
			name:       "defaults without variables",
			environ:    environment("HOME=/home/frog"),
			wantLevel:  slog.LevelInfo,
			wantFormat: config.LogFormatJSON,
		},
		{
			name:       "variables override the defaults",
			environ:    environment("FROGLY_LOG__LEVEL=debug", "FROGLY_LOG__FORMAT=text"),
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
			environ := environment(test.variable + "=" + test.value)

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
	environ := environment("FROGLY_LOG__LEVEL=loud", "FROGLY_LOG__FORMAT=xml")

	// act
	_, err := config.Load(environ)

	// assert
	require.ErrorIs(t, err, config.ErrInvalid)
	require.ErrorContains(t, err, "FROGLY_LOG__LEVEL")
	require.ErrorContains(t, err, "FROGLY_LOG__FORMAT")
}

func TestLoadsTheServerAndOIDCSettings(t *testing.T) {
	tests := []struct {
		name              string
		environ           []string
		wantListenAddress string
	}{
		{
			name:              "default listen address",
			environ:           environment(),
			wantListenAddress: ":8080",
		},
		{
			name:              "variable overrides the listen address",
			environ:           environment("FROGLY_SERVER__LISTEN_ADDRESS=127.0.0.1:9000"),
			wantListenAddress: "127.0.0.1:9000",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			loaded, err := config.Load(test.environ)

			// assert
			require.NoError(t, err)
			assert.Equal(t, test.wantListenAddress, loaded.Server.ListenAddress)
			assert.Equal(t, "http://localhost:8082/oidc/frogly", loaded.OIDC.Issuer)
			assert.Equal(t, "frogly", loaded.OIDC.ClientID)
		})
	}
}

func TestErrorsNameMissingOrEmptyRequiredVariables(t *testing.T) {
	tests := []struct {
		name     string
		environ  []string
		variable string
	}{
		{
			name:     "missing issuer",
			environ:  []string{"FROGLY_OIDC__CLIENT_ID=frogly"},
			variable: "FROGLY_OIDC__ISSUER",
		},
		{
			name:     "empty issuer",
			environ:  []string{"FROGLY_OIDC__ISSUER=", "FROGLY_OIDC__CLIENT_ID=frogly"},
			variable: "FROGLY_OIDC__ISSUER",
		},
		{
			name:     "missing client ID",
			environ:  []string{"FROGLY_OIDC__ISSUER=http://localhost:8082/oidc/frogly"},
			variable: "FROGLY_OIDC__CLIENT_ID",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			_, err := config.Load(test.environ)

			// assert
			require.ErrorIs(t, err, config.ErrInvalid)
			require.ErrorContains(t, err, test.variable)
		})
	}
}

func TestErrorsNameEveryProblemAtOnce(t *testing.T) {
	// arrange
	environ := []string{
		"FROGLY_OIDC__CLIENT_ID=frogly",
		"FROGLY_OIDC__ISUER=http://localhost:8082/oidc/frogly",
		"FROGLY_LOG__LEVEL=loud",
	}

	// act
	_, err := config.Load(environ)

	// assert
	require.ErrorIs(t, err, config.ErrInvalid)
	require.ErrorContains(t, err, "FROGLY_OIDC__ISSUER")
	require.ErrorContains(t, err, "FROGLY_OIDC__ISUER")
	require.ErrorContains(t, err, "FROGLY_LOG__LEVEL")
}
