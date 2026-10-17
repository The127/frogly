// Package config loads frogly's configuration from FROGLY_ environment
// variables. A setting with a sensible default gets it here. A setting
// that can't have one is required.
//
// A variable names its setting in upper case, with __ between the levels:
// FROGLY_LOG__LEVEL sets log.level.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/v2"
)

const (
	envPrefix = "FROGLY_"
	envLevel  = "__"
)

// ErrInvalid is wrapped into every error about a configuration that
// can't be used as given.
var ErrInvalid = errors.New("invalid configuration")

// Config is frogly's configuration.
type Config struct {
	Log      Log      `koanf:"log"`
	Server   Server   `koanf:"server"`
	OIDC     OIDC     `koanf:"oidc"`
	Database Database `koanf:"database"`
}

// Database says where frogly keeps its data. Only the settings of the
// chosen driver matter.
type Database struct {
	Driver   Driver   `koanf:"driver"`
	SQLite   SQLite   `koanf:"sqlite"`
	Postgres Postgres `koanf:"postgres"`
}

// Driver is the database frogly uses.
type Driver string

const (
	// DriverSQLite keeps the data in one file, with nothing else to run.
	DriverSQLite Driver = "sqlite"
	// DriverPostgres keeps the data in a Postgres database.
	DriverPostgres Driver = "postgres"
)

// UnmarshalText accepts sqlite and postgres.
func (driver *Driver) UnmarshalText(text []byte) error {
	switch Driver(text) {
	case DriverSQLite, DriverPostgres:
		*driver = Driver(text)

		return nil

	default:
		return fmt.Errorf("%q is neither sqlite nor postgres", text)
	}
}

// SQLite says where the database file is.
type SQLite struct {
	Path string `koanf:"path"`
}

// Postgres says how to connect, as a URL like
// postgres://user:password@host:5432/database.
type Postgres struct {
	URL Secret `koanf:"url"`
}

// Secret is a setting that mustn't end up in a log, like a URL with a
// password in it.
type Secret string

// LogValue keeps the secret out of slog's output.
func (Secret) LogValue() slog.Value {
	return slog.StringValue("[redacted]")
}

// Server says where frogly listens.
type Server struct {
	ListenAddress string `koanf:"listen_address"`
}

// OIDC says where people log in. frogly accepts the access tokens this
// issuer gives out to this client.
type OIDC struct {
	Issuer   string `koanf:"issuer"`
	ClientID string `koanf:"client_id"`
	Roles    Roles  `koanf:"roles"`
}

// Roles says which of the issuer's roles mean what in frogly. Issuers put
// roles into different claims, so the claim is a setting too. It has to be
// a top-level claim with a list of strings. A nested one like Keycloak's
// realm_access.roles doesn't work yet. frogly reads the roles from the
// access token, so a role taken away stops working once the person's
// current token expires.
type Roles struct {
	// Claim names the claim in the access token that lists a person's
	// roles.
	Claim string `koanf:"claim"`
	// User is the role that lets someone use frogly, exactly as the issuer
	// writes it into the claim, like frogly:frogly-user for Keyline.
	User string `koanf:"user"`
}

// Log says how frogly writes its log.
type Log struct {
	Level  slog.Level `koanf:"level"`
	Format LogFormat  `koanf:"format"`
}

// LogFormat is how each log line is written.
type LogFormat string

const (
	// LogFormatJSON writes one JSON object per line, for a log collector.
	LogFormatJSON LogFormat = "json"
	// LogFormatText writes key=value pairs, easier to read locally.
	LogFormatText LogFormat = "text"
)

// UnmarshalText accepts json and text.
func (format *LogFormat) UnmarshalText(text []byte) error {
	switch LogFormat(text) {
	case LogFormatJSON, LogFormatText:
		*format = LogFormat(text)

		return nil

	default:
		return fmt.Errorf("%q is neither json nor text", text)
	}
}

