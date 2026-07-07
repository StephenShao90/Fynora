package processors

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type StripeOAuthClient interface {
	ConnectURL(clientID, redirectURI, state string) string
	ExchangeCode(ctx context.Context, code string) (StripeOAuthAccount, error)
	Revoke(ctx context.Context, accountID, accessToken string) error
}

type StripeOAuthAccount struct {
	AccountID    string
	DisplayName  string
	AccessToken  string
	RefreshToken string
	Scope        string
}

type MockStripeOAuthClient struct{}

func (MockStripeOAuthClient) ConnectURL(clientID, redirectURI, state string) string {
	if clientID == "" {
		clientID = "ca_mock_clearflow"
	}
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", clientID)
	v.Set("scope", "read_write")
	v.Set("redirect_uri", redirectURI)
	v.Set("state", state)
	return "https://connect.stripe.com/oauth/authorize?" + v.Encode()
}

func (MockStripeOAuthClient) ExchangeCode(ctx context.Context, code string) (StripeOAuthAccount, error) {
	if code == "" {
		return StripeOAuthAccount{}, errors.New("missing authorization code")
	}
	suffix := code
	if len(suffix) > 8 {
		suffix = suffix[len(suffix)-8:]
	}
	return StripeOAuthAccount{
		AccountID:    "acct_mock_" + suffix,
		DisplayName:  "Stripe Test Account",
		AccessToken:  "sk_mock_" + suffix,
		RefreshToken: "rt_mock_" + suffix,
		Scope:        "read_write",
	}, nil
}

func (MockStripeOAuthClient) Revoke(ctx context.Context, accountID, accessToken string) error {
	return nil
}

type HTTPStripeOAuthClient struct {
	ClientID       string
	SecretKey      string
	RedirectURI    string
	HTTPClient     *http.Client
	TokenURL       string
	DeauthorizeURL string
}

func (c HTTPStripeOAuthClient) ConnectURL(clientID, redirectURI, state string) string {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", firstNonEmpty(clientID, c.ClientID))
	v.Set("scope", "read_write")
	v.Set("redirect_uri", firstNonEmpty(redirectURI, c.RedirectURI))
	v.Set("state", state)
	return "https://connect.stripe.com/oauth/authorize?" + v.Encode()
}

func (c HTTPStripeOAuthClient) ExchangeCode(ctx context.Context, code string) (StripeOAuthAccount, error) {
	if c.SecretKey == "" {
		return StripeOAuthAccount{}, errors.New("stripe secret key is required")
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.RedirectURI)
	var payload struct {
		StripeUserID string `json:"stripe_user_id"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}
	if err := c.postForm(ctx, firstNonEmpty(c.TokenURL, "https://connect.stripe.com/oauth/token"), form, &payload); err != nil {
		return StripeOAuthAccount{}, err
	}
	return StripeOAuthAccount{AccountID: payload.StripeUserID, DisplayName: "Stripe Account " + payload.StripeUserID, AccessToken: payload.AccessToken, RefreshToken: payload.RefreshToken, Scope: payload.Scope}, nil
}

func (c HTTPStripeOAuthClient) Revoke(ctx context.Context, accountID, accessToken string) error {
	if c.SecretKey == "" || accountID == "" {
		return errors.New("stripe deauthorization is not configured")
	}
	form := url.Values{}
	form.Set("client_id", c.ClientID)
	form.Set("stripe_user_id", accountID)
	var payload map[string]interface{}
	return c.postForm(ctx, firstNonEmpty(c.DeauthorizeURL, "https://connect.stripe.com/oauth/deauthorize"), form, &payload)
}

func (c HTTPStripeOAuthClient) postForm(ctx context.Context, endpoint string, form url.Values, out interface{}) error {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.SecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("stripe oauth request failed: status %d", res.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type StripeWebhookVerifier struct {
	Secret      string
	AppEnv      string
	AllowUnsafe bool
	Now         func() time.Time
}

func (v StripeWebhookVerifier) Verify(body []byte, signatureHeader string) error {
	if v.Secret == "" {
		if v.AppEnv == "production" && !v.AllowUnsafe {
			return errors.New("stripe webhook secret is required in production")
		}
		return nil
	}
	timestamp, signatures := parseStripeSignature(signatureHeader)
	if timestamp == "" || len(signatures) == 0 {
		return errors.New("missing Stripe-Signature")
	}
	if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		now := time.Now()
		if v.Now != nil {
			now = v.Now()
		}
		if delta := now.Sub(time.Unix(ts, 0)); delta > 5*time.Minute || delta < -5*time.Minute {
			return errors.New("stale Stripe-Signature timestamp")
		}
	}
	mac := hmac.New(sha256.New, []byte(v.Secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, sig := range signatures {
		if hmac.Equal([]byte(expected), []byte(sig)) {
			return nil
		}
	}
	return errors.New("invalid Stripe-Signature")
}

func parseStripeSignature(header string) (string, []string) {
	var timestamp string
	var signatures []string
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			signatures = append(signatures, value)
		}
	}
	return timestamp, signatures
}

func BuildStripeTestSignature(secret string, ts time.Time, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	payload := fmt.Sprintf("%d.%s", ts.Unix(), string(body))
	_, _ = mac.Write([]byte(payload))
	return fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(mac.Sum(nil)))
}

type StripeWebhookProvider struct {
	Verifier StripeWebhookVerifier
}

func (p StripeWebhookProvider) ProviderName() string { return "stripe" }

func (p StripeWebhookProvider) SyncPayouts(ctx context.Context, orgID string, opts SyncOptions) (*SyncResult, error) {
	return &SyncResult{ImportedCount: 0, Cursor: opts.SinceCursor}, nil
}

func (p StripeWebhookProvider) HandleWebhook(ctx context.Context, payload []byte, headers http.Header) (*WebhookResult, error) {
	if err := p.Verifier.Verify(payload, headers.Get("Stripe-Signature")); err != nil {
		return nil, err
	}
	var body struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	if body.ID == "" || body.Type == "" {
		return nil, errors.New("invalid Stripe webhook payload")
	}
	return &WebhookResult{EventType: body.Type, ExternalEventID: body.ID, ShouldSync: stripeEventShouldSync(body.Type)}, nil
}

func stripeEventShouldSync(eventType string) bool {
	switch eventType {
	case "payout.created", "payout.paid", "payout.failed", "charge.succeeded", "charge.refunded", "balance.available":
		return true
	default:
		return false
	}
}
