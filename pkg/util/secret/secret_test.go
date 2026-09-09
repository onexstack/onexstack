package secret

import (
	"os"
	"strings"
	"testing"
)

const testKey = "12345678901234567890123456789012" // 32 bytes for AES-256

func TestEncryptDecryptRoundtrip(t *testing.T) {
	plaintext := "hello, world!"
	ciphertext, err := Encrypt(plaintext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if ciphertext == "" {
		t.Fatal("ciphertext is empty")
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext equals plaintext")
	}

	decrypted, err := Decrypt(ciphertext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecryptEmptyPlaintext(t *testing.T) {
	plaintext := ""
	ciphertext, err := Encrypt(plaintext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Encrypt empty string failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Decrypt empty string failed: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("got %q, want empty", decrypted)
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	_, err := Decrypt("not-valid-base64", WithKey(testKey))
	if err == nil {
		t.Fatal("expected error for invalid Base64, got nil")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	plaintext := "sensitive data"
	ciphertext, err := Encrypt(plaintext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	wrongKey := "abcdefghijklmnopqrstuvwxyz123456"
	_, err = Decrypt(ciphertext, WithKey(wrongKey))
	if err == nil {
		t.Fatal("expected error for wrong key, got nil")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	plaintext := "tamper test"
	ciphertext, err := Encrypt(plaintext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Flip the last character
	tampered := ciphertext[:len(ciphertext)-1] + "A"
	if tampered == ciphertext {
		t.Skip("tampering had no effect, skip")
	}

	_, err = Decrypt(tampered, WithKey(testKey))
	if err == nil {
		t.Fatal("expected error for tampered ciphertext, got nil")
	}
}

func TestEncryptNoKey(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv(DefaultEnvKeyName)

	_, err := Encrypt("hello")
	if err == nil {
		t.Fatal("expected error when no key is provided, got nil")
	}
}

func TestEncryptWithEnvKey(t *testing.T) {
	os.Setenv(DefaultEnvKeyName, testKey)
	defer os.Unsetenv(DefaultEnvKeyName)

	plaintext := "env key test"
	ciphertext, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt with env key failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt with env key failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestWithKeyOverridesEnv(t *testing.T) {
	os.Setenv(DefaultEnvKeyName, "some-other-32-byte-key-override!")
	defer os.Unsetenv(DefaultEnvKeyName)

	plaintext := "override test"
	ciphertext, err := Encrypt(plaintext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Encrypt with explicit key failed: %v", err)
	}

	// Should decrypt with explicit key, not env key
	decrypted, err := Decrypt(ciphertext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}

	// Should fail with env-only (since ciphertext was encrypted with explicit key)
	decrypted2, err := Decrypt(ciphertext)
	if err == nil {
		t.Fatalf("expected error decrypting with wrong key, but got %q", decrypted2)
	}
}

func TestMustDecrypt(t *testing.T) {
	plaintext := "must decrypt test"
	ciphertext, err := Encrypt(plaintext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	result := MustDecrypt(testKey, ciphertext)
	if result != plaintext {
		t.Fatalf("MustDecrypt: got %q, want %q", result, plaintext)
	}
}

func TestMustDecryptPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic, but did not panic")
		}
	}()
	MustDecrypt("short", "not-valid")
}

func TestTruncateSecret(t *testing.T) {
	tests := []struct {
		name string
		input string
		want  string
	}{
		{"empty", "", "<empty>"},
		{"short", "abc", "<masked>"},
		{"exactly 15", "123456789012345", "<masked>"},
		{"16 chars", "1234567890123456", "12345...23456"},
		{"long string", "my-super-secret-api-key-that-is-very-long", "my-su...-long"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateSecret(tt.input)
			if got != tt.want {
				t.Errorf("TruncateSecret(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateAKSK(t *testing.T) {
	ak, sk := GenerateAKSK()

	if !strings.HasPrefix(ak, "ak-") {
		t.Errorf("AK should start with 'ak-', got %q", ak)
	}
	if !strings.HasPrefix(sk, "sk-") {
		t.Errorf("SK should start with 'sk-', got %q", sk)
	}

	// AK: "ak-" (3) + 60 hex chars (30 bytes * 2) = 63 total
	expectedAKLen := 3 + 30*2
	if len(ak) != expectedAKLen {
		t.Errorf("AK length: got %d, want %d (content: %q)", len(ak), expectedAKLen, ak)
	}
	// SK: "sk-" (3) + 120 hex chars (60 bytes * 2) = 123 total
	expectedSKLen := 3 + 60*2
	if len(sk) != expectedSKLen {
		t.Errorf("SK length: got %d, want %d (content: %q)", len(sk), expectedSKLen, sk)
	}

	// Uniqueness: multiple calls should produce different values
	ak2, sk2 := GenerateAKSK()
	if ak == ak2 {
		t.Error("two consecutive AK values should differ")
	}
	if sk == sk2 {
		t.Error("two consecutive SK values should differ")
	}
}

func TestEncryptDecryptChineseText(t *testing.T) {
	plaintext := "你好，世界！Hello, 世界!"
	ciphertext, err := Encrypt(plaintext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, WithKey(testKey))
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptKeyLengths(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"AES-128", "1234567890123456"},           // 16 bytes
		{"AES-192", "123456789012345678901234"},   // 24 bytes
		{"AES-256", "12345678901234567890123456789012"}, // 32 bytes
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := Encrypt("test", WithKey(tt.key))
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}
			decrypted, err := Decrypt(ciphertext, WithKey(tt.key))
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}
			if decrypted != "test" {
				t.Fatalf("got %q, want 'test'", decrypted)
			}
		})
	}
}
