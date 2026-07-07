package plaid

import (
	"context"
	"net/http"
	"testing"
)

func TestConfigurableWebhookVerifierMockBypass(t *testing.T) {
	verifier := ConfigurableWebhookVerifier{Enabled: true, AppEnv: "development"}
	headers := http.Header{"X-Plaid-Mock-Webhook": []string{"true"}}
	if err := verifier.Verify(context.Background(), []byte(`{}`), headers); err != nil {
		t.Fatalf("expected dev mock bypass: %v", err)
	}
}

func TestConfigurableWebhookVerifierRejectsInvalid(t *testing.T) {
	verifier := ConfigurableWebhookVerifier{Enabled: true, AppEnv: "production"}
	if err := verifier.Verify(context.Background(), []byte(`{}`), http.Header{}); err == nil {
		t.Fatal("expected production verification error")
	}
}

func TestStaticJWTWebhookVerifier(t *testing.T) {
	verifier := StaticJWTWebhookVerifier{AllowedJWT: "valid.jwt.token"}
	if err := verifier.Verify(context.Background(), []byte(`{}`), http.Header{"Plaid-Verification": []string{"valid.jwt.token"}}); err != nil {
		t.Fatalf("expected valid JWT: %v", err)
	}
	if err := verifier.Verify(context.Background(), []byte(`{}`), http.Header{"Plaid-Verification": []string{"bad"}}); err == nil {
		t.Fatal("expected invalid JWT error")
	}
}
