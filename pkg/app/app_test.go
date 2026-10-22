// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	genericoptions "github.com/onexstack/onexstack/pkg/options"
)

// testOptions is a minimal options struct implementing FlagSetOptions for tests.
type testOptions struct {
	Level string `mapstructure:"level"`
	Port  int    `mapstructure:"port"`
}

func (o *testOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Level, "level", "info", "log level")
	fs.IntVar(&o.Port, "port", 8080, "listen port")
}

func (o *testOptions) Validate() error { return nil }

func TestFormatBaseName(t *testing.T) {
	if got := formatBaseName("MyApp"); got != "MyApp" {
		t.Fatalf("formatBaseName(MyApp) = %q, want MyApp", got)
	}
}

func TestNamedFlagSetsFlagSet(t *testing.T) {
	n := NewNamedFlagSets()

	fs1 := n.FlagSet("server")
	fs2 := n.FlagSet("server")
	if fs1 != fs2 {
		t.Fatal("expected the same flag set to be returned for the same name")
	}

	fs3 := n.FlagSet("log")
	if fs1 == fs3 {
		t.Fatal("expected a different flag set for a different name")
	}

	// Adding a flag must not panic (regression: zero-value FlagSet with nil maps).
	fs1.String("addr", "", "bind address")
	if fs1.Lookup("addr") == nil {
		t.Fatal("expected addr flag to be registered")
	}
}

func TestAppLifecycleOrder(t *testing.T) {
	var order []string

	app := NewApp("test", "test app",
		WithSilence(),
		WithNoConfig(),
		WithRun(func(ctx context.Context) error {
			order = append(order, "run")
			return nil
		}),
		WithPreRunHook(func(ctx context.Context) error {
			order = append(order, "preRun")
			return nil
		}),
		WithPostRunHook(func(ctx context.Context) error {
			order = append(order, "postRun")
			return nil
		}),
		WithPreShutdownHook(func(ctx context.Context) error {
			order = append(order, "preShutdown")
			return nil
		}),
	)

	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("RunContext failed: %v", err)
	}

	want := []string{"preRun", "run", "postRun", "preShutdown"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("lifecycle order = %v, want %v", order, want)
	}
}

func TestRunContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	returned := make(chan error, 1)

	app := NewApp("test", "test",
		WithSilence(),
		WithNoConfig(),
		WithRun(func(rctx context.Context) error {
			close(started)
			<-rctx.Done()
			return nil
		}),
	)

	go func() { returned <- app.RunContext(ctx) }()

	<-started
	cancel()

	select {
	case err := <-returned:
		if err != nil {
			t.Fatalf("RunContext failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunContext did not return after context cancellation")
	}
}

func TestConfigFileLoading(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "testconfig.yaml"),
		[]byte("level: debug\nport: 9090\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	opts := &testOptions{}
	app := NewApp("test", "test",
		WithSilence(),
		WithOptions(opts),
		WithConfigSearchPaths([]string{dir}),
		WithConfigName("testconfig"),
		WithRun(func(ctx context.Context) error { return nil }),
	)

	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("RunContext failed: %v", err)
	}

	if opts.Level != "debug" {
		t.Fatalf("level = %q, want debug", opts.Level)
	}
	if opts.Port != 9090 {
		t.Fatalf("port = %d, want 9090", opts.Port)
	}
}

func TestWithConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "myapp.yaml"),
		[]byte("level: debug\nport: 9090\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	opts := &testOptions{}
	app := NewApp("test", "test",
		WithSilence(),
		WithOptions(opts),
		WithConfig(dir, "myapp"),
		WithRun(func(ctx context.Context) error { return nil }),
	)

	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("RunContext failed: %v", err)
	}

	if opts.Level != "debug" {
		t.Fatalf("level = %q, want debug", opts.Level)
	}
	if opts.Port != 9090 {
		t.Fatalf("port = %d, want 9090", opts.Port)
	}
}

