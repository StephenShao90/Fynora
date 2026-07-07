package processors

import (
	"context"
	"net/http"
)

type SyncOptions struct {
	SinceCursor string
}

type SyncResult struct {
	ImportedCount int    `json:"imported_count"`
	Cursor        string `json:"cursor,omitempty"`
}

type WebhookResult struct {
	EventType       string `json:"event_type"`
	ExternalEventID string `json:"external_event_id"`
	ShouldSync      bool   `json:"should_sync"`
}

type PaymentProcessorProvider interface {
	ProviderName() string
	SyncPayouts(ctx context.Context, orgID string, opts SyncOptions) (*SyncResult, error)
	HandleWebhook(ctx context.Context, payload []byte, headers http.Header) (*WebhookResult, error)
}
