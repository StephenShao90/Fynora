package recommendations

import (
	"math"

	"github.com/StephenShao90/Fynora/services/api/internal/insights"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

type CashRecommendation struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AmountMinor int64  `json:"amountMinor,omitempty"`
	Currency    string `json:"currency"`
}

func Cash(bank []models.BankTransaction, anomalies []insights.Anomaly, fees []models.Fee, payouts []models.Payout, currency string) []CashRecommendation {
	outflow := int64(0)
	for _, row := range bank {
		if row.Direction == "debit" {
			outflow += toMinor(row.Amount)
		}
	}
	reserve := outflow
	if reserve == 0 {
		reserve = 420000
	}
	out := []CashRecommendation{{Type: "reserve", Priority: "high", Title: "Keep at least 30 days of operating cash", Description: "Based on recent outflows, this organization should maintain a practical operating cash reserve.", AmountMinor: reserve, Currency: currency}}
	for _, anomaly := range anomalies {
		switch anomaly.Type {
		case "unmatched_deposit":
			out = append(out, CashRecommendation{Type: "follow_up_unmatched_deposit", Priority: "medium", Title: "Investigate unmatched deposits", Description: "Resolve unmatched bank deposits before relying on them for operating decisions.", Currency: currency})
		case "missing_payout":
			out = append(out, CashRecommendation{Type: "follow_up_unmatched_deposit", Priority: "high", Title: "Follow up on missing payouts", Description: "A processor payout has not reached the bank. Confirm status before planning spend.", Currency: currency})
		case "high_refund_rate":
			out = append(out, CashRecommendation{Type: "review_refunds", Priority: "medium", Title: "Review refund activity", Description: "Refunds are elevated relative to payout volume.", Currency: currency})
		case "high_processor_fee":
			out = append(out, CashRecommendation{Type: "reduce_fees", Priority: "medium", Title: "Review processor fees", Description: "Fees look elevated. Compare processor pricing and payment methods.", Currency: currency})
		}
	}
	return out
}

func toMinor(v float64) int64 { return int64(math.Round(v * 100)) }
