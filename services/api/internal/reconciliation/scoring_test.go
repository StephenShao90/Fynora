package reconciliation

import (
	"testing"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

func TestExactPayoutDepositMatchScoresHigh(t *testing.T) {
	now := time.Now().UTC()
	score := Score(models.Payout{ID: "po", Amount: 100, Currency: "CAD", Processor: "stripe", ExpectedArrivalAt: now}, models.BankTransaction{ID: "bank", Amount: 100, Currency: "CAD", Direction: "credit", Description: "STRIPE PAYOUT", PostedAt: now}, nil, nil)
	if score.Status != StatusMatched || score.ConfidenceScore < 0.9 {
		t.Fatalf("expected high exact match, got %#v", score)
	}
}

func TestDateShiftedMatchIsLikely(t *testing.T) {
	now := time.Now().UTC()
	score := Score(models.Payout{ID: "po", Amount: 100, Currency: "CAD", Processor: "stripe", ExpectedArrivalAt: now}, models.BankTransaction{ID: "bank", Amount: 100, Currency: "CAD", Direction: "credit", PostedAt: now.AddDate(0, 0, 4)}, nil, nil)
	if score.Status != StatusLikelyMatch && score.Status != StatusDateMismatch {
		t.Fatalf("expected likely/date mismatch, got %#v", score)
	}
}

func TestAmountAndCurrencyMismatchStatuses(t *testing.T) {
	now := time.Now().UTC()
	amount := Score(models.Payout{ID: "po", Amount: 100, Currency: "CAD", ExpectedArrivalAt: now}, models.BankTransaction{ID: "bank", Amount: 95, Currency: "CAD", Direction: "credit", PostedAt: now}, nil, nil)
	if amount.Status != StatusAmountMismatch {
		t.Fatalf("expected amount mismatch, got %#v", amount)
	}
	currency := Score(models.Payout{ID: "po", Amount: 100, Currency: "CAD", ExpectedArrivalAt: now}, models.BankTransaction{ID: "bank", Amount: 100, Currency: "USD", Direction: "credit", PostedAt: now}, nil, nil)
	if currency.Status != StatusCurrencyMismatch {
		t.Fatalf("expected currency mismatch, got %#v", currency)
	}
}
