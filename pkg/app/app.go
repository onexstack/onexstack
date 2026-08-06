// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

// Package app provides a common framework for building CLI applications in the
// OnexStack ecosystem. It wraps Cobra and Viper to provide a standardized
// application lifecycle with signal-aware context propagation, structured
// logging via slog, and optional lifecycle hooks.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "go.uber.org/automaxprocs"

	"github.com/onexstack/onexstack/pkg/options"
	"github.com/onexstack/onexstack/pkg/version"
)

// App is the main structure of a CLI application. It provides a standardized
// lifecycle: config loading → flag parsing → validation → logging init →
// signal-aware RunFunc → graceful shutdown hooks.
type App struct {
	name        string
	shortDesc   string
	description string
	run         RunFunc
	cmd         *cobra.Command
	args        cobra.PositionalArgs

	// +optional
	healthCheckFunc HealthCheckFunc

	// +optional
	options any

	// +optional
	silence   bool
	noConfig  bool
	watch     bool
	noVersion bool

	// +optional - config loading customization
	configName        string
	configSearchPaths []string
	envPrefix         string

	// +optional - lifecycle hooks
	preRun      LifecycleHook
	postRun     LifecycleHook
	preShutdown LifecycleHook

	// +optional - timeout for lifecycle hooks and graceful shutdown.
	shutdownTimeout time.Duration

	// +optional - slog options for structured logging.
	slogOpts *options.SlogOptions

	// +optional - version info for cobra's built-in --version flag.
	versionInfo *version.Info
}

// RunFunc defines the application's startup callback function.
// The ctx parameter is signal-aware: it is canceled on SIGINT, SIGTERM,
// or SIGQUIT, enabling graceful shutdown.
type RunFunc func(ctx context.Context) error

// HealthCheckFunc defines the health check function for the application.
type HealthCheckFunc func() error

// LifecycleHook is a function called at a specific lifecycle stage.
// For PreRun and PreShutdown hooks, ctx is signal-aware.
// For PostRun hooks, ctx is a timeout context (see WithShutdownTimeout).
type LifecycleHook func(ctx context.Context) error

// Option defines optional parameters for initializing the application.
type Option func(*App)

// ---------------------------------------------------------------------------
// Functional Options
// ---------------------------------------------------------------------------

// WithRunFunc sets the application startup callback function.
// The provided ctx will be signal-aware (canceled on SIGINT/SIGTERM/SIGQUIT).
func WithRunFunc(run RunFunc) Option {
	return func(a *App) {
		a.run = run
	}
}

// WithOptions sets the option struct for the application. The struct should
// implement FlagSetOptions or NamedFlagSetOptions to register CLI flags,
// and optionally provide Complete() and Validate() for lifecycle hooks.
func WithOptions(opts any) Option {
	return func(a *App) {
		a.options = opts
	}
}

// WithDescription sets the long description of the application shown in help output.
func WithDescription(desc string) Option {
	return func(a *App) {
		a.description = desc
	}
}

// WithHealthCheckFunc sets a custom health check function.
func WithHealthCheckFunc(fn HealthCheckFunc) Option {
	return func(a *App) {
		a.healthCheckFunc = fn
	}
}

// WithDefaultHealthCheckFunc sets the default health check function
// (starts /healthz on 0.0.0.0:20250).
func WithDefaultHealthCheckFunc() Option {
	return WithHealthCheckFunc(func() error {
		go options.NewHealthOptions().Serve()
		return nil
	})
}

// WithSilence suppresses startup information (name, version, config) in the console.
func WithSilence() Option {
	return func(a *App) {
		a.silence = true
	}
}

// WithNoConfig disables the --config flag and config file loading.
func WithNoConfig() Option {
	return func(a *App) {
		a.noConfig = true
	}
}

// WithValidArgs sets the positional argument validation function.
func WithValidArgs(args cobra.PositionalArgs) Option {
	return func(a *App) {
		a.args = args
	}
}

// WithDefaultValidArgs sets default validation (no positional args allowed).
func WithDefaultValidArgs() Option {
	return func(a *App) {
		a.args = cobra.NoArgs
	}
}

// WithWatchConfig enables watching and re-reading config files at runtime.
func WithWatchConfig() Option {
	return func(a *App) {
		a.watch = true
	}
}

// WithSlogOptions configures structured logging via slog. The framework calls
// SlogOptions.Apply() during PersistentPreRunE to set the global default logger.
func WithSlogOptions(opts *options.SlogOptions) Option {
	return func(a *App) {
		a.slogOpts = opts
	}
}

// WithPreRunHook adds a hook called after config/flags are loaded and logging
// is initialized, but before the RunFunc. The ctx is signal-aware.
func WithPreRunHook(hook LifecycleHook) Option {
	return func(a *App) {
		a.preRun = hook
	}
}

