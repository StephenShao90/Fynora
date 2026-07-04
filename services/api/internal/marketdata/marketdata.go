package marketdata

import (
	"context"
	"strings"
	"time"
)

type Quote struct {
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	Price         float64   `json:"price"`
	Currency      string    `json:"currency"`
	PreviousClose float64   `json:"previous_close"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"change_percent"`
	AsOf          time.Time `json:"as_of"`
}

type Provider interface {
	GetQuote(ctx context.Context, symbol string) (*Quote, error)
	GetQuotes(ctx context.Context, symbols []string) ([]Quote, error)
}

type MockProvider struct{}

func (MockProvider) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	quotes, _ := (MockProvider{}).GetQuotes(ctx, []string{symbol})
	return &quotes[0], nil
}

func (MockProvider) GetQuotes(ctx context.Context, symbols []string) ([]Quote, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	fixtures := map[string]Quote{
		"VFV.TO": {Symbol: "VFV.TO", Name: "Vanguard S&P 500 Index ETF", Price: 148.35, Currency: "CAD", PreviousClose: 147.9},
		"XEQT.TO": {Symbol: "XEQT.TO", Name: "iShares Core Equity ETF Portfolio", Price: 36.42, Currency: "CAD", PreviousClose: 36.1},
		"VOO": {Symbol: "VOO", Name: "Vanguard S&P 500 ETF", Price: 545.11, Currency: "USD", PreviousClose: 542.7},
		"AAPL": {Symbol: "AAPL", Name: "Apple Inc.", Price: 214.29, Currency: "USD", PreviousClose: 211.3},
		"MSFT": {Symbol: "MSFT", Name: "Microsoft Corp.", Price: 498.84, Currency: "USD", PreviousClose: 496.2},
		"CASH": {Symbol: "CASH", Name: "Cash", Price: 1, Currency: "CAD", PreviousClose: 1},
	}
	now := time.Now().UTC()
	out := make([]Quote, 0, len(symbols))
	for _, s := range symbols {
		q, ok := fixtures[strings.ToUpper(strings.TrimSpace(s))]
		if !ok {
			q = Quote{Symbol: strings.ToUpper(s), Name: strings.ToUpper(s), Price: 100, Currency: "USD", PreviousClose: 99}
		}
		q.Change = q.Price - q.PreviousClose
		if q.PreviousClose > 0 {
			q.ChangePercent = q.Change / q.PreviousClose * 100
		}
		q.AsOf = now
		out = append(out, q)
	}
	return out, nil
}
