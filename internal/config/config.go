// Package config loads frogly's configuration from FROGLY_ environment
// variables. A setting with a sensible default gets it here.
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
	Log Log `koanf:"log"`
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
// error, so a typo can't go unnoticed. Every error names the variable.
func Load(environ []string) (*Config, error) {
	// a variable that isn't set leaves its default in place
	config := Config{
		Log: Log{
			Level:  slog.LevelInfo,
			Format: LogFormatJSON,
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
			key := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(name, envPrefix), envLevel, "."))
			variables[key] = name

			return key, value
		},
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("loading the environment: %w", err)
	}

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
		return nil, fmt.Errorf("%w: %w", ErrInvalid, nameVariables(err, variables))
	}

	if len(metadata.Unused) > 0 {
		unknown := make([]string, 0, len(metadata.Unused))
		for _, key := range metadata.Unused {
			unknown = append(unknown, variables[key])
		}

		slices.Sort(unknown)

		return nil, fmt.Errorf("%w: unknown variables %s", ErrInvalid, strings.Join(unknown, ", "))
	}

	return &config, nil
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
