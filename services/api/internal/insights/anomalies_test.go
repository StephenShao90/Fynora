package insights

import (
	"testing"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

func TestDetectsMissingPayoutUnmatchedDepositAndHighFees(t *testing.T) {
	now := time.Now().UTC()
	anoms := DetectAnomalies(
		[]models.Payout{{ID: "po", Amount: 100, Currency: "CAD", ExpectedArrivalAt: now.AddDate(0, 0, -5)}},
		[]models.BankTransaction{{ID: "bank", Amount: 212, Currency: "CAD", Direction: "credit", PostedAt: now}},
		[]models.Fee{{Amount: 20}},
		nil,
	)
	seen := map[string]bool{}
	for _, a := range anoms {
		seen[a.Type] = true
	}
	for _, typ := range []string{"missing_payout", "unmatched_deposit", "high_processor_fee"} {
		if !seen[typ] {
			t.Fatalf("expected anomaly %s in %#v", typ, anoms)
		}
	}
}