// WithPostRunHook adds a hook called after the RunFunc completes successfully.
// The ctx is a timeout context (see WithShutdownTimeout).
func WithPostRunHook(hook LifecycleHook) Option {
	return func(a *App) {
		a.postRun = hook
	}
}

// WithPreShutdownHook adds a hook that always runs after the RunFunc,
// regardless of whether it succeeded or failed. The ctx is a timeout context.
func WithPreShutdownHook(hook LifecycleHook) Option {
	return func(a *App) {
		a.preShutdown = hook
	}
}

// WithShutdownTimeout sets the timeout for PostRun and PreShutdown hooks.
// Default is 10 seconds.
func WithShutdownTimeout(d time.Duration) Option {
	return func(a *App) {
		a.shutdownTimeout = d
	}
}

// WithConfigSearchPaths sets the directories to search for config files.
// Defaults to [".", "$HOME/.<app-prefix>", "/etc/<app-prefix>"].
func WithConfigSearchPaths(paths []string) Option {
	return func(a *App) {
		a.configSearchPaths = paths
	}
}

// WithConfigName overrides the config file name. Defaults to the app name.
func WithConfigName(name string) Option {
	return func(a *App) {
		a.configName = name
	}
}

// WithEnvPrefix overrides the environment variable prefix. Defaults to the
// uppercased app name with hyphens replaced by underscores.
func WithEnvPrefix(prefix string) Option {
	return func(a *App) {
		a.envPrefix = prefix
	}
}

// WithVersionInfo sets the version information for the application. When set,
// cobra's built-in --version flag is used to display version info.
// For advanced version formatting (--version=raw), use pkg/version directly.
func WithVersionInfo(info *version.Info) Option {
	return func(a *App) {
		a.versionInfo = info
	}
}

