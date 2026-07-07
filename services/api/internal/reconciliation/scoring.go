package reconciliation

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

type MatchStatus string

const (
	StatusMatched           MatchStatus = "matched"
	StatusLikelyMatch       MatchStatus = "likely_match"
	StatusAmountMismatch    MatchStatus = "amount_mismatch"
	StatusDateMismatch      MatchStatus = "date_mismatch"
	StatusCurrencyMismatch  MatchStatus = "currency_mismatch"
	StatusMissingPayout     MatchStatus = "missing_payout"
	StatusUnmatchedDeposit  MatchStatus = "unmatched_deposit"
	StatusDuplicatePossible MatchStatus = "duplicate_possible"
	StatusUnresolved        MatchStatus = "unresolved"
)

type ScoredMatch struct {
	ID                    string      `json:"id"`
	ProcessorPayoutID     string      `json:"processorPayoutId,omitempty"`
	BankDepositID         string      `json:"bankDepositId,omitempty"`
	Status                MatchStatus `json:"status"`
	ConfidenceScore       float64     `json:"confidenceScore"`
	AmountDifferenceMinor int64       `json:"amountDifferenceMinor"`
	Currency              string      `json:"currency"`
	Reasons               []string    `json:"reasons"`
	Explanation           string      `json:"explanation"`
}

func Score(payout models.Payout, deposit models.BankTransaction, fees []models.Fee, refunds []models.Refund) ScoredMatch {
	diff := abs(toMinor(payout.Amount) - toMinor(deposit.Amount))
	days := math.Abs(deposit.PostedAt.Sub(payout.ExpectedArrivalAt).Hours() / 24)
	reasons := []string{}
	score := 0.0
	status := StatusUnresolved
	if strings.EqualFold(payout.Currency, deposit.Currency) {
		score += 0.25
		reasons = append(reasons, "same_currency")
	} else {
		return ScoredMatch{ID: payout.ID + ":" + deposit.ID, ProcessorPayoutID: payout.ID, BankDepositID: deposit.ID, Status: StatusCurrencyMismatch, ConfidenceScore: 0.2, AmountDifferenceMinor: diff, Currency: payout.Currency, Reasons: []string{"currency_mismatch"}, Explanation: "The payout and bank deposit use different currencies, so this match is unlikely."}
	}
	if diff == 0 {
		score += 0.45
		reasons = append(reasons, "exact_amount")
	} else if diff <= feeRefundTolerance(fees, refunds) {
		score += 0.30
		reasons = append(reasons, "amount_off_by_fee")
		status = StatusAmountMismatch
	}
	if days <= 2 {
		score += 0.25
		reasons = append(reasons, "date_within_window")
	} else if days <= 5 {
		score += 0.10
		reasons = append(reasons, "date_shifted")
		status = StatusDateMismatch
	}
	if strings.Contains(strings.ToLower(deposit.Description), strings.ToLower(payout.Processor)) {
		score += 0.05
		reasons = append(reasons, "payout_reference_match")
	}
	if score >= 0.9 && diff == 0 {
		status = StatusMatched
	} else if score >= 0.65 {
		if status == StatusUnresolved {
			status = StatusLikelyMatch
		}
	} else if status == StatusUnresolved && diff > 0 {
		status = StatusAmountMismatch
	}
	return ScoredMatch{ID: payout.ID + ":" + deposit.ID, ProcessorPayoutID: payout.ID, BankDepositID: deposit.ID, Status: status, ConfidenceScore: round(score), AmountDifferenceMinor: diff, Currency: payout.Currency, Reasons: reasons, Explanation: explain(status, payout, deposit, diff, days)}
}

func ScoreAll(payouts []models.Payout, deposits []models.BankTransaction, fees []models.Fee, refunds []models.Refund) []ScoredMatch {
	out := []ScoredMatch{}
	usedPayouts := map[string]bool{}
	usedDeposits := map[string]bool{}
	for _, payout := range payouts {
		var best ScoredMatch
		for _, deposit := range deposits {
			if deposit.Direction != "credit" {
				continue
			}
			score := Score(payout, deposit, fees, refunds)
			if score.ConfidenceScore > best.ConfidenceScore {
				best = score
			}
		}
		if best.ConfidenceScore >= 0.5 {
			out = append(out, best)
			usedPayouts[payout.ID] = true
			usedDeposits[best.BankDepositID] = true
		}
	}
	for _, payout := range payouts {
		if !usedPayouts[payout.ID] {
			out = append(out, ScoredMatch{ID: payout.ID + ":missing", ProcessorPayoutID: payout.ID, Status: StatusMissingPayout, ConfidenceScore: 0, AmountDifferenceMinor: toMinor(payout.Amount), Currency: payout.Currency, Reasons: []string{"missing_payout"}, Explanation: "This processor payout has no likely matching bank deposit."})
		}
	}
	for _, deposit := range deposits {
		if deposit.Direction == "credit" && !usedDeposits[deposit.ID] {
			out = append(out, ScoredMatch{ID: "unmatched:" + deposit.ID, BankDepositID: deposit.ID, Status: StatusUnmatchedDeposit, ConfidenceScore: 0, AmountDifferenceMinor: toMinor(deposit.Amount), Currency: deposit.Currency, Reasons: []string{"unmatched_deposit"}, Explanation: "This bank deposit is not tied to a known processor payout."})
		}
	}
	return out
}

func feeRefundTolerance(fees []models.Fee, refunds []models.Refund) int64 {
	total := int64(250)
	for _, fee := range fees {
		total += toMinor(fee.Amount)
	}
	for _, refund := range refunds {
		total += toMinor(refund.Amount)
	}
	if total < 500 {
		return 500
	}
	return total
}

func explain(status MatchStatus, payout models.Payout, deposit models.BankTransaction, diff int64, days float64) string {
	switch status {
	case StatusMatched:
		return "This deposit matches the processor payout because the amount, currency, and deposit timing align."
	case StatusLikelyMatch:
		return "This deposit is likely the processor payout because the timing and currency align closely."
	case StatusAmountMismatch:
		return fmt.Sprintf("This deposit is close to the payout, but the amount differs by %d minor units.", diff)
	case StatusDateMismatch:
		return fmt.Sprintf("This payout and deposit have similar financial details, but the dates differ by %.0f days.", days)
	default:
		return "This payout and deposit need review."
	}
}

func toMinor(v float64) int64 {
	return int64(math.Round(v * 100))
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func round(v float64) float64 {
	return math.Round(v*100) / 100
}

func BusinessDaysBetween(a, b time.Time) int {
	if b.Before(a) {
		a, b = b, a
	}
	days := 0
	for d := a; d.Before(b); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			days++
		}
	}
	return days
}
