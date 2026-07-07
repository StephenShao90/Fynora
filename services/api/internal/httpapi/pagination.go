package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/validation"
)

const (
	DefaultLimit = 50
	MaxLimit     = 100
)

type ListQuery struct {
	Limit  int
	Offset int
	From   *time.Time
	To     *time.Time
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

func ParseListQuery(r *http.Request) (ListQuery, error) {
	values := r.URL.Query()
	query := ListQuery{Limit: DefaultLimit}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > MaxLimit {
			return query, fmt.Errorf("limit must be between 1 and %d", MaxLimit)
		}
		query.Limit = limit
	}
	if raw := values.Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return query, fmt.Errorf("offset must be greater than or equal to 0")
		}
		query.Offset = offset
	}
	from, err := validation.ParseOptionalISODate(values.Get("from"), "from")
	if err != nil {
		return query, err
	}
	to, err := validation.ParseOptionalISODate(values.Get("to"), "to")
	if err != nil {
		return query, err
	}
	query.From = from
	query.To = to
	return query, nil
}

func Page[T any](rows []T, query ListQuery) []T {
	if query.Offset >= len(rows) {
		return []T{}
	}
	end := query.Offset + query.Limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[query.Offset:end]
}

func Paginated(data interface{}, query ListQuery) PaginatedResponse {
	return PaginatedResponse{
		Data: data,
		Pagination: Pagination{
			Limit:  query.Limit,
			Offset: query.Offset,
		},
	}
}
