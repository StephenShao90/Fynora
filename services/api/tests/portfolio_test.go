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
