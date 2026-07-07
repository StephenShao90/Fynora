package validation

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var currencyRE = regexp.MustCompile(`^[A-Z]{3}$`)
var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func RequiredString(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

func Currency(value string) error {
	if !currencyRE.MatchString(strings.TrimSpace(value)) {
		return fmt.Errorf("currency must be an ISO 4217 currency code")
	}
	return nil
}

func UUID(value, field string) error {
	if !uuidRE.MatchString(strings.TrimSpace(value)) {
		return fmt.Errorf("%s must be a valid UUID", field)
	}
	return nil
}

func ParseOptionalISODate(value, field string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("%s must be an ISO date or RFC3339 timestamp", field)
}
