package processors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStripeWebhookSignatureVerification(t *testing.T) {
	body := []byte(`{"id":"evt_1","type":"payout.paid"}`)
	now := time.Unix(1000, 0)
	verifier := StripeWebhookVerifier{Secret: "whsec_test", AppEnv: "production", Now: func() time.Time { return now }}
	sig := BuildStripeTestSignature("whsec_test", now, body)
	if err := verifier.Verify(body, sig); err != nil {
		t.Fatalf("expected valid signature: %v", err)
	}
	if err := verifier.Verify(body, "t=1000,v1=bad"); err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func TestStripeWebhookProviderSyncEvents(t *testing.T) {
	body := []byte(`{"id":"evt_1","type":"charge.succeeded"}`)
	now := time.Unix(1000, 0)
	provider := StripeWebhookProvider{Verifier: StripeWebhookVerifier{Secret: "whsec_test", Now: func() time.Time { return now }}}
	headers := http.Header{"Stripe-Signature": []string{BuildStripeTestSignature("whsec_test", now, body)}}
	result, err := provider.HandleWebhook(nil, body, headers)
	if err != nil {
		t.Fatal(err)
	}
	if !result.ShouldSync || result.ExternalEventID != "evt_1" {
		t.Fatalf("unexpected webhook result: %#v", result)
	}
}

func TestHTTPStripeOAuthClientExchangeAndRevoke(t *testing.T) {
	var sawToken, sawRevoke bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			sawToken = true
			_ = r.ParseForm()
			if r.Form.Get("code") != "ac_test" {
				t.Fatalf("unexpected code %q", r.Form.Get("code"))
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"stripe_user_id": "acct_123", "access_token": "sk_test", "refresh_token": "rt_test", "scope": "read_write"})
		case "/oauth/deauthorize":
			sawRevoke = true
			_ = json.NewEncoder(w).Encode(map[string]bool{"livemode": false})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := HTTPStripeOAuthClient{ClientID: "ca_test", SecretKey: "sk_secret", RedirectURI: "http://localhost/callback", TokenURL: server.URL + "/oauth/token", DeauthorizeURL: server.URL + "/oauth/deauthorize", HTTPClient: server.Client()}
	account, err := client.ExchangeCode(context.Background(), "ac_test")
	if err != nil {
		t.Fatal(err)
	}
	if account.AccountID != "acct_123" || account.AccessToken != "sk_test" {
		t.Fatalf("unexpected account: %#v", account)
	}
	if err := client.Revoke(context.Background(), account.AccountID, account.AccessToken); err != nil {
		t.Fatal(err)
	}
	if !sawToken || !sawRevoke {
		t.Fatalf("expected token and revoke requests, token=%v revoke=%v", sawToken, sawRevoke)
	}
}
