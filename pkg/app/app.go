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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
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

	// v is the viper instance for this app. An App is the single application
	// instance in a process, so this instance is intentionally not shared.
	v *viper.Viper

	// cfgFile holds the value of the --config flag, bound via pflag.
	cfgFile string

	// +optional
	healthCheckFunc HealthCheckFunc

	// +optional
	options any

	// +optional
	silence   bool
	noConfig  bool
	noVersion bool

	// +optional - config loading customization
	configName        string
	configSearchPaths []string
	dirInHome         string
	envPrefix         string

	// +optional - lifecycle hooks. Each stage supports multiple hooks,
	// executed in registration order.
	preRunHooks      []LifecycleHook
	postRunHooks     []LifecycleHook
	preShutdownHooks []LifecycleHook

	// +optional - timeout for lifecycle hooks and graceful shutdown.
	shutdownTimeout time.Duration

	// +optional - slog options for structured logging.
	slogOpts *options.SlogOptions

	// +optional - version info for cobra's built-in --version flag.
	versionInfo *version.Info

	// +optional - signals that trigger graceful shutdown.
	signals []os.Signal

	// shutdowners holds components started during the lifecycle that must be
	// gracefully stopped after the PreShutdown hook.
	shutdowners []Shutdowner

	// ran guards the single-run invariant. It is an atomic flag set exactly
	// once; a subsequent RunContext call fails fast without blocking.
	ran atomic.Bool
}

// RunFunc defines the application's startup callback function.
// The ctx parameter is signal-aware: it is canceled on the configured
// signals (SIGINT, SIGTERM, SIGQUIT by default), enabling graceful shutdown.
type RunFunc func(ctx context.Context) error

// HealthCheckFunc defines the health check function for the application.
// It is called during PreRunE, after config/flags are loaded.
type HealthCheckFunc func() error

// LifecycleHook is a function called at a specific lifecycle stage.
// For PreRun and PreShutdown hooks, ctx is signal-aware.
// For PostRun hooks, ctx is a timeout context (see WithShutdownTimeout).
type LifecycleHook func(ctx context.Context) error

// ConfigChangeFunc is invoked after a watched config file changes and the new
// config has been successfully re-unmarshaled and re-validated. Implementations
// must be safe for concurrent use and must not block the viper watch goroutine.
type ConfigChangeFunc func()

// Option defines optional parameters for initializing the application.
type Option func(*App)

// ---------------------------------------------------------------------------
// Functional Options
// ---------------------------------------------------------------------------

// WithRun sets the application startup callback function.
// The provided ctx will be signal-aware (canceled on SIGINT/SIGTERM/SIGQUIT).
func WithRun(run RunFunc) Option {
	return func(a *App) {
		a.run = run
	}
}

// WithOptions sets the option struct for the application. The struct should
// implement FlagSetOptions or NamedFlagSetOptions to register CLI flags,
// and may implement Completer() and Validate() for lifecycle hooks.
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

// WithHealthCheck sets a custom health check function.
func WithHealthCheck(fn HealthCheckFunc) Option {
	return func(a *App) {
		a.healthCheckFunc = fn
	}
}

// WithDefaultHealthCheck sets the default health check function
// (starts /healthz on 0.0.0.0:20250 by default). If the app's options implement
// healthOptionsProvider (e.g. *options.ServerOptions), the address, path and
// pprof exposure are taken from its *options.HealthOptions instead. The server
// is started in PreRunE and gracefully shut down at the end of the lifecycle.
func WithDefaultHealthCheck() Option {
	return func(a *App) {
		a.healthCheckFunc = a.startDefaultHealthCheck
	}
}

// healthOptionsProvider is implemented by option structs that expose a
// *options.HealthOptions to configure the built-in health check server.
type healthOptionsProvider interface {
	HealthOptions() *options.HealthOptions
}

