package retention

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	day  = 24 * time.Hour
	week = 7 * day
)

var (
	ErrEmptyPolicy     = errors.New("the retention policy must keep at least one snapshot: set at least one rule")
	ErrNegativeRule    = errors.New("retention rules must not be negative")
	ErrInvalidDuration = errors.New("invalid keep within duration (use for example 30d, 2w or 36h)")
)

type Rules struct {
	Last    int
	Daily   int
	Weekly  int
	Monthly int
	Yearly  int
	Within  time.Duration
}

func (r Rules) counts() []int {
	return []int{r.Last, r.Daily, r.Weekly, r.Monthly, r.Yearly}
}

func (r Rules) Validate() error {
	if slices.ContainsFunc(r.counts(), func(count int) bool { return count < 0 }) || r.Within < 0 {
		return ErrNegativeRule
	}

	if !slices.ContainsFunc(r.counts(), func(count int) bool { return count > 0 }) && r.Within == 0 {
		return ErrEmptyPolicy
	}

	return nil
}

func (r Rules) String() string {
	var parts []string
	for _, rule := range []struct {
		name  string
		count int
	}{
		{"last", r.Last}, {"daily", r.Daily}, {"weekly", r.Weekly}, {"monthly", r.Monthly}, {"yearly", r.Yearly},
	} {
		if rule.count > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", rule.name, rule.count))
		}
	}

	if r.Within > 0 {
		parts = append(parts, "within "+FormatWithin(r.Within))
	}

	return strings.Join(parts, ", ")
}

type Policy struct {
	rules Rules
}

func NewPolicy(rules Rules) (Policy, error) {
	if err := rules.Validate(); err != nil {
		return Policy{}, err
	}

	return Policy{rules: rules}, nil
}

func (p Policy) Rules() Rules {
	return p.rules
}

func ParseWithin(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}

	for suffix, unit := range map[string]time.Duration{"d": day, "w": week} {
		number, ok := strings.CutSuffix(value, suffix)
		if !ok {
			continue
		}
		count, err := strconv.Atoi(number)
		if err != nil || count < 0 {
			return 0, fmt.Errorf("%w: %q", ErrInvalidDuration, value)
		}
		return time.Duration(count) * unit, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 {
		return 0, fmt.Errorf("%w: %q", ErrInvalidDuration, value)
	}

	return duration, nil
}

func FormatWithin(duration time.Duration) string {
	switch {
	case duration == 0:
		return ""
	case duration%week == 0:
		return strconv.Itoa(int(duration/week)) + "w"
	case duration%day == 0:
		return strconv.Itoa(int(duration/day)) + "d"
	case duration%time.Hour == 0:
		return strconv.Itoa(int(duration/time.Hour)) + "h"
	default:
		return duration.String()
	}
}
