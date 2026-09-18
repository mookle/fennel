package money

import (
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

// Amount is an exact decimal at a scale of four places.
//
// It carries no currency, and it has no arithmetic, because an amount without
// a currency has no meaning as money. It exists to cross the two boundaries
// that carry the scalar alone: the NUMERIC(15,4) column, and a request that
// supplies a price for a product whose currency the service already holds
// (ADR-0028, ADR-0032). Money is the type that pairs it with a Currency.
type Amount struct {
	units int64
}

// fromUnits builds an Amount from a count of scale-4 units, so 125000 is
// 12.5000. Every arithmetic result passes through it, and it holds the only
// range check those results get. ParseAmount builds its own value, because
// its grammar already bounds the result.
func fromUnits(units int64) (Amount, error) {
	if units > maxUnits || units < -maxUnits {
		return Amount{}, fmt.Errorf("money: %d units: %w", units, ErrRange)
	}
	return Amount{units: units}, nil
}

// ParseAmount reads the wire form. It takes one to eleven integer digits, an
// optional minus sign, and zero to four decimal places, which is the pattern
// the OpenAPI contract states. It pads a short input to the full scale, so
// "12.5" and "12.5000" parse alike.
func ParseAmount(s string) (Amount, error) {
	unsigned, negative := strings.CutPrefix(s, "-")

	whole, fraction, decimal := strings.Cut(unsigned, ".")
	if len(whole) < 1 || len(whole) > maxIntegerDigits || !allDigits(whole) {
		return Amount{}, fmt.Errorf("money: parse %q: %w", s, ErrSyntax)
	}
	if decimal && (len(fraction) < 1 || len(fraction) > Scale || !allDigits(fraction)) {
		return Amount{}, fmt.Errorf("money: parse %q: %w", s, ErrSyntax)
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
	return Amount{units: units}, nil
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
func (a Amount) String() string {
	units, sign := a.units, ""
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
// quoted form needs no escaping and AppendQuote matches JSON exactly.
func (a Amount) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, a.String()), nil
}

// UnmarshalJSON reads the amount from a JSON string. It rejects a JSON number
// and it rejects null, because Amount holds no absent value. A nullable field
// reads into a NullMoney (ADR-0031).
func (a *Amount) UnmarshalJSON(b []byte) error {
	if isJSONNull(b) {
		return fmt.Errorf("money: unmarshal null: %w", ErrNull)
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("money: unmarshal %s: %w", b, ErrSyntax)
	}

	parsed, err := ParseAmount(s)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}

// Value writes the amount to a NUMERIC(15,4) column. The driver reads the
// string back as an exact decimal.
func (a Amount) Value() (driver.Value, error) {
	return a.String(), nil
}

// Scan reads the amount from a NUMERIC(15,4) column.
//
// It rejects a float, which a driver only produces once the value has lost
// precision, and it reports that as ErrPrecision rather than as a malformed
// string, because the text was never the problem. It rejects a NULL, because
// Amount holds no absent value. A nullable column reads into a NullMoney
// (ADR-0031).
func (a *Amount) Scan(src any) error {
	switch v := src.(type) {
	case string:
		parsed, err := ParseAmount(v)
		if err != nil {
			return err
		}
		*a = parsed
	case []byte:
		parsed, err := ParseAmount(string(v))
		if err != nil {
			return err
		}
		*a = parsed
	case nil:
		return fmt.Errorf("money: scan null: %w", ErrNull)
	case float64, float32:
		return fmt.Errorf("money: scan %T: %w", src, ErrPrecision)
	default:
		return fmt.Errorf("money: scan %T: %w", src, ErrSyntax)
	}
	return nil
}
