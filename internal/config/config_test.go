package config_test

import (
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/config"
)

// without these, every configuration is invalid
var required = []string{
	"FROGLY_OIDC__ISSUER=http://localhost:8082/oidc/frogly",
	"FROGLY_OIDC__CLIENT_ID=frogly",
	"FROGLY_OIDC__ROLES__CLAIM=roles",
	"FROGLY_OIDC__ROLES__USER=frogly:frogly-user",
}

func environment(variables ...string) []string {
	return append(slices.Clone(required), variables...)
}

func without(variable string) []string {
	return slices.DeleteFunc(slices.Clone(required), func(set string) bool {
		return strings.HasPrefix(set, variable+"=")
	})
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
			assert.Equal(t, config.Roles{Claim: "roles", User: "frogly:frogly-user"}, loaded.OIDC.Roles)
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
			environ:  without("FROGLY_OIDC__ISSUER"),
			variable: "FROGLY_OIDC__ISSUER",
		},
		{
			name:     "empty issuer",
			environ:  []string{"FROGLY_OIDC__ISSUER=", "FROGLY_OIDC__CLIENT_ID=frogly"},
			variable: "FROGLY_OIDC__ISSUER",
		},
		{
			name:     "missing client ID",
			environ:  without("FROGLY_OIDC__CLIENT_ID"),
			variable: "FROGLY_OIDC__CLIENT_ID",
		},
		{
			name:     "missing roles claim",
			environ:  without("FROGLY_OIDC__ROLES__CLAIM"),
			variable: "FROGLY_OIDC__ROLES__CLAIM",
		},
		{
			name:     "empty roles claim",
			environ:  append(without("FROGLY_OIDC__ROLES__CLAIM"), "FROGLY_OIDC__ROLES__CLAIM="),
			variable: "FROGLY_OIDC__ROLES__CLAIM",
		},
		{
			name:     "missing user role",
			environ:  without("FROGLY_OIDC__ROLES__USER"),
			variable: "FROGLY_OIDC__ROLES__USER",
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

func TestLoadsTheDatabaseSettings(t *testing.T) {
	tests := []struct {
		name    string
		environ []string
		want    config.Database
	}{
		{
			name:    "SQLite next to frogly by default",
			environ: environment(),
			want:    config.Database{Driver: config.DriverSQLite, SQLite: config.SQLite{Path: "frogly.db"}},
		},
		{
			name: "Postgres",
			environ: environment(
				"FROGLY_DATABASE__DRIVER=postgres",
				"FROGLY_DATABASE__POSTGRES__URL=postgres://frogly@localhost:5432/frogly",
			),
			want: config.Database{
				Driver:   config.DriverPostgres,
				SQLite:   config.SQLite{Path: "frogly.db"},
				Postgres: config.Postgres{URL: "postgres://frogly@localhost:5432/frogly"},
			},
		},
		{
			name: "Postgres ignores an empty SQLite path",
			environ: environment(
				"FROGLY_DATABASE__DRIVER=postgres",
				"FROGLY_DATABASE__POSTGRES__URL=postgres://frogly@localhost:5432/frogly",
				"FROGLY_DATABASE__SQLITE__PATH=",
			),
			want: config.Database{
				Driver:   config.DriverPostgres,
				Postgres: config.Postgres{URL: "postgres://frogly@localhost:5432/frogly"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			loaded, err := config.Load(test.environ)

			// assert
			require.NoError(t, err)
			assert.Equal(t, test.want, loaded.Database)
		})
	}
}

func TestRefusesDatabaseSettingsItCantUse(t *testing.T) {
	tests := []struct {
		name     string
		environ  []string
		variable string
	}{
		{
			name:     "unknown driver",
			environ:  environment("FROGLY_DATABASE__DRIVER=mysql"),
			variable: "FROGLY_DATABASE__DRIVER",
		},
		{
			name:     "Postgres without a URL",
			environ:  environment("FROGLY_DATABASE__DRIVER=postgres"),
			variable: "FROGLY_DATABASE__POSTGRES__URL",
		},
		{
			name:     "SQLite without a path",
			environ:  environment("FROGLY_DATABASE__SQLITE__PATH="),
			variable: "FROGLY_DATABASE__SQLITE__PATH",
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

func TestAnUnknownDriverIsTheOnlyDatabaseProblem(t *testing.T) {
	// arrange
	environ := environment("FROGLY_DATABASE__DRIVER=mysql")

	// act
	_, err := config.Load(environ)

	// assert
	require.ErrorContains(t, err, "FROGLY_DATABASE__DRIVER")
	assert.NotContains(t, err.Error(), "FROGLY_DATABASE__SQLITE__PATH")
	assert.NotContains(t, err.Error(), "FROGLY_DATABASE__POSTGRES__URL")
}

func TestThePostgresURLStaysOutOfLogs(t *testing.T) {
	// arrange
	url := config.Secret("postgres://frogly:a-password@localhost:5432/frogly")

	// act
	logged := url.LogValue().String()

	// assert
	assert.NotContains(t, logged, "a-password")
}
