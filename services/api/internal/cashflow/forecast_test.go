package cashflow

import (
	"testing"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

func TestThirtyDayForecastReturnsDailySeries(t *testing.T) {
	now := time.Now().UTC()
	f := Build("org", 30, "CAD", []models.BankTransaction{
		{Amount: 300, Direction: "credit", PostedAt: now.AddDate(0, 0, -2)},
		{Amount: 90, Direction: "debit", PostedAt: now.AddDate(0, 0, -1)},
		{Amount: 100, Direction: "credit", PostedAt: now.AddDate(0, 0, -3)},
		{Amount: 40, Direction: "debit", PostedAt: now.AddDate(0, 0, -4)},
		{Amount: 80, Direction: "credit", PostedAt: now.AddDate(0, 0, -5)},
	}, nil)
	if len(f.Series) != 30 || f.HorizonDays != 30 {
		t.Fatalf("unexpected forecast series: %#v", f)
	}
	if f.Confidence != "medium" {
		t.Fatalf("expected medium confidence, got %s", f.Confidence)
	}
}

func TestSparseForecastIsLowConfidence(t *testing.T) {
	f := Build("org", 7, "CAD", nil, nil)
	if len(f.Series) != 7 || f.Confidence != "low" {
		t.Fatalf("expected sparse low confidence 7-day forecast, got %#v", f)
	}
}
