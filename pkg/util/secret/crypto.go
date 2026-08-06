// Copyright 2025 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style license that can be
// found in the LICENSE file.
// The original repo for this file is https:///sre-gitlab.bitget.tools/srestack/opsassist.
// The professional version of this repository is https://github.com/onexstack/onex.

package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// encryptAES encrypts plaintext using AES-GCM.
// key must be 16 (AES-128), 24 (AES-192), or 32 (AES-256) bytes.
func encryptAES(key string, plaintext string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Prepend nonce to ciphertext for later decryption.
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptAES decrypts a Base64-encoded AES-GCM ciphertext.
// key must match the key used during encryption.
func decryptAES(key string, cryptoText string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", errors.New("decryption failed: integrity check failed or wrong key")
	}

	return string(plaintext), nil
}

// MustDecrypt is like Decrypt but panics on error.
// Useful for initialization paths where decryption failure should halt the program.
func MustDecrypt(key string, cryptoText string) string {
	decrypted, err := decryptAES(key, cryptoText)
	if err != nil {
		panic(err)
	}
	return decrypted
}

// TruncateSecret safely truncates a sensitive string for logging purposes.
// It returns the first 5 and last 5 characters separated by "...".
// If the string is too short, it masks the entire content.
func TruncateSecret(s string) string {
	if len(s) == 0 {
		return "<empty>"
	}
	if len(s) <= 15 {
		return "<masked>"
	}
	return s[:5] + "..." + s[len(s)-5:]
}
