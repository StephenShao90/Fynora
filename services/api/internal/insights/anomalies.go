package insights

import (
	"math"
	"sort"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

type Anomaly struct {
	ID                string    `json:"id"`
	Type              string    `json:"type"`
	Severity          string    `json:"severity"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	ResourceType      string    `json:"resourceType"`
	ResourceID        string    `json:"resourceId"`
	DetectedAt        time.Time `json:"detectedAt"`
	RecommendedAction string    `json:"recommendedAction"`
}

func DetectAnomalies(payouts []models.Payout, deposits []models.BankTransaction, fees []models.Fee, refunds []models.Refund) []Anomaly {
	now := time.Now().UTC()
	out := []Anomaly{}
	matchedDeposits := map[string]bool{}
	for _, payout := range payouts {
		found := false
		for _, deposit := range deposits {
			if deposit.Direction == "credit" && deposit.Currency == payout.Currency && abs(toMinor(deposit.Amount)-toMinor(payout.Amount)) <= 500 {
				found = true
				matchedDeposits[deposit.ID] = true
			}
		}
		if !found {
			out = append(out, Anomaly{ID: "missing_payout:" + payout.ID, Type: "missing_payout", Severity: "high", Title: "Expected payout has not reached the bank", Description: "A processor payout has no matching bank deposit.", ResourceType: "processor_payout", ResourceID: payout.ID, DetectedAt: now, RecommendedAction: "Check processor payout status and bank deposit timing."})
		}
		if payout.ExpectedArrivalAt.Before(now.AddDate(0, 0, -3)) {
			out = append(out, Anomaly{ID: "delayed_payout:" + payout.ID, Type: "delayed_payout", Severity: "medium", Title: "Payout appears delayed", Description: "A payout expected more than 3 days ago has not been reconciled.", ResourceType: "processor_payout", ResourceID: payout.ID, DetectedAt: now, RecommendedAction: "Verify payout status with the processor."})
		}
	}
	for _, deposit := range deposits {
		if deposit.Direction == "credit" && !matchedDeposits[deposit.ID] {
			out = append(out, Anomaly{ID: "unmatched_deposit:" + deposit.ID, Type: "unmatched_deposit", Severity: "medium", Title: "Unmatched bank deposit", Description: "A bank deposit is not tied to a known payout.", ResourceType: "bank_transaction", ResourceID: deposit.ID, DetectedAt: now, RecommendedAction: "Review bank deposit source and processor payout records."})
		}
	}
	gross, feeTotal, refundTotal := int64(0), int64(0), int64(0)
	for _, payout := range payouts {
		gross += toMinor(payout.Amount)
	}
	for _, fee := range fees {
		feeTotal += toMinor(fee.Amount)
	}
	for _, refund := range refunds {
		refundTotal += toMinor(refund.Amount)
	}
	if gross > 0 && float64(feeTotal)/float64(gross) > 0.05 {
		out = append(out, Anomaly{ID: "high_processor_fee", Type: "high_processor_fee", Severity: "medium", Title: "Processor fees look high", Description: "Processor fees exceed 5% of payout volume.", ResourceType: "fees", DetectedAt: now, RecommendedAction: "Review processor fee settings and pricing tier."})
	}
	if gross > 0 && float64(refundTotal)/float64(gross) > 0.10 {
		out = append(out, Anomaly{ID: "high_refund_rate", Type: "high_refund_rate", Severity: "medium", Title: "Refund rate looks high", Description: "Refunds exceed 10% of payout volume.", ResourceType: "refunds", DetectedAt: now, RecommendedAction: "Review recent refunds for operational issues."})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Severity > out[j].Severity })
	return out
}

func toMinor(v float64) int64 { return int64(math.Round(v * 100)) }
func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