// Load reads the configuration from environ, the environment as
// os.Environ returns it. A FROGLY_ variable frogly doesn't know is an
// error, so a typo can't go unnoticed. A required variable that is missing
// or empty is an error too. The error names every variable at fault at
// once.
func Load(environ []string) (*Config, error) {
	// mapstructure only overwrites what a variable sets, so everything else
	// keeps these defaults
	config := Config{
		Log: Log{
			Level:  slog.LevelInfo,
			Format: LogFormatJSON,
		},
		Server: Server{
			ListenAddress: ":8080",
		},
		Database: Database{
			Driver: DriverSQLite,
			SQLite: SQLite{Path: "frogly.db"},
		},
	}

	variables := make(map[string]string)
	loader := koanf.New(".")

	err := loader.Load(env.Provider(".", env.Opt{
		Prefix: envPrefix,
		EnvironFunc: func() []string {
			return environ
		},
		TransformFunc: func(name string, value string) (string, any) {
			key := keyOf(name)
			variables[key] = name

			return key, value
		},
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("loading the environment: %w", err)
	}

	var problems []error

	var metadata mapstructure.Metadata

	err = loader.UnmarshalWithConf("", &config, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.TextUnmarshallerHookFunc(),
			),
			Metadata: &metadata,
		},
	})
	if err != nil {
		problems = append(problems, nameVariables(err, variables))
	}

	// checked on the decoded settings, so a default counts as set and only an
	// empty variable takes it away
	if missing := missingKeys(&config); len(missing) > 0 {
		variables := make([]string, 0, len(missing))
		for _, key := range missing {
			variables = append(variables, variableOf(key))
		}

		problems = append(problems, fmt.Errorf("missing or empty variables %s", strings.Join(variables, ", ")))
	}

	if len(metadata.Unused) > 0 {
		unknown := make([]string, 0, len(metadata.Unused))
		for _, key := range metadata.Unused {
			unknown = append(unknown, variables[key])
		}

		slices.Sort(unknown)

		problems = append(problems, fmt.Errorf("unknown variables %s", strings.Join(unknown, ", ")))
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("%w: %w", ErrInvalid, errors.Join(problems...))
	}

	return &config, nil
}

func missingKeys(config *Config) []string {
	var missing []string

	if config.OIDC.Issuer == "" {
		missing = append(missing, "oidc.issuer")
	}

	if config.OIDC.ClientID == "" {
		missing = append(missing, "oidc.client_id")
	}

	if config.OIDC.Roles.Claim == "" {
		missing = append(missing, "oidc.roles.claim")
	}

	if config.OIDC.Roles.User == "" {
		missing = append(missing, "oidc.roles.user")
	}

	switch config.Database.Driver {
	case DriverSQLite:
		if config.Database.SQLite.Path == "" {
			missing = append(missing, "database.sqlite.path")
		}

	case DriverPostgres:
		if config.Database.Postgres.URL == "" {
			missing = append(missing, "database.postgres.url")
		}
	}

	return missing
}

func keyOf(variable string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(variable, envPrefix), envLevel, "."))
}

func variableOf(key string) string {
	return envPrefix + strings.ToUpper(strings.ReplaceAll(key, ".", envLevel))
}

// mapstructure names each setting that failed by its key, which means
// nothing to someone who set a variable. It joins one error per setting
// below a wrapping error.
func nameVariables(err error, variables map[string]string) error {
	var named []error

	var walk func(err error)

	walk = func(err error) {
		// errors.As would skip to the first DecodeError and miss the others,
		// the walk visits every level itself
		if decodeErr, ok := err.(*mapstructure.DecodeError); ok { //nolint:errorlint // see above
			named = append(named, fmt.Errorf("%s: %w", variables[decodeErr.Name()], decodeErr.Unwrap()))

			return
		}

		if joined, ok := err.(interface{ Unwrap() []error }); ok {
			for _, inner := range joined.Unwrap() {
				walk(inner)
			}

			return
		}

		if inner := errors.Unwrap(err); inner != nil {
			walk(inner)
		}
	}

	walk(err)

	if len(named) == 0 {
		return err
	}

	return errors.Join(named...)
}
