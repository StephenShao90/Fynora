package tests

import (
	"testing"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/advisor"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

func TestSubscriptionDetection(t *testing.T) {
	rows := []models.Transaction{
		{Direction: "expense", NormalizedMerchant: "Netflix", Amount: 18.99, OccurredAt: time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)},
		{Direction: "expense", NormalizedMerchant: "Netflix", Amount: 18.99, OccurredAt: time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)},
	}
	got := advisor.DetectSubscriptions(rows)
	if len(got) != 1 || got[0].Merchant != "Netflix" {
		t.Fatalf("expected Netflix subscription, got %#v", got)
	}
}

func TestInvestmentProjection(t *testing.T) {
	p := advisor.InvestmentProjection(300, 1000, 10, "moderate")
	if len(p.Points) != 10 {
		t.Fatalf("expected 10 yearly points, got %d", len(p.Points))
	}
	if p.Points[9].Expected <= p.TotalContributions {
		t.Fatalf("expected growth above contributions")
	}
}

func TestEmergencyFund(t *testing.T) {
	profile := models.AdvisorProfile{EmergencyFundMonthsTarget: 6, CurrentEmergencyFund: 1000}
	rows := []models.Transaction{
		{Direction: "income", Amount: 4000, Category: "Income", OccurredAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
		{Direction: "expense", Amount: 1200, Category: "Housing", OccurredAt: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)},
		{Direction: "expense", Amount: 300, Category: "Groceries", OccurredAt: time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)},
	}
	plan := advisor.EmergencyFund(profile, rows)
	if plan.Target != 9000 {
		t.Fatalf("expected target 9000, got %.2f", plan.Target)
	}
}
