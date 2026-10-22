// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// addConfigFlag registers a --config/-c flag on the given persistent flag set.
// Config loading itself happens in loadConfig during PersistentPreRunE, so each
// app uses its own isolated viper instance with no global side effects.
func (a *App) addConfigFlag(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.StringVarP(
		&a.cfgFile, "config", "c", "",
		"Read configuration from specified FILE, support JSON, TOML, YAML, HCL, or Java properties formats.",
	)
}

// loadConfig wires up and reads the configuration into the app's isolated
// viper instance. It is called during PersistentPreRunE, after flags are
// parsed, so flag values (including --config) are available.
func (a *App) loadConfig() error {
	if a.noConfig {
		return nil
	}

	if err := a.loadConfigInto(a.v); err != nil {
		return err
	}

	slog.Info("using config file", "path", a.v.ConfigFileUsed())

	return nil
}

// loadConfigInto applies the app's file, environment and search-path settings to
// the given viper instance.
//
// It is parameterized rather than reading a.v directly so that a config reload
// can build a fresh instance with the same sources instead of re-reading into
// the live one, which is neither concurrency-safe nor able to drop a key the new
// configuration no longer defines.
func (a *App) loadConfigInto(v *viper.Viper) error {
	if a.cfgFile != "" {
		v.SetConfigFile(a.cfgFile)
	} else {
		for _, p := range a.effectiveConfigSearchPaths() {
			v.AddConfigPath(p)
		}
		v.SetConfigType("yaml")
		v.SetConfigName(a.effectiveConfigName())
	}

	v.AutomaticEnv()
	v.SetEnvPrefix(a.envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("config file not found, using defaults and env vars", "searchPaths", a.effectiveConfigSearchPaths())
		} else {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	return nil
}

// printConfig logs the resolved configuration at debug level. It is a no-op (and
// cheap) when debug logging is disabled or config loading is off.
//
// Values whose key names a credential are replaced rather than printed. This
// dump exists to answer "what did the service actually read, and from where" —
// a question about the shape of the configuration, not about the password in it
// — and debug logging is usually turned on in a shared environment, where the
// resulting line would outlive the incident that motivated it.
func (a *App) printConfig() {
	if a.noConfig {
		return
	}

	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}

	keys := a.v.AllKeys()
	attrs := make([]any, 0, len(keys)*2)
	for _, key := range keys {
		value := any(a.v.Get(key))
		if namesACredential(key) {
			value = "<redacted>"
		}
		attrs = append(attrs, key, value)
	}
	slog.Debug("configuration", attrs...)
}

// credentialKeyParts are the substrings that mark a configuration key as
// carrying a credential. They are deliberately broad: redacting a public key or
// a key name costs a line of debugging, while printing a private one cannot be
// taken back.
var credentialKeyParts = []string{
	"password", "secret", "token", "credential", "api-key", "apikey", "private-key",
	"dsn", "salt", "auth-key",
}

// namesACredential reports whether a configuration key is likely to hold a
// credential. Both the plain and the hyphen/underscore forms are matched, since
// a key arrives spelled the way its source spelled it.
func namesACredential(key string) bool {
	normalized := strings.ToLower(key)
	normalized = strings.NewReplacer("_", "-", ".", "-").Replace(normalized)

	for _, part := range credentialKeyParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	// A bare "key" or a "…-key" suffix is a credential in this platform: auth.key
	// signs tokens and capability-key signs knowledge-base entries.
	return normalized == "key" || strings.HasSuffix(normalized, "-key")
}
