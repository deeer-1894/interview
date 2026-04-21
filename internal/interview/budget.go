package interview

import (
	"strconv"
	"strings"
)

func DeriveInterviewTurnLimit(timeBudget string) int {
	minutes := ParseTimeBudgetMinutes(timeBudget)
	switch {
	case minutes <= 0:
		return 5
	case minutes <= 15:
		return 4
	case minutes <= 25:
		return 6
	case minutes <= 30:
		return 8
	case minutes <= 45:
		return 12
	default:
		return 14
	}
}

func ParseTimeBudgetMinutes(raw string) int {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return 0
	}
	digits := make([]rune, 0, len(raw))
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) == 0 {
		return 0
	}
	value, err := strconv.Atoi(string(digits))
	if err != nil {
		return 0
	}
	return value
}
