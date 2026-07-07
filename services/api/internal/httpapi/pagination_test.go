package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestParseListQueryDefaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/payments", nil)
	query, err := ParseListQuery(req)
	if err != nil {
		t.Fatal(err)
	}
	if query.Limit != 50 || query.Offset != 0 {
		t.Fatalf("unexpected defaults: %#v", query)
	}
}

func TestParseListQueryRejectsInvalidValues(t *testing.T) {
	cases := []string{
		"/api/v1/payments?limit=0",
		"/api/v1/payments?limit=999",
		"/api/v1/payments?offset=-1",
		"/api/v1/payments?from=bad-date",
		"/api/v1/payments?to=bad-date",
	}
	for _, target := range cases {
		req := httptest.NewRequest("GET", target, nil)
		if _, err := ParseListQuery(req); err == nil {
			t.Fatalf("expected validation error for %s", target)
		}
	}
}
