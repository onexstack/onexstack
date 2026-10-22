// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onexstack.

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// fakeRemoteSource is a RemoteConfigSource with no config center behind it. It
// is the test double that makes the remote path exercisable without a Polaris
// to talk to — and it is deliberately a test double: nothing ships one, because
// a configuration source that fabricates values is worse than no source at all.
type fakeRemoteSource struct {
	tree map[string]any
	err  error

	// changes, when non-nil, drives the watch: each value is delivered to the
	// change callback in order.
	changes chan map[string]any

	mu      sync.Mutex
	closed  bool
	watched bool
}

func (f *fakeRemoteSource) LoadConfigTree(ctx context.Context) (map[string]any, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.tree, nil
}

func (f *fakeRemoteSource) WatchConfigTree(ctx context.Context, onChange func(tree map[string]any)) error {
	f.mu.Lock()
	f.watched = true
	f.mu.Unlock()

	if f.changes == nil {
		return nil
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case tree, ok := <-f.changes:
				if !ok {
					return
				}
				onChange(tree)
			}
		}
	}()
	return nil
}

func (f *fakeRemoteSource) CloseConfigSource() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeRemoteSource) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

func (f *fakeRemoteSource) wasWatched() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.watched
}

// writeConfig writes a config file into a temp dir and returns the dir.
func writeConfig(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// runOnce starts the app through the real startup path and returns the options
// as the RunFunc saw them. args are command-line arguments, which is how a test
// marks a flag as explicitly set — a flag at its default value must not beat a
// remote value, and only HasChanged distinguishes the two.
func runOnce(t *testing.T, dir string, src RemoteConfigSource, args ...string) testOptions {
	t.Helper()

	opts := &testOptions{}
	var captured testOptions

	app := NewApp("prectest", "prectest",
		WithSilence(),
		WithOptions(opts),
		WithConfigSearchPaths([]string{dir}),
		WithConfigName("prectest"),
		WithRemoteConfigSource(src),
		WithRun(func(ctx context.Context) error {
			captured = *opts
			return nil
		}),
	)
	app.cmd.SetArgs(args)

	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("RunContext failed: %v", err)
	}
	return captured
}

// TestRemoteConfigPrecedenceChain pins the ordering the whole feature rests on:
// flag, then environment, then the remote source, then the config file, then the
// flag's default value.
//
// Each layer is asserted in turn rather than only the end state, because the
// interesting property is not that the remote value arrives — it is where it
// arrives relative to the two layers around it. A merge into viper's override
// layer would put it above the flags; a merge read before the file would put it
// below. Both would pass a test that only checked "the remote value is present".
func TestRemoteConfigPrecedenceChain(t *testing.T) {
	const file = "level: file\nport: 9090\n"
	remote := &fakeRemoteSource{tree: map[string]any{"level": "remote"}}

	t.Run("remote beats the config file", func(t *testing.T) {
		got := runOnce(t, writeConfig(t, "prectest", file), remote)
		if got.Level != "remote" {
			t.Fatalf("level = %q, want remote", got.Level)
		}
		if got.Port != 9090 {
			t.Fatalf("port = %d, want 9090 from the file: the remote tree must not clear what it does not define", got.Port)
		}
	})

	t.Run("environment beats the remote source", func(t *testing.T) {
		t.Setenv("PRECTEST_LEVEL", "env")
		got := runOnce(t, writeConfig(t, "prectest", file), remote)
		if got.Level != "env" {
			t.Fatalf("level = %q, want env", got.Level)
		}
	})

	t.Run("an explicit flag beats the environment", func(t *testing.T) {
		t.Setenv("PRECTEST_LEVEL", "env")
		got := runOnce(t, writeConfig(t, "prectest", file), remote, "--level=flag")
		if got.Level != "flag" {
			t.Fatalf("level = %q, want flag", got.Level)
		}
	})

	t.Run("a flag left at its default does not beat the remote source", func(t *testing.T) {
		// --port is not passed, so it holds its default (8080). The file says
		// 9090 and must win: viper prefers a flag only when it has changed.
		got := runOnce(t, writeConfig(t, "prectest", file), remote)
		if got.Port != 9090 {
			t.Fatalf("port = %d, want 9090: an unset flag's default must not beat the config", got.Port)
		}
	})

	t.Run("with no remote source the file still decides", func(t *testing.T) {
		got := runOnce(t, writeConfig(t, "prectest", file), nil)
		if got.Level != "file" {
			t.Fatalf("level = %q, want file", got.Level)
		}
	})
}

// TestRemoteConfigCannotRedefineItsOwnSection checks that a remote tree cannot
// carry the section that configures the remote source. The source is already
// constructed by the time the tree arrives and could not honour a new address,
// so applying the value would leave the running configuration disagreeing with
// the file that describes it.
func TestRemoteConfigCannotRedefineItsOwnSection(t *testing.T) {
	dir := writeConfig(t, "prectest", "level: file\n")
	remote := &fakeRemoteSource{tree: map[string]any{
		"level":         "remote",
		"config-source": map[string]any{"type": "polaris"},
		"Config-Source": map[string]any{"type": "file"},
		"config_source": map[string]any{"type": "file"},
	}}

	opts := &testOptions{}
	app := NewApp("prectest", "prectest",
		WithSilence(),
		WithOptions(opts),
		WithConfigSearchPaths([]string{dir}),
		WithConfigName("prectest"),
		WithRemoteConfigSource(remote),
		WithRun(func(ctx context.Context) error { return nil }),
	)
	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("RunContext failed: %v", err)
	}

	for _, key := range []string{"config-source", "Config-Source", "config_source"} {
		if app.v.InConfig(key) {
			t.Errorf("%q survived the merge; a remote tree must not configure the source that produced it", key)
		}
	}
	if got := app.v.GetString("level"); got != "remote" {
		t.Fatalf("level = %q, want remote — the rest of the tree must still be merged", got)
	}
}

