package advisor

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

type CategoryTotal struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type CashFlowSummary struct {
	AverageMonthlyIncome      float64 `json:"average_monthly_income"`
	AverageFixedExpenses      float64 `json:"average_fixed_expenses"`
	AverageVariableExpenses   float64 `json:"average_variable_expenses"`
	AverageNetCashFlow        float64 `json:"average_net_cash_flow"`
	SavingsCapacity           float64 `json:"savings_capacity"`
	SafeSavingsRecommendation float64 `json:"safe_savings_recommendation"`
	BufferRecommendation      float64 `json:"buffer_recommendation"`
}

type Subscription struct {
	Merchant      string    `json:"merchant"`
	Amount        float64   `json:"amount_estimate"`
	Frequency     string    `json:"frequency"`
	LastChargedAt time.Time `json:"last_charged_at"`
	Confidence    float64   `json:"confidence"`
}

type Anomaly struct {
	Reason      string             `json:"reason"`
	Transaction models.Transaction `json:"transaction"`
	Severity    string             `json:"severity"`
	Explanation string             `json:"explanation"`
}

type EmergencyFundPlan struct {
	Target                  float64 `json:"target"`
	Current                 float64 `json:"current"`
	Gap                     float64 `json:"gap"`
	RecommendedContribution float64 `json:"recommended_monthly_contribution"`
	MonthsToTarget          int     `json:"months_to_target"`
	Explanation             string  `json:"explanation"`
}

type ProjectionPoint struct {
	Year          int     `json:"year"`
	Lower         float64 `json:"lower"`
	Expected      float64 `json:"expected"`
	Upper         float64 `json:"upper"`
	Contributions float64 `json:"contributions"`
}

type Projection struct {
	Points             []ProjectionPoint `json:"points"`
	TotalContributions float64           `json:"total_contributions"`
	EstimatedGrowth    float64           `json:"estimated_growth"`
	Disclaimer         string            `json:"disclaimer"`
}

func NormalizeMerchant(description, merchant string) string {
	source := strings.ToUpper(strings.TrimSpace(merchant))
	if source == "" {
		source = strings.ToUpper(description)
	}
	rules := map[string]string{
		"UBER EATS": "Uber Eats", "UBER": "Uber", "LYFT": "Lyft",
		"AMZN": "Amazon", "AMAZON": "Amazon", "STARBUCKS": "Starbucks",
		"NETFLIX": "Netflix", "SPOTIFY": "Spotify", "APPLE.COM/BILL": "Apple",
		"APPLE": "Apple", "MCDONALD": "McDonald's", "WALMART": "Walmart",
	}
	for needle, value := range rules {
		if strings.Contains(source, needle) {
			return value
		}
	}
	clean := strings.NewReplacer("_", " ", "-", " ", "*", " ").Replace(strings.ToLower(source))
	parts := strings.Fields(clean)
	for i := range parts {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	if len(parts) == 0 {
		return "Unknown"
	}
	return strings.Join(parts, " ")
}

func Categorize(t models.Transaction) string {
	source := strings.ToLower(t.NormalizedMerchant + " " + t.Description + " " + t.Category)
	switch {
	case strings.Contains(source, "payroll") || strings.Contains(source, "salary") || strings.Contains(source, "deposit"):
		return "Income"
	case strings.Contains(source, "rent") || strings.Contains(source, "landlord"):
		return "Housing"
	case strings.Contains(source, "grocery") || strings.Contains(source, "supermarket"):
		return "Groceries"
	case strings.Contains(source, "uber eats") || strings.Contains(source, "doordash") || strings.Contains(source, "mcdonald") || strings.Contains(source, "starbucks") || strings.Contains(source, "restaurant"):
		return "Food"
	case strings.Contains(source, "uber") || strings.Contains(source, "lyft") || strings.Contains(source, "transit"):
		return "Transportation"
	case strings.Contains(source, "netflix") || strings.Contains(source, "spotify") || strings.Contains(source, "apple") || strings.Contains(source, "youtube"):
		return "Subscriptions"
	case strings.Contains(source, "amazon") || strings.Contains(source, "walmart"):
		return "Shopping"
	default:
		if t.Direction == "income" {
			return "Income"
		}
		return "Other"
	}
}

func CashFlow(transactions []models.Transaction) CashFlowSummary {
	months := monthCount(transactions)
	if months == 0 {
		months = 1
	}
	var income, fixed, variable float64
	for _, t := range transactions {
		if t.Direction == "income" {
			income += t.Amount
			continue
		}
		if isFixed(t.Category) {
			fixed += t.Amount
		} else {
			variable += t.Amount
		}
	}
	avgIncome := income / float64(months)
	avgFixed := fixed / float64(months)
	avgVariable := variable / float64(months)
	net := avgIncome - avgFixed - avgVariable
	safe := math.Max(0, math.Round(net*0.7))
	return CashFlowSummary{
		AverageMonthlyIncome: avgIncome, AverageFixedExpenses: avgFixed,
		AverageVariableExpenses: avgVariable, AverageNetCashFlow: net,
		SavingsCapacity: math.Max(0, net), SafeSavingsRecommendation: safe,
		BufferRecommendation: math.Max(0, math.Round(net-safe)),
	}
}

func CategoryBreakdown(transactions []models.Transaction) []CategoryTotal {
	m := map[string]float64{}
	for _, t := range transactions {
		if t.Direction == "expense" {
			m[t.Category] += t.Amount
		}
	}
	return totals(m)
}

func MerchantBreakdown(transactions []models.Transaction) []CategoryTotal {
	m := map[string]float64{}
	for _, t := range transactions {
		if t.Direction == "expense" {
			m[t.NormalizedMerchant] += t.Amount
		}
	}
	return totals(m)
}

func DetectSubscriptions(transactions []models.Transaction) []Subscription {
	groups := map[string][]models.Transaction{}
	for _, t := range transactions {
		if t.Direction == "expense" {
			groups[t.NormalizedMerchant] = append(groups[t.NormalizedMerchant], t)
		}
	}
	var out []Subscription
	for merchant, rows := range groups {
		if len(rows) < 2 {
			continue
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].OccurredAt.Before(rows[j].OccurredAt) })
		var similar int
		for i := 1; i < len(rows); i++ {
			diffDays := rows[i].OccurredAt.Sub(rows[i-1].OccurredAt).Hours() / 24
			base := math.Max(rows[i-1].Amount, 1)
			if diffDays >= 25 && diffDays <= 35 && math.Abs(rows[i].Amount-rows[i-1].Amount)/base <= 0.15 {
				similar++
			}
		}
		if similar > 0 {
			out = append(out, Subscription{Merchant: merchant, Amount: rows[len(rows)-1].Amount, Frequency: "monthly", LastChargedAt: rows[len(rows)-1].OccurredAt, Confidence: math.Min(0.95, 0.65+float64(similar)*0.15)})
		}
	}
	return out
}

