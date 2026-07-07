package plaid

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type PlaidWebhookVerifier interface {
	Verify(ctx context.Context, body []byte, headers http.Header) error
}

type ConfigurableWebhookVerifier struct {
	Enabled bool
	AppEnv  string
	Real    PlaidWebhookVerifier
}

func (v ConfigurableWebhookVerifier) Verify(ctx context.Context, body []byte, headers http.Header) error {
	if !v.Enabled {
		return nil
	}
	if v.Real != nil {
		return v.Real.Verify(ctx, body, headers)
	}
	if v.AppEnv != "production" && strings.EqualFold(headers.Get("X-Plaid-Mock-Webhook"), "true") {
		return nil
	}
	return errors.New("invalid Plaid webhook verification")
}

type StaticJWTWebhookVerifier struct {
	AllowedJWT string
}

func (v StaticJWTWebhookVerifier) Verify(ctx context.Context, body []byte, headers http.Header) error {
	got := headers.Get("Plaid-Verification")
	if got == "" {
		got = headers.Get("Plaid-Verification-Jwt")
	}
	if got == "" || got != v.AllowedJWT {
		return errors.New("invalid Plaid webhook JWT")
	}
	return nil
}

func WebhookVerificationEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "required", "enabled":
		return true
	default:
		return false
	}
}
