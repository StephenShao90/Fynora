package security

import (
	"context"
	"testing"
)

func TestAESGCMTokenProtectorRoundTrip(t *testing.T) {
	protector, err := NewAESGCMTokenProtector("test-encryption-key")
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := protector.Protect(context.Background(), "stripe-secret-fixture")
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == "stripe-secret-fixture" {
		t.Fatal("expected ciphertext to differ from plaintext")
	}
	plain, err := protector.Unprotect(context.Background(), ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "stripe-secret-fixture" {
		t.Fatalf("expected round trip token, got %q", plain)
	}
}

func TestTokenProtectorProductionRequiresKey(t *testing.T) {
	if _, err := NewTokenProtector("production", ""); err == nil {
		t.Fatal("expected production key error")
	}
	if _, err := NewTokenProtector("development", ""); err != nil {
		t.Fatalf("expected development fallback protector: %v", err)
	}
}
