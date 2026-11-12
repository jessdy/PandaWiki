package utils

import (
	"strings"
	"sync"
	"testing"
)

func TestFieldCipherRoundTrip(t *testing.T) {
	fc, err := NewFieldCipher("primary-secret")
	if err != nil {
		t.Fatal(err)
	}
	enc, err := fc.Encrypt("sk-test-123")
	if err != nil {
		t.Fatal(err)
	}
	if !IsFieldEncrypted(enc) || strings.Contains(enc, "sk-test-123") {
		t.Fatalf("value not encrypted: %q", enc)
	}
	plain, err := fc.Decrypt(enc)
	if err != nil || plain != "sk-test-123" {
		t.Fatalf("decrypt = %q, %v", plain, err)
	}
	if empty, _ := fc.Encrypt(""); empty != "" {
		t.Fatalf("empty input should stay empty, got %q", empty)
	}
}

func TestFieldCipherLegacyPlaintextAndRotation(t *testing.T) {
	old, _ := NewFieldCipher("jwt-secret")
	encOld, _ := old.Encrypt("sk-old")

	rotated, _ := NewFieldCipher("data-key", "jwt-secret")
	if plain, err := rotated.Decrypt(encOld); err != nil || plain != "sk-old" {
		t.Fatalf("fallback key decrypt = %q, %v", plain, err)
	}
	if !rotated.NeedsReencrypt(encOld) {
		t.Fatal("value encrypted with fallback key should need re-encryption")
	}
	encNew, _ := rotated.Encrypt("sk-old")
	if rotated.NeedsReencrypt(encNew) {
		t.Fatal("value encrypted with primary key should not need re-encryption")
	}

	if plain, err := rotated.Decrypt("legacy-plain"); err != nil || plain != "legacy-plain" {
		t.Fatalf("legacy plaintext = %q, %v", plain, err)
	}
	if !rotated.NeedsReencrypt("legacy-plain") || rotated.NeedsReencrypt("") {
		t.Fatal("legacy plaintext should need encryption, empty should not")
	}

	other, _ := NewFieldCipher("other")
	if _, err := other.Decrypt(encOld); err == nil {
		t.Fatal("decrypt with wrong key should fail")
	}
	if _, err := other.Decrypt("sm4:not-base64!"); err == nil {
		t.Fatal("malformed ciphertext should fail")
	}
}

func TestFieldCipherConcurrent(t *testing.T) {
	fc, _ := NewFieldCipher("concurrent")
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				enc, err := fc.Encrypt("sk-concurrent")
				if err != nil {
					t.Error(err)
					return
				}
				if plain, err := fc.Decrypt(enc); err != nil || plain != "sk-concurrent" {
					t.Errorf("decrypt = %q, %v", plain, err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestVerifySecret(t *testing.T) {
	hash, err := HashSecret("Passw0rd!")
	if err != nil {
		t.Fatal(err)
	}
	if !IsBcryptHash(hash) {
		t.Fatalf("not a bcrypt hash: %q", hash)
	}
	if !VerifySecret(hash, "Passw0rd!") || VerifySecret(hash, "wrong") {
		t.Fatal("bcrypt verification mismatch")
	}
	if !VerifySecret("legacy", "legacy") || VerifySecret("legacy", "other") {
		t.Fatal("legacy plaintext verification mismatch")
	}
	if VerifySecret("", "") || VerifySecret(hash, "") {
		t.Fatal("empty values must not verify")
	}
}
