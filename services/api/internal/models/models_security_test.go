package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSensitiveModelFieldsAreNeverSerialized(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		blocked []string
	}{
		{
			name:    "user password hash",
			value:   User{ID: "user_1", Email: "owner@example.com", PasswordHash: "bcrypt-password-hash"},
			blocked: []string{"PasswordHash", "password_hash", "bcrypt-password-hash"},
		},
		{
			name:    "plaid token ciphertext",
			value:   PlaidConnection{ID: "conn_1", AccessTokenCiphertext: "plaid-token-ciphertext"},
			blocked: []string{"AccessTokenCiphertext", "access_token", "plaid-token-ciphertext"},
		},
		{
			name:    "refresh token hash",
			value:   RefreshSession{ID: "session_1", TokenHash: "refresh-token-hash"},
			blocked: []string{"TokenHash", "token_hash", "refresh-token-hash"},
		},
		{
			name:    "provider token ciphertexts",
			value:   ProviderConnection{ID: "provider_1", AccessTokenCiphertext: "access-ciphertext", RefreshTokenCiphertext: "refresh-ciphertext"},
			blocked: []string{"AccessTokenCiphertext", "RefreshTokenCiphertext", "access_token", "refresh_token", "access-ciphertext", "refresh-ciphertext"},
		},
		{
			name:    "oauth state hash",
			value:   OAuthState{ID: "state_1", StateHash: "oauth-state-hash"},
			blocked: []string{"StateHash", "state_hash", "oauth-state-hash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			serialized := string(body)
			for _, blocked := range tt.blocked {
				if strings.Contains(serialized, blocked) {
					t.Fatalf("serialized sensitive field %q in %s", blocked, serialized)
				}
			}
		})
	}
}