func DetectAnomalies(transactions []models.Transaction) []Anomaly {
	byCategory := map[string][]float64{}
	for _, t := range transactions {
		if t.Direction == "expense" {
			byCategory[t.Category] = append(byCategory[t.Category], t.Amount)
		}
	}
	var out []Anomaly
	for _, t := range transactions {
		vals := byCategory[t.Category]
		if t.Direction != "expense" || len(vals) < 2 {
			continue
		}
		avg, sd := meanSD(vals)
		if sd > 0 && t.Amount > avg+2*sd {
			out = append(out, Anomaly{Reason: "category_spike", Transaction: t, Severity: "medium", Explanation: fmt.Sprintf("%s is more than two standard deviations above your usual %s spending.", money(t.Amount), t.Category)})
		}
	}
	return append(out, DetectDuplicateCharges(transactions)...)
}

func DetectDuplicateCharges(transactions []models.Transaction) []Anomaly {
	var out []Anomaly
	for i := 0; i < len(transactions); i++ {
		for j := i + 1; j < len(transactions); j++ {
			a, b := transactions[i], transactions[j]
			if a.Direction == "expense" && b.Direction == "expense" && a.NormalizedMerchant == b.NormalizedMerchant && math.Abs(a.Amount-b.Amount) < 0.01 && math.Abs(a.OccurredAt.Sub(b.OccurredAt).Hours()) <= 72 {
				out = append(out, Anomaly{Reason: "possible_duplicate_charge", Transaction: b, Severity: "high", Explanation: "Same merchant, amount, and close date. Review whether this was intentional."})
			}
		}
	}
	return out
}

func EmergencyFund(profile models.AdvisorProfile, transactions []models.Transaction) EmergencyFundPlan {
	cf := CashFlow(transactions)
	var essential float64
	for _, t := range transactions {
		if t.Direction == "expense" && isEssential(t.Category) {
			essential += t.Amount
		}
	}
	months := monthCount(transactions)
	if months == 0 {
		months = 1
	}
	targetMonths := profile.EmergencyFundMonthsTarget
	if targetMonths == 0 {
		targetMonths = 6
	}
	target := essential / float64(months) * float64(targetMonths)
	gap := math.Max(0, target-profile.CurrentEmergencyFund)
	contribution := math.Min(gap, math.Max(0, cf.AverageNetCashFlow*0.35))
	monthsToTarget := 0
	if contribution > 0 {
		monthsToTarget = int(math.Ceil(gap / contribution))
	}
	return EmergencyFundPlan{Target: target, Current: profile.CurrentEmergencyFund, Gap: gap, RecommendedContribution: contribution, MonthsToTarget: monthsToTarget, Explanation: fmt.Sprintf("Target uses %.0f months of essential expenses. Projections are educational estimates, not financial advice.", float64(targetMonths))}
}

