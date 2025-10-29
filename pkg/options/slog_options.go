package options

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

var _ IOptions = (*SlogOptions)(nil)

// SlogOptions contains configuration items related to slog.
type SlogOptions struct {
	// Level specifies the minimum log level to output.
	// Possible values: debug, info, warn, error
	Level string `json:"level,omitempty" mapstructure:"level"`
	// AddSource adds source code position (file:line) to log records
	AddSource bool `json:"add-source,omitempty" mapstructure:"add-source"`
	// Format specifies the structure of log messages.
	// Possible values: json, text
	Format string `json:"format,omitempty" mapstructure:"format"`
	// TimeFormat specifies the time format for text output.
	// Uses Go time format layout. Empty means RFC3339.
	TimeFormat string `json:"time-format,omitempty" mapstructure:"time-format"`
	// Output specifies where to write logs.
	// Possible values: stdout, stderr, or file path
	Output string `json:"output,omitempty" mapstructure:"output"`
}

// NewSlogOptions creates an Options object with default parameters.
func NewSlogOptions() *SlogOptions {
	return &SlogOptions{
		Level:      "info",
		AddSource:  false,
		Format:     "text",
		TimeFormat: "",
		Output:     "stdout",
	}
}

// Validate verifies flags passed to SlogOptions.
func (o *SlogOptions) Validate() []error {
	var errs []error

	// Validate log level
	switch strings.ToUpper(strings.TrimSpace(o.Level)) {
	case "DEBUG", "INFO", "WARN", "WARNING", "ERROR":
	default:
		errs = append(errs, fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", o.Level))
	}

	// Validate format
	switch o.Format {
	case "json", "text":
	default:
		errs = append(errs, fmt.Errorf("invalid log format: %s (must be json or text)", o.Format))
	}

	// Validate output
	if o.Output != "stdout" && o.Output != "stderr" && o.Output != "" {
		// Check if it's a valid file path (basic validation)
		if !filepath.IsAbs(o.Output) && !strings.Contains(o.Output, "/") {
			errs = append(errs, fmt.Errorf("invalid output path: %s", o.Output))
		}
	}

	return errs
}

// AddFlags adds command line flags for the configuration.
func (o *SlogOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Level, join(prefixes...)+"slog.level", o.Level, "Sets the log level. Permitted levels: debug, info, warn, error.")
	fs.StringVar(&o.Format, join(prefixes...)+"slog.format", o.Format, "Sets the log format. Permitted formats: json, text.")
	fs.BoolVar(&o.AddSource, join(prefixes...)+"slog.add-source", o.AddSource, "Add source file:line to log records.")
	fs.StringVar(&o.TimeFormat, join(prefixes...)+"slog.time-format", o.TimeFormat, ""+
		"Time format for text logs using Go's time layout format. Leave empty for RFC3339. "+
		"Examples: '2006-01-02 15:04:05'")
	fs.StringVar(&o.Output, join(prefixes...)+"slog.output", o.Output, "Log output destination (stdout, stderr, or file path).")
}

// ToSlogLevel converts string level to slog.Level.
func (o *SlogOptions) ToSlogLevel() slog.Level {
	switch strings.ToUpper(strings.TrimSpace(o.Level)) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		// Default to INFO if unknown level
		return slog.LevelInfo
	}
}

// GetWriter returns the appropriate io.Writer based on output configuration.
func (o *SlogOptions) GetWriter() (io.Writer, error) {
	switch o.Output {
	case "stdout", "":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		// File output
		file, err := os.OpenFile(o.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", o.Output, err)
		}
		return file, nil
	}
}

// BuildHandler creates a slog.Handler based on the configuration.
func (o *SlogOptions) BuildHandler() (slog.Handler, error) {
	writer, err := o.GetWriter()
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		Level:     o.ToSlogLevel(),
		AddSource: o.AddSource,
	}

	// Set custom time format for text handler
	if o.Format == "text" && o.TimeFormat != "" {
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format(o.TimeFormat))
			}
			return a
		}
	}

	var handler slog.Handler
	switch o.Format {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	return handler, nil
}

// BuildLogger constructs and returns a configured slog.Logger instance without affecting the global logger.
func (o *SlogOptions) BuildLogger() (*slog.Logger, error) {
	handler, err := o.BuildHandler()
	if err != nil {
		return nil, err
	}
	return slog.New(handler), nil
}

// Apply applies the configuration to the global default slog logger.
func (o *SlogOptions) Apply() error {
	logger, err := o.BuildLogger()
	if err != nil {
		return err
	}
	slog.SetDefault(logger)
	return nil
}
