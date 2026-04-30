package http

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultHistoryLimit = 100
	maxHistoryLimit     = 1000
)

func parseOptionalTimeParam(values url.Values, name string) (time.Time, error) {
	rawValue := values.Get(name)
	if rawValue == "" {
		return time.Time{}, nil
	}

	parsedTime, err := time.Parse(time.RFC3339, rawValue)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339 datetime", name)
	}

	return parsedTime, nil
}

func parseLimitParam(values url.Values) (int, error) {
	rawValue := values.Get("limit")
	if rawValue == "" {
		return defaultHistoryLimit, nil
	}

	limit, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, fmt.Errorf("limit must be integer")
	}

	if limit <= 0 {
		return 0, fmt.Errorf("limit must be positive")
	}

	if limit > maxHistoryLimit {
		return maxHistoryLimit, nil
	}

	return limit, nil
}

func validateTimeRange(from time.Time, to time.Time) error {
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return fmt.Errorf("from must be before or equal to to")
	}

	return nil
}
