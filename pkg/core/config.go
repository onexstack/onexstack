package core

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// OnInitialize returns a function that initializes the application configuration using Viper.
// It sets up the configuration file source, environment variable overrides, and handles loading errors.
// This function is typically used with cobra.OnInitialize.
func OnInitialize(cfgFile *string, envPrefix string, searchPaths []string, configName string) func() {
	return func() {
		if cfgFile != nil && *cfgFile != "" {
			// Use the specific config file defined by the flag.
			viper.SetConfigFile(*cfgFile)
		} else {
			// Search for the config file in the provided paths.
			for _, path := range searchPaths {
				viper.AddConfigPath(path)
			}
			viper.SetConfigType("yaml")
			viper.SetConfigName(configName)
		}

		viper.AutomaticEnv()
		viper.SetEnvPrefix(envPrefix)
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// It is acceptable if the config file is missing, as we may rely solely on environment variables.
				slog.Warn("Config file not found; falling back to defaults or environment variables", "err", err)
			} else {
				// If the config file exists but is invalid (e.g., syntax error), the application should fail fast.
				slog.Error("Failed to parse config file", "err", err)
				os.Exit(1)
			}
		}

		if path := viper.ConfigFileUsed(); path != "" {
			slog.Info("Using config file", "path", path)
		} else {
			slog.Info("No config file used")
		}
	}
}
