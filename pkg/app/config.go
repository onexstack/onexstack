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

	if a.cfgFile != "" {
		a.v.SetConfigFile(a.cfgFile)
	} else {
		for _, p := range a.effectiveConfigSearchPaths() {
			a.v.AddConfigPath(p)
		}
		a.v.SetConfigType("yaml")
		a.v.SetConfigName(a.effectiveConfigName())
	}

	a.v.AutomaticEnv()
	a.v.SetEnvPrefix(a.envPrefix)
	a.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := a.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("config file not found, using defaults and env vars", "searchPaths", a.effectiveConfigSearchPaths())
		} else {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	slog.Info("using config file", "path", a.v.ConfigFileUsed())

	return nil
}

// printConfig logs all viper configuration keys at debug level. It is a no-op
// (and cheap) when debug logging is disabled.
func (a *App) printConfig() {
	if !a.noConfig {
		return
	}

	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}

	keys := a.v.AllKeys()
	attrs := make([]any, 0, len(keys)*2)
	for _, key := range keys {
		attrs = append(attrs, key, a.v.Get(key))
	}
	slog.Debug("configuration", attrs...)
}
