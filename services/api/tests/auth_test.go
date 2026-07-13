package tests

import (
	"testing"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
)

func TestJWTSignVerify(t *testing.T) {
	token, err := auth.SignJWT("secret", "u1", "demo@clearflow.dev", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.VerifyJWT("secret", token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "u1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestPasswordHash(t *testing.T) {
	hash, err := auth.HashPassword("demo-password")
	if err != nil {
		t.Fatal(err)
	}
	if !auth.CheckPassword(hash, "demo-password") {
		t.Fatal("password should match")
	}
}
