package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPlaidConnectionDoesNotExposeAccessTokenCiphertext(t *testing.T) {
	row := PlaidConnection{ID: "conn_1", UserID: "user_1", ItemID: "item_1", InstitutionName: "Test Bank", AccessTokenCiphertext: "secret-ciphertext"}
	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-ciphertext") || strings.Contains(string(raw), "access_token") {
		t.Fatalf("Plaid token material leaked in JSON: %s", string(raw))
	}
}
