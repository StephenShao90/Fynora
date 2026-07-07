package insights

import (
	"testing"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

func TestSpendingSummarizesCategories(t *testing.T) {
	now := time.Now().UTC()
	result := Spending([]models.BankTransaction{
		{Amount: 800, Direction: "debit", Description: "Software SaaS", PostedAt: now},
		{Amount: 200, Direction: "debit", Description: "Venue rental", PostedAt: now},
	}, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1), "CAD")
	if result.TotalSpendMinor != 100000 || len(result.Categories) == 0 || result.Categories[0].Category != "software" {
		t.Fatalf("unexpected spending insight: %#v", result)
	}
}