func AccountPriority(profile models.AdvisorProfile) []string {
	if strings.EqualFold(profile.Country, "CA") {
		return []string{"Build a starter emergency fund", "Pay high-interest debt", "Build a 3-6 month emergency fund", "Use TFSA for flexible tax-free investing", "Use FHSA if saving for a first home", "Use RRSP depending on income and tax bracket", "Use non-registered investing for additional long-term goals"}
	}
	steps := []string{"Build a starter emergency fund"}
	if profile.HasEmployerMatch {
		steps = append(steps, "Contribute enough to capture employer match")
	}
	return append(steps, "Pay high-interest debt", "Build a 3-6 month emergency fund", "Consider IRA or Roth IRA depending on eligibility", "Increase workplace retirement contributions", "Use taxable brokerage for additional long-term investing")
}

func MonthlyAllocation(profile models.AdvisorProfile, transactions []models.Transaction) map[string]float64 {
	cf := CashFlow(transactions)
	net := math.Max(0, cf.AverageNetCashFlow)
	emergency := EmergencyFund(profile, transactions)
	emergencyPart := math.Min(net*0.35, emergency.Gap)
	debtPart := 0.0
	if profile.HasHighInterestDebt {
		debtPart = net * 0.25
	}
	investing := math.Max(0, net*0.35-debtPart*0.2)
	shortTerm := net * 0.18
	buffer := math.Max(0, net-emergencyPart-debtPart-investing-shortTerm)
	return map[string]float64{"emergency_fund": math.Round(emergencyPart), "debt_priority": math.Round(debtPart), "long_term_investing": math.Round(investing), "short_term_savings": math.Round(shortTerm), "buffer": math.Round(buffer)}
}

func InvestmentProjection(monthly, initial float64, years int, risk string) Projection {
	if years <= 0 {
		years = 30
	}
	expected := map[string]float64{"conservative": 0.04, "moderate": 0.06, "aggressive": 0.08}[strings.ToLower(risk)]
	if expected == 0 {
		expected = 0.06
	}
	var points []ProjectionPoint
	lower, mid, upper := initial, initial, initial
	for y := 1; y <= years; y++ {
		for m := 0; m < 12; m++ {
			lower = (lower + monthly) * (1 + math.Max(0, expected-0.02)/12)
			mid = (mid + monthly) * (1 + expected/12)
			upper = (upper + monthly) * (1 + (expected+0.02)/12)
		}
		points = append(points, ProjectionPoint{Year: y, Lower: lower, Expected: mid, Upper: upper, Contributions: initial + monthly*12*float64(y)})
	}
	contrib := initial + monthly*12*float64(years)
	return Projection{Points: points, TotalContributions: contrib, EstimatedGrowth: mid - contrib, Disclaimer: "Hypothetical educational projection only. Returns are not guaranteed and this is not financial advice."}
}

func RuleBasedChat(message string, profile models.AdvisorProfile, transactions []models.Transaction, portfolioValue float64, concentration []string) string {
	cf := CashFlow(transactions)
	ef := EmergencyFund(profile, transactions)
	alloc := MonthlyAllocation(profile, transactions)
	risk := ""
	if len(concentration) > 0 {
		risk = " " + concentration[0]
	}
	return fmt.Sprintf("Based on your average net cash flow of %s/month and emergency fund gap of %s, a balanced educational plan is %s toward emergency fund, %s toward long-term investing, %s toward short-term savings, and %s as a buffer. Your tracked portfolio value is %s.%s This is educational guidance, not financial advice, and it avoids individual stock buy/sell recommendations.", money(cf.AverageNetCashFlow), money(ef.Gap), money(alloc["emergency_fund"]), money(alloc["long_term_investing"]), money(alloc["short_term_savings"]), money(alloc["buffer"]), money(portfolioValue), risk)
}

func monthCount(rows []models.Transaction) int {
	seen := map[string]bool{}
	for _, t := range rows {
		seen[t.OccurredAt.Format("2006-01")] = true
	}
	return len(seen)
}

func isFixed(cat string) bool {
	return cat == "Housing" || cat == "Subscriptions" || cat == "Insurance" || cat == "Utilities" || cat == "Debt"
}

func isEssential(cat string) bool {
	return cat == "Housing" || cat == "Groceries" || cat == "Transportation" || cat == "Insurance" || cat == "Utilities" || cat == "Debt"
}

func totals(m map[string]float64) []CategoryTotal {
	out := make([]CategoryTotal, 0, len(m))
	for k, v := range m {
		out = append(out, CategoryTotal{Name: k, Amount: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Amount > out[j].Amount })
	return out
}

func meanSD(vals []float64) (float64, float64) {
	var sum float64
	for _, v := range vals {
		sum += v
	}
	mean := sum / float64(len(vals))
	variance := 0.0
	for _, v := range vals {
		variance += math.Pow(v-mean, 2)
	}
	return mean, math.Sqrt(variance / float64(len(vals)))
}

func money(v float64) string { return fmt.Sprintf("$%.0f", v) }
