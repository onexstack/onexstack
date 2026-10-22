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

	"github.com/spf13/viper"
)

// RemoteConfigSource supplies configuration from outside the config file — a
// config center such as Polaris. An application wires one in with
// WithRemoteConfigSource; from then on the app asks it for a configuration tree
// at startup and after that for changes.
//
// The interface is satisfied structurally. An implementation lives in the
// module that owns the config center client and does not import this package,
// so the only place the two meet is the service that passes one to the other.
//
// Both loaders own the fail-open decision, because the implementation is the
// only party that knows what was configured. Their contract:
//
//   - a configuration that cannot be read, in strict mode: return an error,
//     which aborts startup;
//   - a configuration that cannot be read, otherwise: report it and return
//     (nil, nil), so the app proceeds on the local configuration alone;
//   - no source configured: return (nil, nil) and do nothing, which is what
//     makes the feature inert unless an operator turns it on.
type RemoteConfigSource interface {
	// LoadConfigTree returns the merged remote configuration, or (nil, nil) when
	// there is nothing to load.
	LoadConfigTree(ctx context.Context) (map[string]any, error)
	// WatchConfigTree invokes onChange with a freshly loaded tree on every
	// change, until ctx is cancelled. It returns once the watch is established.
	WatchConfigTree(ctx context.Context, onChange func(tree map[string]any)) error
	// CloseConfigSource releases the source's resources.
	CloseConfigSource() error
}

// remoteConfigSection is the top-level configuration key that configures the
// remote source itself.
const remoteConfigSection = "config-source"

// WithRemoteConfigSource enables a remote configuration source. Without it the
// app reads only the config file, the environment and the flags, as before.
func WithRemoteConfigSource(src RemoteConfigSource) Option {
	return func(a *App) {
		a.remoteConfig = src
	}
}

// WithConfigChange registers a callback for remote configuration changes.
//
// The callback receives a freshly built viper instance holding the new
// configuration. It is a *new* instance, rebuilt from the sources on each
// change and discarded afterwards — not the app's live one, and not the live
// options value.
//
// That is deliberate. The RunFunc and everything it started read the options
// they were given without locking, so re-unmarshaling into that value from a
// watcher goroutine would be a data race, and for most settings a late write
// could not take effect anyway: connections are built once, at startup, from
// values copied out of the options. Handing the callback a fresh instance puts
// the decision of what can be applied at runtime, and how, with the code that
// knows — a log level or a rate limit can be swapped, a database pool cannot.
//
// The callback runs on its own goroutine and must not block. Reloads are
// serialized: a callback never runs concurrently with another reload.
func WithConfigChange(fn ConfigChangeFunc) Option {
	return func(a *App) {
		a.configChangeHooks = append(a.configChangeHooks, fn)
	}
}

// loadRemoteConfig resolves the remote source, loads it, and merges what it
// returns over the config file.
//
// It runs after the config file and the flags are in viper and before the
// options are applied, so a value from the config center lands in the options
// as if it had been written in the file — except that it wins over the file and
// loses to an explicit flag or an environment variable, which is the precedence
// the rest of the framework already implements (viper resolves it per lookup:
// flag, then environment, then config).
func (a *App) loadRemoteConfig(ctx context.Context) error {
	if a.remoteConfig == nil || a.noConfig {
		return nil
	}

	// The source's own settings — which source, at which address, reading which
	// file — come from the local config file, the environment and the flags like
	// every other setting, so they have to be resolved before the source can be
	// asked for anything. Hence the unmarshal here and again in applyOptions.
	//
	// Validate is deliberately skipped on this pass: a required setting that only
	// the remote configuration supplies would otherwise be rejected before it
	// could be read, which is the one failure mode this ordering exists to
	// avoid.
	if err := a.unmarshalOptions(); err != nil {
		return err
	}

	tree, err := a.remoteConfig.LoadConfigTree(ctx)
	if err != nil {
		return fmt.Errorf("failed to load remote configuration: %w", err)
	}
	if tree == nil {
		return nil
	}

	// A remote tree must not redefine where remote configuration comes from. The
	// source is already constructed and could not honour a new type or address,
	// so applying the value would leave the running configuration silently
	// disagreeing with the file that describes it.
	removeRemoteConfigSection(tree)

	// MergeConfigMap writes into the config layer, not the override layer, so a
	// remote value is stronger than the file and weaker than the environment and
	// the flags. Merge rather than ReadConfig because the file's values must
	// survive: the file is the bootstrap, not the thing being replaced.
	if err := a.v.MergeConfigMap(tree); err != nil {
		return fmt.Errorf("failed to merge remote configuration: %w", err)
	}

	slog.Info("merged remote configuration",
		"keys", len(tree),
		"localFile", a.v.ConfigFileUsed(),
	)

	// Tear the client down on shutdown, so a process does not exit with a
	// half-open connection and a running background task.
	a.registerShutdowner(&remoteConfigShutdowner{src: a.remoteConfig})
	return nil
}

