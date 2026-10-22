// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onexstack.

package app

import (
	"log/slog"
	"strings"
	"testing"
)

// TestPrintConfigRedactsCredentials pins that the configuration dump does not
// become a place credentials are written down.
//
// The dump is the diagnostic for "what did the service actually read"; turning
// it on by fixing its guard would otherwise have made every debug-level startup
// print the database password and the JWT signing key, in an environment that is
// usually shared and whose logs usually outlive the incident.
func TestPrintConfigRedactsCredentials(t *testing.T) {
	var buf strings.Builder
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(previous)

	opts := &testOptions{}
	app := NewApp("redacttest", "redacttest", WithSilence(), WithOptions(opts))

	// The marker makes the assertion independent of the key names: a value that
	// shares a substring with its own key ("tok" in internal-api-token) would
	// otherwise look like a leak that is not one.
	secrets := map[string]string{
		"postgresql.password":   "zz-secret-pg",
		"auth.key":              "zz-secret-signing",
		"auth.redis.password":   "zz-secret-redis",
		"sms.salt":              "zz-secret-salt",
		"capability-key":        "zz-secret-cap",
		"internal-api-token":    "zz-secret-token",
		"storage.secret-id":     "zz-secret-id",
		"authz.keto.access-key": "zz-secret-keto",
	}
	for key, value := range secrets {
		app.v.Set(key, value)
	}
	app.v.Set("metrics-addr", "0.0.0.0:9090")

	app.printConfig()

	out := buf.String()
	if strings.Contains(out, "zz-secret-") {
		t.Errorf("the configuration dump printed a credential:\n%s", out)
	}

	// The keys are still listed and the non-secret values still printed: the
	// dump exists to show what the service read, and redaction that removed the
	// whole line would answer nothing.
	for key := range secrets {
		if !strings.Contains(out, key) {
			t.Errorf("the dump dropped the key %q rather than redacting it:\n%s", key, out)
		}
	}
	if !strings.Contains(out, "0.0.0.0:9090") {
		t.Errorf("the dump dropped a non-secret value:\n%s", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Errorf("nothing was redacted, so the test is not exercising anything:\n%s", out)
	}
}

// TestNamesACredential covers the classifier directly, including the shapes that
// would only show up as a leaked credential in a log file.
func TestNamesACredential(t *testing.T) {
	secret := []string{
		"postgresql.password",
		"auth.key",
		"auth.redis.password",
		"authz.keto.token",
		"sms.salt",
		"capability-key",
		"storage.secret-id",
		"internal-api-token",
		"wechat.api_key",
		"mysql.dsn",
	}
	for _, key := range secret {
		if !namesACredential(key) {
			t.Errorf("namesACredential(%q) = false, want true", key)
		}
	}

	plain := []string{
		"mesh.service-name",
		"postgresql.port",
		"metrics-addr",
		"log.level",
		"registry.polaris.addr",
		"config-source.type",
	}
	for _, key := range plain {
		if namesACredential(key) {
			t.Errorf("namesACredential(%q) = true, want false", key)
		}
	}
}

// TestPrintConfigIsSkippedWhenConfigIsOff pins the guard that was inverted: the
// dump used to run only when config loading was disabled, which is to say never.
func TestPrintConfigIsSkippedWhenConfigIsOff(t *testing.T) {
	var buf strings.Builder
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(previous)

	app := NewApp("nocfgtest", "nocfgtest", WithSilence(), WithNoConfig())
	app.v.Set("level", "debug")
	app.printConfig()

	if buf.Len() != 0 {
		t.Errorf("printConfig logged while config loading is off:\n%s", buf.String())
	}
}
