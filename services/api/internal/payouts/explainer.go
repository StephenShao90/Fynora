package payouts

import (
	"fmt"
	"math"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

type BankDeposit struct {
	ID          string    `json:"id"`
	AmountMinor int64     `json:"amountMinor"`
	PostedAt    time.Time `json:"postedAt"`
}

type Explanation struct {
	PayoutID         string        `json:"payoutId"`
	Processor        string        `json:"processor"`
	GrossAmountMinor int64         `json:"grossAmountMinor"`
	FeesMinor        int64         `json:"feesMinor"`
	RefundsMinor     int64         `json:"refundsMinor"`
	NetAmountMinor   int64         `json:"netAmountMinor"`
	Currency         string        `json:"currency"`
	BankDeposit      *BankDeposit  `json:"bankDeposit,omitempty"`
	Summary          string        `json:"summary"`
	LineItems        []interface{} `json:"lineItems"`
	Warnings         []string      `json:"warnings"`
}

func Explain(payout models.Payout, payments []models.Payment, fees []models.Fee, refunds []models.Refund, deposits []models.BankTransaction) Explanation {
	gross, feeTotal, refundTotal := int64(0), int64(0), int64(0)
	for _, p := range payments {
		if p.OrganizationID == payout.OrganizationID {
			gross += toMinor(p.Amount)
		}
	}
	for _, f := range fees {
		if f.OrganizationID == payout.OrganizationID {
			feeTotal += toMinor(f.Amount)
		}
	}
	for _, r := range refunds {
		if r.OrganizationID == payout.OrganizationID {
			refundTotal += toMinor(r.Amount)
		}
	}
	net := toMinor(payout.Amount)
	if gross == 0 {
		gross = net + feeTotal + refundTotal
	}
	var deposit *BankDeposit
	warnings := []string{}
	for _, d := range deposits {
		if d.Direction == "credit" && d.Currency == payout.Currency && abs(toMinor(d.Amount)-net) <= 500 {
			deposit = &BankDeposit{ID: d.ID, AmountMinor: toMinor(d.Amount), PostedAt: d.PostedAt}
			break
		}
	}
	if deposit == nil {
		warnings = append(warnings, "no matching bank deposit found")
	} else if deposit.AmountMinor != net {
		warnings = append(warnings, "payout amount does not equal deposit amount")
	}
	if gross > 0 && float64(feeTotal)/float64(gross) > 0.05 {
		warnings = append(warnings, "unusually high fees")
	}
	if gross > 0 && float64(refundTotal)/float64(gross) > 0.10 {
		warnings = append(warnings, "unusually high refunds")
	}
	if deposit != nil && deposit.PostedAt.Sub(payout.ExpectedArrivalAt) > 72*time.Hour {
		warnings = append(warnings, "payout delayed longer than expected")
	}
	return Explanation{PayoutID: payout.ID, Processor: payout.Processor, GrossAmountMinor: gross, FeesMinor: feeTotal, RefundsMinor: refundTotal, NetAmountMinor: net, Currency: payout.Currency, BankDeposit: deposit, Summary: fmt.Sprintf("This payout represents %s in gross payments minus %s in fees and %s in refunds, resulting in a %s bank deposit.", money(gross), money(feeTotal), money(refundTotal), money(net)), LineItems: []interface{}{}, Warnings: warnings}
}

func toMinor(v float64) int64 { return int64(math.Round(v * 100)) }
func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
func money(v int64) string { return fmt.Sprintf("$%.2f", float64(v)/100) }