// startDefaultHealthCheck builds and starts the built-in health check server,
// deriving its address, path and pprof exposure from the app options when
// available, and registers it as a shutdowner.
func (a *App) startDefaultHealthCheck() error {
	addr, path := options.DefaultHealthCheckAddress, options.DefaultHealthCheckPath
	enableProfiler := false

	if p, ok := a.options.(healthOptionsProvider); ok {
		if ho := p.HealthOptions(); ho != nil {
			if ho.HealthCheckAddress != "" {
				addr = ho.HealthCheckAddress
			}
			if ho.HealthCheckPath != "" {
				path = ho.HealthCheckPath
			}
			enableProfiler = ho.HTTPProfile
		}
	}

	h := newHealthServer(addr, path, enableProfiler)
	if err := h.Start(); err != nil {
		return err
	}
	a.registerShutdowner(h)
	return nil
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

// WithSlogOptions configures structured logging via slog. The framework calls
// SlogOptions.Apply() during PersistentPreRunE to set the global default logger.
func WithSlogOptions(opts *options.SlogOptions) Option {
	return func(a *App) {
		a.slogOpts = opts
	}
}

// WithPreRunHook adds a hook called after config/flags are loaded and logging
// is initialized, but before the RunFunc. The ctx is signal-aware. Multiple
// hooks are executed in the order they are registered.
func WithPreRunHook(hook LifecycleHook) Option {
	return func(a *App) {
		a.preRunHooks = append(a.preRunHooks, hook)
	}
}

// WithPostRunHook adds a hook called after the RunFunc completes successfully.
// The ctx is a timeout context (see WithShutdownTimeout). Multiple hooks are
// executed in registration order and share a single timeout budget.
func WithPostRunHook(hook LifecycleHook) Option {
	return func(a *App) {
		a.postRunHooks = append(a.postRunHooks, hook)
	}
}

// WithPreShutdownHook adds a hook that always runs after the RunFunc,
// regardless of whether it succeeded or failed. The ctx is a timeout context.
// Multiple hooks are executed in registration order and share a single timeout
// budget.
func WithPreShutdownHook(hook LifecycleHook) Option {
	return func(a *App) {
		a.preShutdownHooks = append(a.preShutdownHooks, hook)
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
// Defaults to [".", "$HOME/.<name>", "/etc/<name>"].
func WithConfigSearchPaths(paths []string) Option {
	return func(a *App) {
		a.configSearchPaths = paths
	}
}

// WithConfig sets the default directory where the configuration file is stored
// and the configuration file name, in one call. It is a convenience wrapper
// around WithConfigSearchPaths and WithConfigName: the directory replaces the
// default search paths, and the name overrides the app name.
func WithConfig(dir, name string) Option {
	return func(a *App) {
		a.configSearchPaths = []string{dir}
		a.configName = name
	}
}

// WithConfigName overrides the config file name. Defaults to the app name.
func WithConfigName(name string) Option {
	return func(a *App) {
		a.configName = name
	}
}

// WithDirInHome sets the directory (relative to the user's home directory) in
// which the app searches for its config file. For example, WithDirInHome(".onexai")
// makes the app search in $HOME/.onexai instead of the default $HOME/.<name>.
// It only affects the default search paths; explicit WithConfigSearchPaths or
// WithConfig calls take precedence.
func WithDirInHome(dir string) Option {
	return func(a *App) {
		a.dirInHome = dir
	}
}

// WithEnvPrefix overrides the environment variable prefix. Defaults to the
// uppercased app name with hyphens replaced by underscores.
func WithEnvPrefix(prefix string) Option {
	return func(a *App) {
		a.envPrefix = prefix
	}
}

// WithSignals overrides the set of signals that trigger graceful shutdown.
// Defaults to SIGINT, SIGTERM, and SIGQUIT.
func WithSignals(sigs ...os.Signal) Option {
	return func(a *App) {
		a.signals = sigs
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
		description:     shortDesc,
		shutdownTimeout: 10 * time.Second,
		signals:         defaultSignals(),
		envPrefix:       strings.ReplaceAll(strings.ToUpper(name), "-", "_"),
		v:               viper.New(),
		args:            cobra.NoArgs,
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
//	PersistentPreRunE (setupCommand) → PreRunE (health check) → RunE (runCommand)
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
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if a.healthCheckFunc != nil {
				return a.healthCheckFunc()
			}
			return nil
		},
		RunE: a.runCommand,

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

	// Register user option flags on the local flag set. This keeps viper flag
	// binding (BindPFlags) uniform and preserves flag > config precedence.
	switch typed := a.options.(type) {
	case NamedFlagSetOptions:
		fss := typed.Flags()
		for _, f := range fss.FlagSets {
			cmd.Flags().AddFlagSet(f.FlagSet)
		}
	case FlagSetOptions:
		typed.AddFlags(cmd.Flags())
	}

	// Register framework flags (version/config) on persistent flags so they
	// are inherited by subcommands.
	fs := cmd.PersistentFlags()

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
// when EnableTraverseRunHooks is set. It handles: version check, config/env
// loading, flag binding, options completion/validation, and logging init.
func (a *App) setupCommand(cmd *cobra.Command, args []string) error {
	// Version check (only when using pkg/version, not cobra's built-in).
	if a.versionInfo == nil && !a.noVersion {
		version.PrintAndExitIfRequested()
	}

	// Load config file and env vars (if enabled) before binding flags, so
	// explicitly-set flags take precedence over config and env values.
	if err := a.loadConfig(); err != nil {
		return err
	}

	// Bind flags to viper. Both local flags (user options) and persistent
	// flags (config/version) are bound.
	if err := a.v.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	if err := a.v.BindPFlags(cmd.PersistentFlags()); err != nil {
		return err
	}

	if err := a.applyOptions(); err != nil {
		return err
	}

	// Initialize structured logging.
	if a.slogOpts != nil {
		if err := a.slogOpts.Apply(); err != nil {
			return fmt.Errorf("failed to apply slog options: %w", err)
		}
	}

	// Startup info.
	if !a.silence {
		slog.Info(
			"Starting application",
			"name", a.name,
			"goVersion", runtime.Version(),
			"platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		)
		if a.versionInfo != nil {
			slog.Info("Version", "version", a.versionInfo.String())
		}
		slog.Info(
			"Golang settings",
			"GOGC", os.Getenv("GOGC"),
			"GOMAXPROCS", os.Getenv("GOMAXPROCS"),
			"GOTRACEBACK", os.Getenv("GOTRACEBACK"),
		)

		a.printConfig()
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
	ctx, cancel := signal.NotifyContext(cmd.Context(), a.signals...)
	defer cancel()

	// PreRun hooks: called after setup, before the RunFunc.
	// The ctx is signal-aware, so hooks can set up resources that need
	// to be canceled on shutdown.
	for _, hook := range a.preRunHooks {
		if err := hook(ctx); err != nil {
			return fmt.Errorf("pre-run hook failed: %w", err)
		}
	}

	// Execute the user's RunFunc. This typically blocks until the
	// context is canceled (shutdown) or a fatal error occurs.
	runErr := a.run(ctx)

	// PostRun hooks: only called on successful completion. All hooks in this
	// stage share a single timeout budget (see WithShutdownTimeout).
	if runErr == nil && len(a.postRunHooks) > 0 {
		postCtx, postCancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		for _, hook := range a.postRunHooks {
			if err := hook(postCtx); err != nil {
				slog.Error("post-run hook failed", "err", err)
			}
		}
		postCancel()
	}

	// PreShutdown hooks: always called, regardless of success/failure. All
	// hooks in this stage share a single timeout budget (see WithShutdownTimeout).
	if len(a.preShutdownHooks) > 0 {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		for _, hook := range a.preShutdownHooks {
			if err := hook(shutdownCtx); err != nil {
				slog.Error("pre-shutdown hook failed", "err", err)
			}
		}
		shutdownCancel()
	}

	// Stop framework-managed components (e.g. the default health server) in
	// reverse registration order (LIFO), so components started later are
	// stopped before the ones they may depend on.
	for i := len(a.shutdowners) - 1; i >= 0; i-- {
		s := a.shutdowners[i]
		sCtx, sCancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		if err := s.Shutdown(sCtx); err != nil {
			slog.Error("component shutdown failed", "err", err)
		}
		sCancel()
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
		slog.Error("application exited with error", "err", err)
		os.Exit(1)
	}
}

// RunContext launches the application with a parent context. This enables
// testing and embedding scenarios. The parent context is stored by cobra
// and later used to derive the signal-aware context for the RunFunc:
//
//	parentCtx → cobra cmd.ctx → signal ctx → RunFunc
//
// An App may only be run once; subsequent calls fail fast with an error.
func (a *App) RunContext(ctx context.Context) error {
	if a.ran.Swap(true) {
		return errors.New("app: RunContext called more than once")
	}

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

// registerShutdowner appends a component to be gracefully stopped at the end
// of the lifecycle. It is safe to call from lifecycle hooks, which run on the
// single cobra execution goroutine.
func (a *App) registerShutdowner(s Shutdowner) {
	a.shutdowners = append(a.shutdowners, s)
}

// applyOptions unmarshals the loaded configuration into the app options and
// runs the optional Complete and Validate hooks. It is shared by the initial
// setup (setupCommand) and by config hot-reload (reloadConfig).
func (a *App) applyOptions() error {
	if a.options == nil {
		return nil
	}

	if err := a.v.Unmarshal(a.options); err != nil {
		return fmt.Errorf("failed to unmarshal options: %w", err)
	}

	// Complete: post-processing after flag binding.
	if c, ok := a.options.(OptionsCompleter); ok {
		if err := c.Complete(); err != nil {
			return fmt.Errorf("failed to complete options: %w", err)
		}
	}

	// Validate: error-style.
	if v, ok := a.options.(OptionsValidator); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("invalid options: %w", err)
		}
	}

	return nil
}

// reloadConfig re-applies the current viper configuration to the app options
// after a watched config file changes, mirroring the startup sequence in
// setupCommand. It runs on the viper watch goroutine; a concurrent RunFunc
// reading the options must synchronize with this write via the caller's own
// locking, as viper's hot-reload path is not internally synchronized.
func (a *App) reloadConfig() error {
	return a.applyOptions()
}

// effectiveConfigName returns the config file name to use.
func (a *App) effectiveConfigName() string {
	if a.configName != "" {
		return a.configName
	}
	return a.name
}

// effectiveConfigSearchPaths returns the directories to search for config files.
func (a *App) effectiveConfigSearchPaths() []string {
	if len(a.configSearchPaths) > 0 {
		return a.configSearchPaths
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	homeConfigDir := "." + a.name
	if a.dirInHome != "" {
		homeConfigDir = a.dirInHome
	}

	return []string{
		".",
		filepath.Join(homeDir, homeConfigDir),
		filepath.Join("/etc", a.name),
	}
}

// formatBaseName formats the app name for the current OS (lowercase on Windows,
// strips .exe suffix).
func formatBaseName(name string) string {
	if runtime.GOOS == "windows" {
		name = strings.ToLower(name)
		name = strings.TrimSuffix(name, ".exe")
	}
	return name
}
