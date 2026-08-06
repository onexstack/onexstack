// Copyright 2025 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style license that can be
// found in the LICENSE file.
// The original repo for this file is https:///sre-gitlab.bitget.tools/srestack/opsassist.
// The professional version of this repository is https://github.com/onexstack/onex.

// Package secretutil provides encryption/decryption and AKSK generation utilities.
// Key is supplied via the functional options pattern:
//
//	// Explicit key
//	secretutil.Encrypt("hello", secretutil.WithKey("my-key"))
//
//	// Key from env var (falls back to ENCRYPT_KEY if WithKey is not set)
//	secretutil.Decrypt(ciphertext)
//
//	// Explicit key overrides env var
//	secretutil.Encrypt("hello", secretutil.WithKey("my-key"))
package secret

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

// DefaultEnvKeyName is the default environment variable name for the encrypt key.
const DefaultEnvKeyName = "ENCRYPT_KEY"

// Option is a functional option for configuring the encrypt/decrypt key.
type Option func(*options)

type options struct {
	key      string
	keyIsSet bool
}

// WithKey sets the encryption/decryption key explicitly.
// This takes highest priority and overrides any env var lookup.
func WithKey(key string) Option {
	return func(o *options) {
		o.key = key
		o.keyIsSet = true
	}
}

// resolveKey resolves the final key using the following priority:
//  1. Explicitly set key via WithKey (if non-empty)
//  2. ENCRYPT_KEY environment variable
//  3. Error if neither provides a key
func (o *options) resolveKey() (string, error) {
	if o.keyIsSet && o.key != "" {
		return o.key, nil
	}

	if envVal := os.Getenv(DefaultEnvKeyName); envVal != "" {
		return envVal, nil
	}

	return "", fmt.Errorf("no encryption key provided: use WithKey() or set %s environment variable", DefaultEnvKeyName)
}

// Encrypt encrypts plaintext using AES-GCM.
// If no key is provided via opts, defaults to the ENCRYPT_KEY env var.
func Encrypt(plaintext string, opts ...Option) (string, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	key, err := o.resolveKey()
	if err != nil {
		return "", err
	}

	return encryptAES(key, plaintext)
}

// Decrypt decrypts a Base64-encoded AES-GCM ciphertext.
// If no key is provided via opts, defaults to the ENCRYPT_KEY env var.
func Decrypt(cryptoText string, opts ...Option) (string, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	key, err := o.resolveKey()
	if err != nil {
		return "", err
	}

	return decryptAES(key, cryptoText)
}

// GenerateAKSK generates a random AccessKey/SecretKey pair with standard prefixes.
// - AK format: ak-<30-char-random> (Total: 33 chars)
// - SK format: sk-<60-char-random> (Total: 63 chars)
func GenerateAKSK() (ak, sk string) {
    ak = "ak-" + generateRandomString(30)
    sk = "sk-" + generateRandomString(60)
    return ak, sk
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
