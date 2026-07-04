package validation

import (
	"errors"
	"regexp"
	"strings"
)

var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func Email(email string) error {
	if !emailRE.MatchString(strings.TrimSpace(email)) {
		return errors.New("email must be valid")
	}
	return nil
}

func Password(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func RiskTolerance(v string) string {
	switch strings.ToLower(v) {
	case "conservative", "moderate", "aggressive":
		return strings.ToLower(v)
	default:
		return "moderate"
	}
}
