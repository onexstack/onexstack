package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// SearchDirs returns the default directories to search for the configuration file.
func SearchDirs(defaultHomeDir string) []string {
	// Get the user's home directory.
	homeDir, err := os.UserHomeDir()
	// If unable to get the user's home directory, print an error message and exit the program.
	cobra.CheckErr(err)
	return []string{filepath.Join(homeDir, defaultHomeDir), "."}
}

// FilePath retrieves the full path to the default configuration file.
func FilePath(defaultHomeDir string, defaultConfigName string) string {
	home, err := os.UserHomeDir()
	// If the user's home directory cannot be retrieved, log an error and return an empty path.
	cobra.CheckErr(err)
	return filepath.Join(home, defaultHomeDir, defaultConfigName)
}
