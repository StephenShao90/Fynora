package portfolio

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/marketdata"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

type BrokerageConnector interface {
	Name() string
	Connect(ctx context.Context, userID string) (*ConnectionResult, error)
	SyncHoldings(ctx context.Context, userID string) ([]HoldingInput, error)
	SyncTransactions(ctx context.Context, userID string, from, to time.Time) ([]PortfolioTransactionInput, error)
}

type ConnectionResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
type HoldingInput = models.Holding
type PortfolioTransactionInput = models.PortfolioTransaction

type ManualConnector struct{}

func (ManualConnector) Name() string { return "manual" }
func (ManualConnector) Connect(context.Context, string) (*ConnectionResult, error) {
	return &ConnectionResult{Status: "manual", Message: "Manual holdings enabled."}, nil
}
func (ManualConnector) SyncHoldings(context.Context, string) ([]HoldingInput, error) { return nil, nil }
func (ManualConnector) SyncTransactions(context.Context, string, time.Time, time.Time) ([]PortfolioTransactionInput, error) {
	return nil, nil
}

type CSVConnector struct{ ManualConnector }

func (CSVConnector) Name() string { return "csv" }

type MockPlaidInvestmentsConnector struct{ ManualConnector }

func (MockPlaidInvestmentsConnector) Name() string { return "mock_plaid_investments" }

type MockWealthsimpleCSVConnector struct{ ManualConnector }

func (MockWealthsimpleCSVConnector) Name() string { return "mock_wealthsimple_csv" }

type Summary struct {
	TotalMarketValue      float64            `json:"total_market_value"`
	TotalCostBasis        float64            `json:"total_cost_basis"`
	UnrealizedGainLoss    float64            `json:"unrealized_gain_loss"`
	UnrealizedGainLossPct float64            `json:"unrealized_gain_loss_pct"`
	CashValue             float64            `json:"cash_value"`
	InvestedValue         float64            `json:"invested_value"`
	AccountBreakdown      map[string]float64 `json:"account_breakdown"`
	TopHoldings           []AllocationItem   `json:"top_holdings"`
}

type AllocationItem struct {
	Name    string  `json:"name"`
	Value   float64 `json:"value"`
	Percent float64 `json:"percent"`
}
type Allocation struct {
	ByAccountType  []AllocationItem `json:"by_account_type"`
	BySecurityType []AllocationItem `json:"by_security_type"`
	ByCurrency     []AllocationItem `json:"by_currency"`
	BySymbol       []AllocationItem `json:"by_symbol"`
}
type RiskFinding struct {
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
}

func PriceHoldings(ctx context.Context, provider marketdata.Provider, holdings []models.Holding) []models.Holding {
	symbols := make([]string, 0, len(holdings))
	for _, h := range holdings {
		symbols = append(symbols, h.Symbol)
	}
	quotes, _ := provider.GetQuotes(ctx, symbols)
	bySymbol := map[string]marketdata.Quote{}
	for _, q := range quotes {
		bySymbol[strings.ToUpper(q.Symbol)] = q
	}
	for i := range holdings {
		if q, ok := bySymbol[strings.ToUpper(holdings[i].Symbol)]; ok {
			holdings[i].LastPrice = q.Price
			holdings[i].PriceAsOf = q.AsOf
			if holdings[i].MarketValue == 0 {
				holdings[i].MarketValue = holdings[i].Quantity * q.Price
			}
		}
	}
	return holdings
}