// WithNoVersion disables the version flag entirely.
func WithNoVersion() Option {
	return func(a *App) {
		a.noVersion = true
	}
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewApp creates a new application instance with the given name, short
// description, and options. The cobra command is built immediately.
func NewApp(name string, shortDesc string, opts ...Option) *App {
	a := &App{
		name:            name,
		shortDesc:       shortDesc,
		shutdownTimeout: DefaultShutdownTimeout,
		run:             func(ctx context.Context) error { return nil },
	}

	for _, o := range opts {
		o(a)
	}

	a.buildCommand()
	return a
}

// ---------------------------------------------------------------------------
// Command Construction
// ---------------------------------------------------------------------------

// buildCommand constructs the cobra.Command using cobra's native hook chain:
//
//	PersistentPreRunE (setupCommand) → PreRunE (initCommand) → RunE (runCommand)
//
// This ensures framework setup propagates to all subcommands when
// EnableTraverseRunHooks is set (useful for multi-command CLI tools).
func (a *App) buildCommand() {
	cmd := &cobra.Command{
		Use:   formatBaseName(a.name),
		Short: a.shortDesc,
		Long:  a.description,
		Args:  a.args,

		// Cobra native hook chain:
		//   PersistentPreRunE: inherited by subcommands (config, logging)
		//   PreRunE: runs only for this command (health check)
		//   RunE: main signal-aware business logic
		PersistentPreRunE: a.setupCommand,
		PreRunE:           a.initCommand,
		RunE:              a.runCommand,

		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set version on cobra command if provided. Cobra auto-adds --version.
	if a.versionInfo != nil {
		cmd.Version = a.versionInfo.String()
		cmd.SetVersionTemplate(`{{.Version}}`)
	}

	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.Flags().SortFlags = true

	// Disable default completion command (backward compat).
	cmd.CompletionOptions.DisableDefaultCmd = true

	// Register flags based on options type.
	var fs *pflag.FlagSet
	switch typed := a.options.(type) {
	case NamedFlagSetOptions:
		fss := typed.Flags()
		for _, f := range fss.FlagSets {
			cmd.Flags().AddFlagSet(f.FlagSet)
		}
		fs = cmd.PersistentFlags()
	case FlagSetOptions:
		fs = cmd.PersistentFlags()
		typed.AddFlags(fs)
	default:
		fs = cmd.PersistentFlags()
	}

	// Add version flag using pkg/version if cobra.Version not set.
	if a.versionInfo == nil && !a.noVersion {
		version.AddFlags(fs)
	}

	// Add config flag if config loading is enabled.
	if !a.noConfig {
		a.addConfigFlag(cmd, fs)
	}

	a.cmd = cmd
}

// ---------------------------------------------------------------------------
// Cobra Hook: PersistentPreRunE
// ---------------------------------------------------------------------------

// setupCommand runs as PersistentPreRunE. It is inherited by all subcommands
// when EnableTraverseRunHooks is set. It handles: version check, flag
// unmarshaling, options completion/validation, and logging initialization.
func (a *App) setupCommand(cmd *cobra.Command, args []string) error {
	// Version check (only when using pkg/version, not cobra's built-in).
	if a.versionInfo == nil && !a.noVersion {
		version.PrintAndExitIfRequested()
	}

	// Bind parsed flags to viper and unmarshal into options.
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	if a.options != nil {
		if err := viper.Unmarshal(a.options); err != nil {
			return fmt.Errorf("failed to unmarshal options: %w", err)
		}

		// Complete: post-processing after flag binding.
		if c, ok := a.options.(interface{ Complete() error }); ok {
			if err := c.Complete(); err != nil {
				return fmt.Errorf("failed to complete options: %w", err)
			}
		}

		// Validate: supports both []error (IOptions style) and error styles.
		if v, ok := a.options.(OptionsValidator); ok {
			if errs := v.Validate(); len(errs) > 0 {
				return fmt.Errorf("invalid options: %v", errs)
			}
		} else if v, ok := a.options.(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return fmt.Errorf("invalid options: %w", err)
			}
		}
	}

	// Initialize structured logging.
	if a.slogOpts != nil {
		if err := a.slogOpts.Apply(); err != nil {
			return fmt.Errorf("failed to apply slog options: %w", err)
		}
	}

	// Startup info.
	if !a.silence {
		slog.Info("Starting application",
			"name", a.name,
			"goVersion", runtime.Version(),
			"platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		)
		if a.versionInfo != nil {
			slog.Info("Version", "version", a.versionInfo.String())
		}
		slog.Info("Golang settings",
			"GOGC", os.Getenv("GOGC"),
			"GOMAXPROCS", os.Getenv("GOMAXPROCS"),
			"GOTRACEBACK", os.Getenv("GOTRACEBACK"),
		)
		if !a.noConfig {
			a.printConfig()
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Cobra Hook: PreRunE
// ---------------------------------------------------------------------------

// initCommand runs as PreRunE for the root command only (not inherited).
// It starts the health check server if configured.
func (a *App) initCommand(cmd *cobra.Command, args []string) error {
	if a.healthCheckFunc != nil {
		if err := a.healthCheckFunc(); err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Cobra Hook: RunE
// ---------------------------------------------------------------------------

// runCommand is the main RunE. It creates a signal-aware context from cobra's
// command context, runs lifecycle hooks, and invokes the user's RunFunc.
func (a *App) runCommand(cmd *cobra.Command, args []string) error {
	// Derive a signal-aware context from cobra's context.
	// cmd.Context() is set by ExecuteContext (from RunContext / Run).
	ctx, cancel := signal.NotifyContext(
		cmd.Context(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT,
	)
	defer cancel()

	// PreRun hooks: called after setup, before the RunFunc.
	// The ctx is signal-aware, so hooks can set up resources that need
	// to be canceled on shutdown.
	if a.preRun != nil {
		if err := a.preRun(ctx); err != nil {
			return fmt.Errorf("pre-run hook failed: %w", err)
		}
	}

	// Execute the user's RunFunc. This typically blocks until the
	// context is canceled (shutdown) or a fatal error occurs.
	runErr := a.run(ctx)

	// PostRun hooks: only called on successful completion.
	if runErr == nil && a.postRun != nil {
		postCtx, postCancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer postCancel()
		if err := a.postRun(postCtx); err != nil {
			slog.Error("post-run hook failed", "err", err)
		}
	}

	// PreShutdown hooks: always called, regardless of success/failure.
	if a.preShutdown != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer shutdownCancel()
		if err := a.preShutdown(shutdownCtx); err != nil {
			slog.Error("pre-shutdown hook failed", "err", err)
		}
	}

	return runErr
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Run launches the application. It uses context.Background() as the parent
// context and calls os.Exit(1) on error.
func (a *App) Run() {
	if err := a.RunContext(context.Background()); err != nil {
		os.Exit(1)
	}
}

// RunContext launches the application with a parent context. This enables
// testing and embedding scenarios. The parent context is stored by cobra
// and later used to derive the signal-aware context for the RunFunc:
//
//	parentCtx → cobra cmd.ctx → signal ctx → RunFunc
func (a *App) RunContext(ctx context.Context) error {
	return a.cmd.ExecuteContext(ctx)
}

// Command returns the underlying cobra command. This enables advanced use
// cases such as adding subcommands, customizing help templates, or
// registering flag groups (MarkFlagsMutuallyExclusive, etc.).
func (a *App) Command() *cobra.Command {
	return a.cmd
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// formatBaseName formats the app name for the current OS (lowercase on Windows,
// strips .exe suffix).
func formatBaseName(name string) string {
	if runtime.GOOS == "windows" {
		name = strings.ToLower(name)
		name = strings.TrimSuffix(name, ".exe")
	}
	return name
}