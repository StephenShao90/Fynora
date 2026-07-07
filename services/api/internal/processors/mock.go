package processors

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type MockProvider struct {
	Name string
}

func (p MockProvider) ProviderName() string {
	if p.Name == "" {
		return "mock"
	}
	return p.Name
}

func (p MockProvider) SyncPayouts(ctx context.Context, orgID string, opts SyncOptions) (*SyncResult, error) {
	return &SyncResult{ImportedCount: 0, Cursor: opts.SinceCursor}, nil
}

func (p MockProvider) HandleWebhook(ctx context.Context, payload []byte, headers http.Header) (*WebhookResult, error) {
	var body map[string]interface{}
	_ = json.Unmarshal(payload, &body)
	eventType, _ := body["type"].(string)
	externalID, _ := body["id"].(string)
	if externalID == "" {
		sum := sha256.Sum256(payload)
		externalID = hex.EncodeToString(sum[:])
	}
	return &WebhookResult{EventType: eventType, ExternalEventID: externalID, ShouldSync: true}, nil
}