func TestEffectiveConfigSearchPathsWithDirInHome(t *testing.T) {
	t.Setenv("HOME", "/home/test")

	// Default: uses $HOME/.<name>.
	app := NewApp("test", "test")
	want := []string{".", filepath.Join("/home/test", ".test"), filepath.Join("/etc", "test")}
	if got := app.effectiveConfigSearchPaths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("default search paths = %v, want %v", got, want)
	}

	// WithDirInHome: uses $HOME/<dir>.
	app = NewApp("test", "test", WithDirInHome(".onexai"))
	want = []string{".", filepath.Join("/home/test", ".onexai"), filepath.Join("/etc", "test")}
	if got := app.effectiveConfigSearchPaths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("search paths with dirInHome = %v, want %v", got, want)
	}
}

func TestConfigFileLoadingFromDirInHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".onexai")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "dirinhome.yaml"),
		[]byte("level: debug\nport: 9090\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	opts := &testOptions{}
	app := NewApp("dirinhome", "test",
		WithSilence(),
		WithOptions(opts),
		WithDirInHome(".onexai"),
		WithRun(func(ctx context.Context) error { return nil }),
	)

	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("RunContext failed: %v", err)
	}

	if opts.Level != "debug" {
		t.Fatalf("level = %q, want debug", opts.Level)
	}
	if opts.Port != 9090 {
		t.Fatalf("port = %d, want 9090", opts.Port)
	}
}

func TestFlagPrecedenceOverConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "testconfig.yaml"),
		[]byte("level: debug\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	opts := &testOptions{}
	app := NewApp("test", "test",
		WithSilence(),
		WithOptions(opts),
		WithConfigSearchPaths([]string{dir}),
		WithConfigName("testconfig"),
		WithRun(func(ctx context.Context) error { return nil }),
	)

	app.Command().SetArgs([]string{"--level=error"})

	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("RunContext failed: %v", err)
	}

	if opts.Level != "error" {
		t.Fatalf("level = %q, want error (flag must override config file)", opts.Level)
	}
}

func TestSingleRunGuard(t *testing.T) {
	app := NewApp("test", "test",
		WithSilence(),
		WithNoConfig(),
		WithRun(func(ctx context.Context) error { return nil }),
	)

	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("first RunContext failed: %v", err)
	}
	if err := app.RunContext(context.Background()); err == nil {
		t.Fatal("expected an error on the second RunContext call")
	}
}

func TestHealthServerLifecycle(t *testing.T) {
	h := newHealthServer("127.0.0.1:0", "/healthz", false)

	if err := h.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestHealthServerHandler(t *testing.T) {
	h := newHealthServer("127.0.0.1:0", "/healthz", true)
	srv := httptest.NewServer(h.server.Handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200", resp.StatusCode)
	}

	// pprof must be mounted when the profiler is enabled.
	resp, err = http.Get(srv.URL + "/debug/pprof/")
	if err != nil {
		t.Fatalf("GET /debug/pprof/ failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /debug/pprof/ status = %d, want 200", resp.StatusCode)
	}
}

func TestDefaultHealthCheckUsesOptions(t *testing.T) {
	opts := genericoptions.NewServerOptions()
	opts.Health.HealthCheckAddress = "127.0.0.1:0"
	opts.Health.HealthCheckPath = "/custom-healthz"

	app := NewApp("test", "test",
		WithSilence(),
		WithNoConfig(),
		WithOptions(opts),
		WithDefaultHealthCheck(),
		WithRun(func(ctx context.Context) error { return nil }),
	)

	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("RunContext failed: %v", err)
	}
}

// recordingShutdowner records the order in which Shutdown is called.
type recordingShutdowner struct {
	name  string
	order *[]string
}

func (s *recordingShutdowner) Shutdown(ctx context.Context) error {
	*s.order = append(*s.order, s.name)
	return nil
}