// TestRemoteConfigFailureAbortsStartup checks the error branch of the source
// contract: a source that fails its own fail-open policy by returning an error
// takes the process down, rather than starting on a configuration the operator
// did not intend.
//
// The policy itself — which failures warn and which abort — lives with the
// source, since it is the only party that knows what was configured. This is the
// app's half of the agreement.
func TestRemoteConfigFailureAbortsStartup(t *testing.T) {
	dir := writeConfig(t, "prectest", "level: file\n")
	remote := &fakeRemoteSource{err: errors.New("polaris unreachable")}

	app := NewApp("prectest", "prectest",
		WithSilence(),
		WithOptions(&testOptions{}),
		WithConfigSearchPaths([]string{dir}),
		WithConfigName("prectest"),
		WithRemoteConfigSource(remote),
		WithRun(func(ctx context.Context) error { return nil }),
	)

	err := app.RunContext(context.Background())
	if err == nil {
		t.Fatal("expected startup to fail when the config source returns an error")
	}
	if !errors.Is(err, remote.err) {
		t.Fatalf("error = %v, want it to wrap %v", err, remote.err)
	}
}

// TestRemoteConfigAbsentIsInert checks the other branch: a source that reports
// there is nothing to load — which is what an unconfigured or non-strict
// unreachable source does — leaves startup exactly as it was before the feature
// existed.
func TestRemoteConfigAbsentIsInert(t *testing.T) {
	dir := writeConfig(t, "prectest", "level: file\nport: 9090\n")
	remote := &fakeRemoteSource{}

	got := runOnce(t, dir, remote)
	if got.Level != "file" || got.Port != 9090 {
		t.Fatalf("level=%q port=%d, want file/9090 from the file alone", got.Level, got.Port)
	}
}

// TestRemoteConfigIsNotWatchedWithoutACallback pins that a service which only
// wants its configuration resolved at startup does not pay for a subscription.
// Nothing reads the new configuration without a callback — the app does not
// write it back to the live options — so watching would poll a config center for
// values that are discarded on arrival.
func TestRemoteConfigIsNotWatchedWithoutACallback(t *testing.T) {
	dir := writeConfig(t, "prectest", "level: file\n")
	remote := &fakeRemoteSource{tree: map[string]any{"level": "remote"}}

	got := runOnce(t, dir, remote)
	if got.Level != "remote" {
		t.Fatalf("level = %q, want remote: the startup load must still happen", got.Level)
	}
	if remote.wasWatched() {
		t.Error("the source was watched even though no change callback was registered")
	}
}

// TestRemoteConfigChangeIsAppliedWithoutTouchingLiveOptions runs a change
// through the real lifecycle, concurrently with a RunFunc that reads the
// options. Run it under -race: the point is that a reload does not write the
// options the RunFunc is holding.
func TestRemoteConfigChangeIsAppliedWithoutTouchingLiveOptions(t *testing.T) {
	dir := writeConfig(t, "prectest", "level: file\nport: 9090\n")
	remote := &fakeRemoteSource{
		tree:    map[string]any{"level": "remote"},
		changes: make(chan map[string]any, 8),
	}

	opts := &testOptions{}
	var (
		mu       sync.Mutex
		reloaded *viper.Viper
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := NewApp("prectest", "prectest",
		WithSilence(),
		WithOptions(opts),
		WithConfigSearchPaths([]string{dir}),
		WithConfigName("prectest"),
		WithRemoteConfigSource(remote),
		WithConfigChange(func(v *viper.Viper) error {
			mu.Lock()
			reloaded = v
			mu.Unlock()
			return nil
		}),
		WithRun(func(runCtx context.Context) error {
			// Read the options the way a wired server does, for as long as the
			// app runs. A reload that wrote them would race with this loop, and
			// the race detector is what this test is for.
			for {
				_ = opts.Level
				_ = opts.Port
				select {
				case <-runCtx.Done():
					return nil
				default:
				}
			}
		}),
	)

	done := make(chan error, 1)
	go func() { done <- app.RunContext(ctx) }()

	// Startup runs concurrently; the channel is buffered so the values are held
	// until the watch is established.
	for range 3 {
		remote.changes <- map[string]any{"level": "remote", "port": 7070}
	}

	// Wait for the change to be observed rather than sleeping a fixed time.
	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		seen := reloaded
		mu.Unlock()
		if seen != nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("the change callback was never invoked")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunContext failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunContext did not return after the context was cancelled")
	}

	mu.Lock()
	got := reloaded
	mu.Unlock()
	if got.GetString("level") != "remote" || got.GetInt("port") != 7070 {
		t.Fatalf("reloaded level=%q port=%d, want remote/7070", got.GetString("level"), got.GetInt("port"))
	}

	// The source is closed as part of shutdown, so a process does not exit with
	// a half-open connection and a background task still running.
	if !remote.isClosed() {
		t.Error("the config source was not closed on shutdown")
	}
}
