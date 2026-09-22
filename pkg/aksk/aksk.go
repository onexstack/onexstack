// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package aksk

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

const (
	// AccessKeyPrefix is the literal an access key starts with. A prefix makes
	// a credential recognisable in a log line or a paste — which is the whole
	// reason for spending three characters on it.
	AccessKeyPrefix = "ak-"
	// SecretKeyPrefix is the literal a secret key starts with.
	SecretKeyPrefix = "sk-"

	// accessKeyChars is the length of the random body of an access key, after
	// the prefix. 20 Base62 characters carry log2(62)*20 ≈ 119 bits, past any
	// collision concern for a table of access keys.
	accessKeyChars = 20
	// secretKeyChars is the length of the random body of a secret key, after
	// the prefix. 32 Base62 characters carry ≈ 190 bits, which is the half an
	// attacker would have to guess.
	secretKeyChars = 32
)

// alphabet is the Base62 character set: A-Z, a-z, 0-9.
//
// It deliberately excludes `-` and `_`, so a credential body can never be
// mistaken for, or confused with, the prefix separator.
var alphabet = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

// maxUsable is the largest byte value that can be reduced modulo len(alphabet)
// without bias: 256 / 62 == 4, so 4*62 == 248 values map evenly and the
// remaining 8 (248-255) are discarded. Without this, the first 8 characters of
// the alphabet would be ~1.6% more likely than the rest.
const maxUsable = 256 / 62 * 62

// Generate mints a fresh Access Key / Secret Key pair.
//
// The returned access key is AccessKeyPrefix followed by 20 Base62 characters
// (23 characters in total); the secret key is SecretKeyPrefix followed by 32
// Base62 characters (35 in total). Both halves come from crypto/rand.
//
// The error is reported rather than swallowed because the sole caller is a
// write path: a half-formed credential must not be persisted, and crypto/rand
// failing is not a condition a caller can work around by changing its input.
func Generate() (accessKey, secretKey string, err error) {
	if accessKey, err = GenerateAccessKey(); err != nil {
		return "", "", err
	}
	if secretKey, err = GenerateSecretKey(); err != nil {
		return "", "", err
	}
	return accessKey, secretKey, nil
}

// GenerateAccessKey mints the public half of a credential pair.
//
// The access key is an identifier, not a secret: it is stored and rendered in
// clear, so its length is chosen against collision, not against guessing.
func GenerateAccessKey() (string, error) {
	body, err := randomString(accessKeyChars)
	if err != nil {
		return "", err
	}
	return AccessKeyPrefix + body, nil
}

// GenerateSecretKey mints the secret half of a credential pair.
//
// This value is shown to the caller exactly once, at creation. A caller that
// stores it in clear puts a working credential in every dump of its config;
// that is the caller's decision to make, and Digest exists so it need not.
func GenerateSecretKey() (string, error) {
	body, err := randomString(secretKeyChars)
	if err != nil {
		return "", err
	}
	return SecretKeyPrefix + body, nil
}

// Digest is the one-way function a secret key is stored under.
//
// There is no salt, unlike a password: the input is 190 bits of crypto/rand
// output, so there is no dictionary to precompute against and the rainbow-table
// argument that forces a salt on a guessable secret does not apply.
//
// The digest is what a store keeps, so that presenting the secret can be
// checked without the store holding a working credential.
func Digest(secretKey string) string {
	sum := sha256.Sum256([]byte(secretKey))
	return hex.EncodeToString(sum[:])
}

// randomString returns n characters drawn uniformly from alphabet.
//
// Bytes at or above maxUsable are rejected rather than reduced, so every
// character in the set is equally likely. Rejection costs at most one extra
// read per 31 accepted bytes and removes the modulo bias entirely.
func randomString(n int) (string, error) {
	out := make([]byte, 0, n)
	buf := make([]byte, 32)
	for len(out) < n {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if int(b) >= maxUsable {
				continue
			}
			out = append(out, alphabet[int(b)%len(alphabet)])
			if len(out) == n {
				break
			}
		}
	}
	return string(out), nil
}
