package insights

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

type CategorySpend struct {
	Category               string  `json:"category"`
	AmountMinor            int64   `json:"amountMinor"`
	Percentage             float64 `json:"percentage"`
	ChangeVsPreviousPeriod float64 `json:"changeVsPreviousPeriod"`
}

type MerchantSpend struct {
	Merchant    string `json:"merchant"`
	AmountMinor int64  `json:"amountMinor"`
}

type SpendingInsight struct {
	TotalSpendMinor int64           `json:"totalSpendMinor"`
	Currency        string          `json:"currency"`
	Categories      []CategorySpend `json:"categories"`
	TopMerchants    []MerchantSpend `json:"topMerchants"`
	Notes           []string        `json:"notes"`
}

func Spending(bank []models.BankTransaction, from, to time.Time, currency string) SpendingInsight {
	if from.IsZero() {
		from = time.Now().UTC().AddDate(0, -1, 0)
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	cat := map[string]int64{}
	merchants := map[string]int64{}
	total := int64(0)
	for _, row := range bank {
		if row.Direction != "debit" || row.PostedAt.Before(from) || row.PostedAt.After(to) {
			continue
		}
		amount := toMinor(row.Amount)
		total += amount
		category := categorize(row.Description)
		cat[category] += amount
		merchants[row.Description] += amount
	}
	categories := []CategorySpend{}
	for name, amount := range cat {
		pct := 0.0
		if total > 0 {
			pct = float64(amount) / float64(total) * 100
		}
		categories = append(categories, CategorySpend{Category: name, AmountMinor: amount, Percentage: math.Round(pct*10) / 10, ChangeVsPreviousPeriod: 0})
	}
	sort.Slice(categories, func(i, j int) bool { return categories[i].AmountMinor > categories[j].AmountMinor })
	top := []MerchantSpend{}
	for name, amount := range merchants {
		top = append(top, MerchantSpend{Merchant: name, AmountMinor: amount})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].AmountMinor > top[j].AmountMinor })
	if len(top) > 5 {
		top = top[:5]
	}
	notes := []string{}
	if len(categories) > 0 {
		notes = append(notes, categories[0].Category+" is the largest spending category in this period.")
	}
	return SpendingInsight{TotalSpendMinor: total, Currency: currency, Categories: categories, TopMerchants: top, Notes: notes}
}

func categorize(description string) string {
	v := strings.ToLower(description)
	switch {
	case strings.Contains(v, "software"), strings.Contains(v, "saas"):
		return "software"
	case strings.Contains(v, "venue"), strings.Contains(v, "rent"):
		return "facilities"
	case strings.Contains(v, "food"), strings.Contains(v, "restaurant"):
		return "food"
	default:
		return "other"
	}
}
