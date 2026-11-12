package utils

import (
	"testing"
)

func TestHMACSM3HexStable(t *testing.T) {
	key := DeriveKey("secret", "audit-chain")
	a := HMACSM3Hex(key, "prev", "1", "event")
	b := HMACSM3Hex(key, "prev", "1", "event")
	if a != b || len(a) != 64 {
		t.Fatalf("unexpected hmac %s", a)
	}
	if HexEqual(a, HMACSM3Hex(key, "prev", "1", "eventX")) {
		t.Fatal("different parts must produce different mac")
	}
}

func TestPasswordKeySealRoundTrip(t *testing.T) {
	salt := []byte("0123456789abcdef")
	key, err := PasswordKey("export-pass", salt, PasswordKDFIterations)
	if err != nil {
		t.Fatal(err)
	}
	nonce, sealed, err := SealSM4GCM(key, []byte("hello-audit"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := OpenSM4GCM(key, nonce, sealed)
	if err != nil || string(plain) != "hello-audit" {
		t.Fatalf("round trip failed: %v %q", err, plain)
	}
	if _, err := OpenSM4GCM(key, nonce, append(sealed, 1)); err == nil {
		t.Fatal("tampered ciphertext must fail")
	}
}
