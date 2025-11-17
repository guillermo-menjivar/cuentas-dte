package models

import (
	"strings"
	"time"
)

// DateOnly is a custom type for date-only fields (YYYY-MM-DD)
type DateOnly struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for date-only format
func (d *DateOnly) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		return nil
	}

	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}

	d.Time = t
	return nil
}

// MarshalJSON implements custom JSON marshaling for date-only format
func (d DateOnly) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte("\"" + d.Time.Format("2006-01-02") + "\""), nil
}
