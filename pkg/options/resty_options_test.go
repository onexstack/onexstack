// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package options

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"resty.dev/v3"
)

// containsError reports whether any error in errs contains substr.
func containsError(errs []error, substr string) bool {
	for _, err := range errs {
		if strings.Contains(err.Error(), substr) {
			return true
		}
	}
	return false
}

func TestRestyOptionsValidateEndpointRequired(t *testing.T) {
	opts := NewRestyOptions()
	opts.AddFlags(pflag.NewFlagSet("test", pflag.ContinueOnError), "resty")
	opts.Endpoint = ""

	if errs := opts.Validate(); !containsError(errs, "--resty.endpoint is required") {
		t.Fatalf("expected endpoint required error, got %v", errs)
	}
}

func TestRestyOptionsValidateSecretPair(t *testing.T) {
	opts := NewRestyOptions()
	opts.AddFlags(pflag.NewFlagSet("test", pflag.ContinueOnError), "resty")
	opts.Endpoint = "http://example.com"
	opts.SecretID = "id"
	opts.SecretKey = ""

	if errs := opts.Validate(); !containsError(errs, "both --resty.secret-id and --resty.secret-key must be provided together") {
		t.Fatalf("expected secret pair error, got %v", errs)
	}
}

func TestRestyOptionsValidateBasicAuthPair(t *testing.T) {
	opts := NewRestyOptions()
	opts.AddFlags(pflag.NewFlagSet("test", pflag.ContinueOnError), "resty")
	opts.Endpoint = "http://example.com"
	opts.Username = "admin"
	opts.Password = ""

	if errs := opts.Validate(); !containsError(errs, "both --resty.username and --resty.password must be provided together") {
		t.Fatalf("expected basic auth pair error, got %v", errs)
	}
}

func TestRestyOptionsNewClientCached(t *testing.T) {
	opts := NewRestyOptions()
	opts.Endpoint = "http://example.com"

	c1 := opts.NewClient()
	c2 := opts.NewClient()

	// NewClient 应返回同一个缓存的 client，避免重复建 client 与重复注册中间件.
	if c1 != c2 {
		t.Fatal("NewClient should return the same cached client instance")
	}
}

func TestRestyOptionsAuthMiddlewareNotAccumulated(t *testing.T) {
	opts := NewRestyOptions()
	opts.Endpoint = "http://example.com"

	before := len(opts.middlewares)

	// 多次 applyToClient（如反复 NewClient 的场景）不应累积 auth 中间件.
	opts.applyToClient(resty.New())
	opts.applyToClient(resty.New())

	if len(opts.middlewares) != before {
		t.Fatalf("middlewares accumulated: before=%d after=%d", before, len(opts.middlewares))
	}
}
