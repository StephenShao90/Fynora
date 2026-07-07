package tests

import (
	"context"
	"os"
	"testing"

	"github.com/StephenShao90/Fynora/services/api/internal/db"
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
