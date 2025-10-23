package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// Money represents a monetary value with proper rounding for accounting
// All operations are rounded to 2 decimal places to comply with accounting standards
type Money float64

// NewMoney creates a Money value rounded to 2 decimal places
func NewMoney(value float64) Money {
	return Money(math.Round(value*100) / 100)
}

// MarshalJSON serializes to JSON (already rounded)
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(m))
}

// UnmarshalJSON handles deserialization from JSON with rounding
func (m *Money) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	*m = NewMoney(f)
	return nil
}

// Scan implements the sql.Scanner interface with rounding
func (m *Money) Scan(value interface{}) error {
	if value == nil {
		*m = 0
		return nil
	}

	switch v := value.(type) {
	case float64:
		*m = NewMoney(v)
	case float32:
		*m = NewMoney(float64(v))
	case int64:
		*m = Money(v)
	case int:
		*m = Money(v)
	case []byte:
		// Handle PostgreSQL numeric/decimal as bytes
		f, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return fmt.Errorf("cannot parse []byte to Money: %v", err)
		}
		*m = NewMoney(f)
	case string:
		// Handle string representation
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("cannot parse string to Money: %v", err)
		}
		*m = NewMoney(f)
	default:
		return fmt.Errorf("cannot scan type %T into Money", value)
	}
	return nil
}

// Value implements the driver.Valuer interface
func (m Money) Value() (driver.Value, error) {
	return float64(m), nil
}

// Float64 returns the float64 value (already rounded)
func (m Money) Float64() float64 {
	return float64(m)
}

// Add adds two Money values with proper rounding
func (m Money) Add(other Money) Money {
	return NewMoney(float64(m) + float64(other))
}

// Sub subtracts two Money values with proper rounding
func (m Money) Sub(other Money) Money {
	return NewMoney(float64(m) - float64(other))
}

// Mul multiplies Money by a quantity with proper rounding
func (m Money) Mul(quantity float64) Money {
	return NewMoney(float64(m) * quantity)
}

// Div divides Money by a divisor with proper rounding
func (m Money) Div(divisor float64) Money {
	if divisor == 0 {
		return 0
	}
	return NewMoney(float64(m) / divisor)
}

// String returns formatted money string
func (m Money) String() string {
	return fmt.Sprintf("%.2f", float64(m))
}
