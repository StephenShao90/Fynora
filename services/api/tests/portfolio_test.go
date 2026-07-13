package tests

import (
	"testing"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/portfolio"
)

func TestPortfolioSummaryAndConcentration(t *testing.T) {
	holdings := []models.Holding{
		{Symbol: "AAPL", SecurityType: "stock", Quantity: 10, AverageCost: 100, MarketValue: 4000},
		{Symbol: "VFV.TO", SecurityType: "etf", Quantity: 10, AverageCost: 100, MarketValue: 1000},
	}
	s := portfolio.BuildSummary(holdings, nil)
	if s.TotalMarketValue != 5000 || s.UnrealizedGainLoss != 3000 {
		t.Fatalf("bad summary: %#v", s)
	}
	risk := portfolio.ConcentrationRisk(holdings, models.AdvisorProfile{RiskTolerance: "conservative"})
	if len(risk) == 0 {
		t.Fatalf("expected concentration findings")
	}
}

func TestPortfolioPerformanceSeparatesContributionsAndReturns(t *testing.T) {
	holdings := []models.Holding{
		{Symbol: "AAPL", SecurityType: "stock", Quantity: 10, AverageCost: 100, MarketValue: 1600},
		{Symbol: "CASH", SecurityType: "cash", Quantity: 200, AverageCost: 1, MarketValue: 200},
	}
	txs := []models.PortfolioTransaction{
		{TransactionType: "deposit", Amount: 1000},
		{TransactionType: "dividend", Amount: 25},
		{TransactionType: "fee", Amount: 5},
	}
	perf := portfolio.BuildPerformance(holdings, txs, nil)
	if perf.NetContributions != 1000 {
		t.Fatalf("expected net contributions 1000, got %#v", perf)
	}
	if perf.InvestmentReturnEstimate != 800 {
		t.Fatalf("expected investment return estimate 800, got %#v", perf)
	}
	if perf.Dividends != 25 || perf.Fees != 5 {
		t.Fatalf("expected dividend/fee rollups, got %#v", perf)
	}
}