func BuildSummary(holdings []models.Holding, accounts []models.BrokerageAccount) Summary {
	accountTypes := map[string]string{}
	for _, a := range accounts {
		accountTypes[a.ID] = a.AccountType
	}
	s := Summary{AccountBreakdown: map[string]float64{}}
	for _, h := range holdings {
		value := h.MarketValue
		cost := h.AverageCost * h.Quantity
		s.TotalMarketValue += value
		s.TotalCostBasis += cost
		if strings.EqualFold(h.SecurityType, "cash") || strings.EqualFold(h.Symbol, "CASH") {
			s.CashValue += value
		} else {
			s.InvestedValue += value
		}
		kind := accountTypes[h.BrokerageAccountID]
		if kind == "" {
			kind = "other"
		}
		s.AccountBreakdown[kind] += value
	}
	s.UnrealizedGainLoss = s.TotalMarketValue - s.TotalCostBasis
	if s.TotalCostBasis > 0 {
		s.UnrealizedGainLossPct = s.UnrealizedGainLoss / s.TotalCostBasis * 100
	}
	s.TopHoldings = allocationBy(holdings, func(h models.Holding) string { return h.Symbol }, s.TotalMarketValue)
	if len(s.TopHoldings) > 5 {
		s.TopHoldings = s.TopHoldings[:5]
	}
	return s
}

func BuildAllocation(holdings []models.Holding, accounts []models.BrokerageAccount) Allocation {
	total := 0.0
	for _, h := range holdings {
		total += h.MarketValue
	}
	accountTypes := map[string]string{}
	for _, a := range accounts {
		accountTypes[a.ID] = a.AccountType
	}
	return Allocation{
		ByAccountType: allocationBy(holdings, func(h models.Holding) string {
			if accountTypes[h.BrokerageAccountID] == "" {
				return "other"
			}
			return accountTypes[h.BrokerageAccountID]
		}, total),
		BySecurityType: allocationBy(holdings, func(h models.Holding) string { return h.SecurityType }, total),
		ByCurrency:     allocationBy(holdings, func(h models.Holding) string { return h.Currency }, total),
		BySymbol:       allocationBy(holdings, func(h models.Holding) string { return h.Symbol }, total),
	}
}

func ConcentrationRisk(holdings []models.Holding, profile models.AdvisorProfile) []RiskFinding {
	total := 0.0
	for _, h := range holdings {
		total += h.MarketValue
	}
	if total == 0 {
		return nil
	}
	bySymbol := allocationBy(holdings, func(h models.Holding) string { return h.Symbol }, total)
	var findings []RiskFinding
	if len(bySymbol) > 0 && bySymbol[0].Percent > 25 {
		findings = append(findings, RiskFinding{Severity: "medium", Title: "Single-holding concentration", Explanation: bySymbol[0].Name + " is above 25% of tracked holdings. Consider reviewing diversification across broad-market funds, cash, fixed income, or other assets based on your goals."})
	}
	top5 := 0.0
	for i, item := range bySymbol {
		if i < 5 {
			top5 += item.Percent
		}
	}
	if top5 > 60 {
		findings = append(findings, RiskFinding{Severity: "medium", Title: "Top-five concentration", Explanation: "Your top five holdings exceed 60% of tracked value. This can increase portfolio-specific risk."})
	}
	stockValue := 0.0
	for _, h := range holdings {
		if h.SecurityType == "stock" {
			stockValue += h.MarketValue
		}
	}
	if strings.EqualFold(profile.RiskTolerance, "conservative") && stockValue/total > 0.75 {
		findings = append(findings, RiskFinding{Severity: "high", Title: "Risk tolerance mismatch", Explanation: "Your risk tolerance is conservative, but most tracked holdings are individual equities. Review whether the allocation fits your time horizon."})
	}
	return findings
}

func RebalanceSuggestions(findings []RiskFinding) []RiskFinding {
	if len(findings) == 0 {
		return []RiskFinding{{Severity: "info", Title: "Diversification check", Explanation: "No major concentration warnings detected. Keep reviewing allocation, cash needs, fees, and long-term goals."}}
	}
	return findings
}

func allocationBy(holdings []models.Holding, key func(models.Holding) string, total float64) []AllocationItem {
	m := map[string]float64{}
	for _, h := range holdings {
		name := key(h)
		if name == "" {
			name = "other"
		}
		m[name] += h.MarketValue
	}
	out := make([]AllocationItem, 0, len(m))
	for k, v := range m {
		p := 0.0
		if total > 0 {
			p = math.Round(v/total*1000) / 10
		}
		out = append(out, AllocationItem{Name: k, Value: v, Percent: p})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	return out
}
