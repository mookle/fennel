// Package money holds the one representation of a monetary value that
// storage, the wire and memory share (ADR-0017).
//
// A value is a decimal at a fixed scale of four places, which covers the
// widest ISO-4217 minor unit. It never becomes a binary float, and it
// crosses the wire as a string, never as a JSON number.
package money

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Scale is the number of decimal places every amount carries. It matches
// the scale of the NUMERIC(15,4) column.
const Scale = 4

const (
	scaleFactor      = 10_000
	maxIntegerDigits = 15 - Scale

	// maxUnits is the unit count of the largest value NUMERIC(15,4) holds,
	// which is 99999999999.9999.
	maxUnits = 999_999_999_999_999
)

// Money is an exact decimal amount at a scale of four places.
//
// It carries no currency. An amount without a currency has no meaning, so
// every entity that holds a Money holds a currency code beside it, and a
// product holds its currency from creation, before it has a price at all
// (ADR-0027, ADR-0028). Arithmetic across two currencies has no defined
// result, and the caller that owns both currencies is the only place that
// can enforce this.
type Money struct {
	units int64
}

// Parse reads the wire form. It takes one to eleven integer digits, an
// optional minus sign, and zero to four decimal places, which is the pattern
// the OpenAPI contract states. It pads a short input to the full scale, so
// "12.5" and "12.5000" parse alike.
func Parse(s string) (Money, error) {
	unsigned, negative := strings.CutPrefix(s, "-")

	whole, fraction, decimal := strings.Cut(unsigned, ".")
	if len(whole) < 1 || len(whole) > maxIntegerDigits || !allDigits(whole) {
		return Money{}, fmt.Errorf("money: parse %q: %w", s, ErrSyntax)
	}
	if decimal && (len(fraction) < 1 || len(fraction) > Scale || !allDigits(fraction)) {
		return Money{}, fmt.Errorf("money: parse %q: %w", s, ErrSyntax)
	}

	// Eleven integer digits and four decimal digits reach 999999999999999
	// units, which is exactly maxUnits, so the accumulation below cannot
	// leave the range and needs no check.
	var units int64
	for _, r := range whole {
		units = units*10 + int64(r-'0')
	}
	for i := range Scale {
		units *= 10
		if i < len(fraction) {
			units += int64(fraction[i] - '0')
		}
	}
	if negative {
		units = -units
	}
	return Money{units: units}, nil
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// String returns the wire form, padded to exactly four decimal places. A yen
// amount reads "1000.0000", because padding a currency's own minor unit would
// round away a price quoted finer than it (ADR-0017).
func (m Money) String() string {
	units, sign := m.units, ""
	if units < 0 {
		units, sign = -units, "-"
	}
	return fmt.Sprintf("%s%d.%0*d", sign, units/scaleFactor, Scale, units%scaleFactor)
}

// MarshalJSON writes the amount as a string. JSON parsers commonly read a
// number as an IEEE 754 double, which loses the precision this type exists
// to keep.
//
// String emits a minus sign, digits and a point and nothing else, so the
// quoted form needs no escaping.
func (m Money) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, m.String()), nil
}

// isJSONNull reports whether b is the JSON null literal. encoding/json reads
// null into a string as a no-op rather than as an error, so UnmarshalJSON
// has to rule it out before it decodes.
func isJSONNull(b []byte) bool {
	return bytes.Equal(bytes.TrimSpace(b), []byte("null"))
}

// UnmarshalJSON reads the amount from a JSON string. It rejects a JSON number
// and it rejects null, because Money holds no absent value.
func (m *Money) UnmarshalJSON(b []byte) error {
	if isJSONNull(b) {
		return fmt.Errorf("money: unmarshal null: %w", ErrNull)
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("money: unmarshal %s: %w", b, ErrSyntax)
	}

	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// Value writes the amount to a NUMERIC(15,4) column. The driver reads the
// string back as an exact decimal.
func (m Money) Value() (driver.Value, error) {
	return m.String(), nil
}

// Scan reads the amount from a NUMERIC(15,4) column.
//
// It rejects a float, which a driver only produces once the value has lost
// precision, and it reports that as ErrPrecision rather than as a malformed
// string, because the text was never the problem. It rejects a NULL, because
// Money holds no absent value.
func (m *Money) Scan(src any) error {
	switch v := src.(type) {
	case string:
		parsed, err := Parse(v)
		if err != nil {
			return err
		}
		*m = parsed
	case []byte:
		parsed, err := Parse(string(v))
		if err != nil {
			return err
		}
		*m = parsed
	case nil:
		return fmt.Errorf("money: scan null: %w", ErrNull)
	case float64, float32:
		return fmt.Errorf("money: scan %T: %w", src, ErrPrecision)
	default:
		return fmt.Errorf("money: scan %T: %w", src, ErrSyntax)
	}
	return nil
}
