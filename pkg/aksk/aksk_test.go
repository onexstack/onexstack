// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package aksk

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerateShape(t *testing.T) {
	accessKey, secretKey, err := Generate()
	if err != nil {
		t.Fatalf("Generate() returned an error: %v", err)
	}

	accessKeyRE := regexp.MustCompile(`^ak-[0-9A-Za-z]{20}$`)
	secretKeyRE := regexp.MustCompile(`^sk-[0-9A-Za-z]{32}$`)

	if !accessKeyRE.MatchString(accessKey) {
		t.Errorf("access key %q does not match %s", accessKey, accessKeyRE)
	}
	if !secretKeyRE.MatchString(secretKey) {
		t.Errorf("secret key %q does not match %s", secretKey, secretKeyRE)
	}
}

func TestGenerateLengths(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		wantLen  int
		prefixCR string
	}{
		// "ak-" is 3 characters, plus 20 random ones.
		{name: "access key", wantLen: 3 + accessKeyChars, prefixCR: AccessKeyPrefix},
		// "sk-" is 3 characters, plus 32 random ones.
		{name: "secret key", wantLen: 3 + secretKeyChars, prefixCR: SecretKeyPrefix},
	}

	accessKey, secretKey, err := Generate()
	if err != nil {
		t.Fatalf("Generate() returned an error: %v", err)
	}
	tests[0].got = accessKey
	tests[1].got = secretKey

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.got) != tt.wantLen {
				t.Errorf("%s length = %d, want %d (%q)", tt.name, len(tt.got), tt.wantLen, tt.got)
			}
			if !strings.HasPrefix(tt.got, tt.prefixCR) {
				t.Errorf("%s %q does not start with %q", tt.name, tt.got, tt.prefixCR)
			}
		})
	}
}

// TestGenerateUniqueness is a smoke test, not a proof: it catches a generator
// that is accidentally seeded once or constant, which is the realistic failure
// mode. Real collisions are covered by the length budget, not by this.
func TestGenerateUniqueness(t *testing.T) {
	const runs = 100

	seenAccess := make(map[string]bool, runs)
	seenSecret := make(map[string]bool, runs)
	for i := 0; i < runs; i++ {
		accessKey, secretKey, err := Generate()
		if err != nil {
			t.Fatalf("Generate() returned an error: %v", err)
		}
		if seenAccess[accessKey] {
			t.Fatalf("access key %q repeated after %d runs", accessKey, i)
		}
		if seenSecret[secretKey] {
			t.Fatalf("secret key %q repeated after %d runs", secretKey, i)
		}
		seenAccess[accessKey] = true
		seenSecret[secretKey] = true
	}
}

// TestGenerateBodyExcludesSeparators pins the property the alphabet choice
// exists for: a body character can never be confused with the prefix
// separator, so a credential splits into exactly two parts on "-".
func TestGenerateBodyExcludesSeparators(t *testing.T) {
	for i := 0; i < 50; i++ {
		accessKey, secretKey, err := Generate()
		if err != nil {
			t.Fatalf("Generate() returned an error: %v", err)
		}
		for _, cred := range []string{accessKey, secretKey} {
			if got := strings.Count(cred, "-"); got != 1 {
				t.Fatalf("credential %q contains %d separators, want exactly 1", cred, got)
			}
			if strings.Contains(cred, "_") {
				t.Fatalf("credential %q contains a character outside Base62", cred)
			}
		}
	}
}

func TestDigest(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			// The published SHA-256 test vector, so this test fails if the
			// function is ever changed to a different hash or encoding.
			name:  "known vector",
			input: "abc",
			want:  "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		},
		{
			name:  "empty input",
			input: "",
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Digest(tt.input); got != tt.want {
				t.Errorf("Digest(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDigestIsDeterministicAndDistinct(t *testing.T) {
	accessKey, secretKey, err := Generate()
	if err != nil {
		t.Fatalf("Generate() returned an error: %v", err)
	}
	_ = accessKey

	if Digest(secretKey) != Digest(secretKey) {
		t.Error("Digest is not deterministic for the same input")
	}
	if Digest(secretKey) == Digest(secretKey+"x") {
		t.Error("Digest collided for two different inputs")
	}
	if len(Digest(secretKey)) != 64 {
		t.Errorf("Digest length = %d, want 64 hex characters", len(Digest(secretKey)))
	}
}