func TestMultipleLifecycleHooks(t *testing.T) {
	var order []string

	app := NewApp("test", "test",
		WithSilence(),
		WithNoConfig(),
		WithRun(func(ctx context.Context) error {
			order = append(order, "run")
			return nil
		}),
		WithPreRunHook(func(ctx context.Context) error {
			order = append(order, "preRun1")
			return nil
		}),
		WithPreRunHook(func(ctx context.Context) error {
			order = append(order, "preRun2")
			return nil
		}),
		WithPostRunHook(func(ctx context.Context) error {
			order = append(order, "postRun1")
			return nil
		}),
		WithPostRunHook(func(ctx context.Context) error {
			order = append(order, "postRun2")
			return nil
		}),
		WithPreShutdownHook(func(ctx context.Context) error {
			order = append(order, "preShutdown1")
			return nil
		}),
		WithPreShutdownHook(func(ctx context.Context) error {
			order = append(order, "preShutdown2")
			return nil
		}),
	)

	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("RunContext failed: %v", err)
	}

	want := []string{
		"preRun1", "preRun2",
		"run",
		"postRun1", "postRun2",
		"preShutdown1", "preShutdown2",
	}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("lifecycle order = %v, want %v", order, want)
	}
}

func TestShutdownersStopInLIFOOrder(t *testing.T) {
	var order []string

	app := NewApp("test", "test",
		WithSilence(),
		WithNoConfig(),
		WithRun(func(ctx context.Context) error { return nil }),
	)

	app.registerShutdowner(&recordingShutdowner{name: "first", order: &order})
	app.registerShutdowner(&recordingShutdowner{name: "second", order: &order})
	app.registerShutdowner(&recordingShutdowner{name: "third", order: &order})

	if err := app.RunContext(context.Background()); err != nil {
		t.Fatalf("RunContext failed: %v", err)
	}

	want := []string{"third", "second", "first"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("shutdown order = %v, want %v", order, want)
	}
}

// TestReloadOffersAFreshConfigWithoutTouchingLiveOptions pins the contract a
// reload is built on: it hands the callback a configuration derived from the
// sources, and leaves the options the RunFunc is holding alone.
//
// Writing the new configuration back into the live options is the obvious
// implementation and the one this replaced. It is a data race — the RunFunc's
// goroutine reads those fields without a lock — and it cannot work for most
// settings anyway, since connections are built once from values copied out of
// the options at startup. So the test asserts the absence of that write, not
// just the presence of the callback.
func TestReloadOffersAFreshConfigWithoutTouchingLiveOptions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "testconfig.yaml"),
		[]byte("level: debug\nport: 9090\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	opts := &testOptions{}
	var got *viper.Viper
	app := NewApp("test", "test",
		WithSilence(),
		WithOptions(opts),
		WithConfigSearchPaths([]string{dir}),
		WithConfigName("testconfig"),
		WithConfigChange(func(v *viper.Viper) error {
			got = v
			return nil
		}),
		WithRun(func(ctx context.Context) error { return nil }),
	)

	// Load and apply the initial config.
	if err := app.loadConfig(); err != nil {
		t.Fatal(err)
	}
	if err := app.applyOptions(); err != nil {
		t.Fatal(err)
	}
	if opts.Level != "debug" || opts.Port != 9090 {
		t.Fatalf("initial load: level=%q port=%d, want debug/9090", opts.Level, opts.Port)
	}

	// The file changes, and a remote tree arrives with a value that beats the
	// file but not the environment.
	if err := os.WriteFile(
		filepath.Join(dir, "testconfig.yaml"),
		[]byte("level: warn\nport: 9090\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	if err := app.reloadConfig(map[string]any{"level": "error"}); err != nil {
		t.Fatal(err)
	}

	if got == nil {
		t.Fatal("the change callback was not invoked")
	}
	if v := got.GetString("level"); v != "error" {
		t.Fatalf("reloaded level = %q, want error (remote beats the file)", v)
	}
	if v := got.GetInt("port"); v != 9090 {
		t.Fatalf("reloaded port = %d, want 9090 (from the file)", v)
	}

	// The live options are untouched, because nothing owns the right to write
	// them while the RunFunc may be reading them.
	if opts.Level != "debug" || opts.Port != 9090 {
		t.Fatalf("live options were mutated by the reload: level=%q port=%d, want debug/9090",
			opts.Level, opts.Port)
	}

	// Read the file valueless: the reload must not be observing the app's live
	// viper instance either.
	if got == app.v {
		t.Fatal("the callback received the app's live viper instance instead of a fresh one")
	}
}
