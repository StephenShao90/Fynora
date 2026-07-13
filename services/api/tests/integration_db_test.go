package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/db"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/repository"
)

func TestDBIntegrationReadinessSkipsWithoutExplicitDB(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB-backed integration tests")
	}
	conn, err := db.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer conn.Close()
	repo := repository.NewClearflow(conn)
	if err := repo.Ping(context.Background()); err != nil {
		t.Fatalf("ping test database: %v", err)
	}
}

func TestDBIntegrationPortfolioPersistenceSkipsWithoutExplicitDB(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB-backed integration tests")
	}
	ctx := context.Background()
	conn, err := db.Open(ctx, url)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer conn.Close()
	repo := repository.NewClearflow(conn)
	user := models.User{ID: auth.NewID(), Email: auth.NewID() + "@clearflow.test", PasswordHash: "test", CreatedAt: time.Now().UTC()}
	if err := repo.EnsureUser(ctx, user); err != nil {
		t.Fatalf("ensure user: %v", err)
	}
	accountID, err := repo.EnsureDefaultBrokerageAccount(ctx, user.ID)
	if err != nil {
		t.Fatalf("ensure default account: %v", err)
	}
	imp := models.RawImport{ID: auth.NewID(), UserID: user.ID, ImportType: "holdings", OriginalFilename: "integration_holdings.csv", RowCount: 1, ImportedCount: 1, CreatedAt: time.Now().UTC()}
	holding := models.Holding{ID: auth.NewID(), UserID: user.ID, BrokerageAccountID: accountID, Symbol: "AAPL", SecurityName: "Apple Inc.", SecurityType: "stock", Quantity: 2, AverageCost: 100, Currency: "USD", MarketValue: 420, LastPrice: 210, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if _, err := repo.SavePortfolioImport(ctx, imp, []models.Holding{holding}, nil); err != nil {
		t.Fatalf("save import: %v", err)
	}
	holdings, err := repo.ListHoldings(ctx, user.ID)
	if err != nil {
		t.Fatalf("list holdings: %v", err)
	}
	if len(holdings) != 1 || holdings[0].Symbol != "AAPL" {
		t.Fatalf("unexpected holdings: %#v", holdings)
	}
	imports, err := repo.ListPortfolioImports(ctx, user.ID)
	if err != nil {
		t.Fatalf("list imports: %v", err)
	}
	if len(imports) != 1 || imports[0].OriginalFilename != "integration_holdings.csv" {
		t.Fatalf("unexpected imports: %#v", imports)
	}
}