// startRemoteConfigWatch begins watching the remote source, if there is one and
// something is listening for changes. It must be called after the options have
// been applied, on the RunFunc's goroutine, so the ctx it receives is the
// signal-aware one.
func (a *App) startRemoteConfigWatch(ctx context.Context) {
	if a.remoteConfig == nil || a.noConfig {
		return
	}

	// No callbacks means no one to hand a change to, and nothing else reads the
	// new configuration — the app does not write it back to the live options.
	// Subscribing anyway would open a connection and poll a config center for
	// values that are discarded on arrival, so a service that only wants its
	// configuration resolved at startup gets no watch at all.
	if len(a.configChangeHooks) == 0 {
		slog.Debug("no config change callback registered; remote changes will not be watched")
		return
	}

	onChange := func(tree map[string]any) {
		if err := a.reloadConfig(tree); err != nil {
			slog.Error("remote configuration changed but could not be applied; keeping the running configuration",
				"err", err)
			return
		}
		slog.Info("applied remote configuration change")
	}

	if err := a.remoteConfig.WatchConfigTree(ctx, onChange); err != nil {
		// The configuration was loadable at startup, so this is the watch
		// failing rather than the source — a reason to log loudly, not to take a
		// healthy, serving process down.
		slog.Error("failed to watch remote configuration; changes will need a restart", "err", err)
	}
}

// reloadConfig rebuilds the configuration from its sources in response to a
// remote change and offers the result to the registered callbacks.
//
// It rebuilds into a fresh viper instance rather than re-reading into a.v, for
// two reasons. viper is not safe for concurrent use, and a.v is read by
// whichever goroutine a Get... reaches. And viper's merge only adds or
// overwrites: it has no way to express that a key the previous revision defined
// is gone, so updating in place would let a setting survive its own removal —
// the running configuration drifting from the one an operator can see. Both
// problems disappear when the configuration is derived from its sources again
// instead of amended.
func (a *App) reloadConfig(tree map[string]any) error {
	// Nothing to hand the result to. startRemoteConfigWatch does not subscribe
	// in this case, so reaching here means a direct call; it is still not worth
	// re-reading the file and merging for a value nobody will read.
	if len(a.configChangeHooks) == 0 {
		return nil
	}

	a.reloadMu.Lock()
	defer a.reloadMu.Unlock()

	v, err := a.seedConfig()
	if err != nil {
		// Includes the local file having become unreadable. Aborting leaves the
		// running configuration untouched, which is the safe end of the range:
		// the alternative would be to serve a configuration with the file's
		// settings silently missing.
		return fmt.Errorf("failed to re-read the local configuration: %w", err)
	}
	removeRemoteConfigSection(tree)
	if err := v.MergeConfigMap(tree); err != nil {
		return fmt.Errorf("failed to merge remote configuration: %w", err)
	}

	for _, hook := range a.configChangeHooks {
		if err := hook(v); err != nil {
			return fmt.Errorf("config change callback failed: %w", err)
		}
	}
	return nil
}

// seedConfig builds a viper instance holding what the app starts with, minus
// the remote merge: the same config file, environment settings and bound flags
// as the startup instance. It exists so a reload can derive the configuration
// from its sources rather than from the instance it is about to replace.
func (a *App) seedConfig() (*viper.Viper, error) {
	v := viper.New()
	if err := a.loadConfigInto(v); err != nil {
		return nil, err
	}
	if a.cmd != nil {
		if err := v.BindPFlags(a.cmd.Flags()); err != nil {
			return nil, err
		}
		if err := v.BindPFlags(a.cmd.PersistentFlags()); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// removeRemoteConfigSection deletes the remote source's own section from a
// remote tree. Matching ignores case, hyphens and underscores, so "Config-Source"
// and "config_source" are removed too: a section that survived the check would be
// merged, and would then configure the source on the next reload. The underscore
// form is not the same key to viper, and would be inert there — it is removed
// anyway, because a near-miss spelling is not worth leaving as a second, live-
// looking description of where the configuration came from.
func removeRemoteConfigSection(tree map[string]any) {
	for k := range tree {
		if normalizeConfigKey(k) == normalizeConfigKey(remoteConfigSection) {
			delete(tree, k)
		}
	}
}

func normalizeConfigKey(s string) string {
	s = strings.ToLower(s)
	return strings.NewReplacer("-", "", "_", "").Replace(s)
}

// remoteConfigShutdowner closes a remote config source at shutdown.
type remoteConfigShutdowner struct {
	src RemoteConfigSource
}

func (s *remoteConfigShutdowner) Shutdown(ctx context.Context) error {
	return s.src.CloseConfigSource()
}
