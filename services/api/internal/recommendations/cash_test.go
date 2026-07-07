package recommendations

import (
	"testing"

	"github.com/StephenShao90/Fynora/services/api/internal/insights"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

func TestReserveRecommendationUsesOutflow(t *testing.T) {
	recs := Cash([]models.BankTransaction{{Amount: 42, Direction: "debit"}}, []insights.Anomaly{{Type: "unmatched_deposit"}}, nil, nil, "CAD")
	if len(recs) < 2 || recs[0].Type != "reserve" || recs[0].AmountMinor != 4200 {
		t.Fatalf("unexpected recommendations: %#v", recs)
	}
}
