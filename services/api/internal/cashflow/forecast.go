package cashflow

import (
	"math"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

type Point struct {
	Date                  string `json:"date"`
	ProjectedBalanceMinor int64  `json:"projectedBalanceMinor"`
	ExpectedInflowsMinor  int64  `json:"expectedInflowsMinor"`
	ExpectedOutflowsMinor int64  `json:"expectedOutflowsMinor"`
}

type Forecast struct {
	OrganizationID              string   `json:"organizationId"`
	HorizonDays                 int      `json:"horizonDays"`
	StartingBalanceMinor        int64    `json:"startingBalanceMinor"`
	ProjectedEndingBalanceMinor int64    `json:"projectedEndingBalanceMinor"`
	Currency                    string   `json:"currency"`
	Series                      []Point  `json:"series"`
	Assumptions                 []string `json:"assumptions"`
	Confidence                  string   `json:"confidence"`
}

func Build(orgID string, horizon int, currency string, bank []models.BankTransaction, payouts []models.Payout) Forecast {
	if horizon != 7 && horizon != 30 && horizon != 60 && horizon != 90 {
		horizon = 30
	}
	starting := int64(0)
	inflow, outflow, observed := int64(0), int64(0), 0
	cutoff := time.Now().UTC().AddDate(0, 0, -60)
	for _, row := range bank {
		amount := toMinor(row.Amount)
		if row.Direction == "credit" {
			starting += amount
			if row.PostedAt.After(cutoff) {
				inflow += amount
				observed++
			}
		} else {
			starting -= amount
			if row.PostedAt.After(cutoff) {
				outflow += amount
				observed++
			}
		}
	}
	dailyIn, dailyOut := inflow/60, outflow/60
	confidence := "medium"
	if observed < 5 {
		confidence = "low"
	}
	series := []Point{}
	balance := starting
	for i := 1; i <= horizon; i++ {
		expectedIn, expectedOut := dailyIn, dailyOut
		for _, payout := range payouts {
			if payout.Status != "paid" && daysFromNow(payout.ExpectedArrivalAt) == i {
				expectedIn += toMinor(payout.Amount)
			}
		}
		balance += expectedIn - expectedOut
		series = append(series, Point{Date: time.Now().UTC().AddDate(0, 0, i).Format("2006-01-02"), ProjectedBalanceMinor: balance, ExpectedInflowsMinor: expectedIn, ExpectedOutflowsMinor: expectedOut})
	}
	return Forecast{OrganizationID: orgID, HorizonDays: horizon, StartingBalanceMinor: starting, ProjectedEndingBalanceMinor: balance, Currency: currency, Series: series, Assumptions: []string{"Used average daily inflow from the last 60 days.", "Excluded unresolved unmatched deposits from special treatment.", "Included known pending payouts when expected arrival dates were present."}, Confidence: confidence}
}

func daysFromNow(t time.Time) int { return int(math.Round(t.Sub(time.Now().UTC()).Hours() / 24)) }
func toMinor(v float64) int64     { return int64(math.Round(v * 100)) }
