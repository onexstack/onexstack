// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const configFlagName = "config"

// addConfigFlag adds a --config/-c flag and wires up viper-based config loading
// via cobra.OnInitialize. The cfgFile variable is captured via closure — no
// global state. Each app gets its own isolated config loading path.
func (a *App) addConfigFlag(cmd *cobra.Command, fs *pflag.FlagSet) {
	cfgFile := ""

	fs.StringVarP(&cfgFile, configFlagName, "c", "",
		"Read configuration from specified FILE, support JSON, TOML, YAML, HCL, or Java properties formats.",
	)

	// Determine config name.
	configName := a.name
	if a.configName != "" {
		configName = a.configName
	}

	// Determine env prefix.
	envPrefix := strings.ReplaceAll(strings.ToUpper(a.name), "-", "_")
	if a.envPrefix != "" {
		envPrefix = a.envPrefix
	}

	// Determine search paths.
	searchPaths := a.configSearchPaths
	if len(searchPaths) == 0 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		searchPaths = []string{"."}
		if names := strings.Split(a.name, "-"); len(names) > 1 {
			searchPaths = append(searchPaths,
				filepath.Join(homeDir, "."+names[0]),
				filepath.Join("/etc", names[0]),
			)
		}
	}

	// Wire up config loading via cobra.OnInitialize.
	// This is called lazily when Execute() runs, so cfgFile is correctly
	// populated from the flag before loading begins. The cfgFile variable
	// is captured by closure — each app gets its own isolated path.
	cobra.OnInitialize(func() {
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			for _, p := range searchPaths {
				viper.AddConfigPath(p)
			}
			viper.SetConfigType("yaml")
			viper.SetConfigName(configName)
		}
		viper.AutomaticEnv()
		viper.SetEnvPrefix(envPrefix)
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				slog.Info("config file not found, using defaults and env vars",
					"searchPaths", searchPaths)
			} else {
				slog.Error("failed to parse config file", "err", err)
				os.Exit(1)
			}
		}

		if path := viper.ConfigFileUsed(); path != "" {
			slog.Info("using config file", "path", path)
		}

		if a.watch {
			viper.WatchConfig()
			viper.OnConfigChange(func(e fsnotify.Event) {
				slog.Info("config file changed", "name", e.Name)
			})
		}
	})
}

// printConfig logs all viper configuration keys at debug level.
func (a *App) printConfig() {
	for _, key := range viper.AllKeys() {
		slog.Debug(fmt.Sprintf("CFG: %s=%v", key, viper.Get(key)))
	}
}
