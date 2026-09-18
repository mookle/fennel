// Package money holds the two types a monetary value takes in this service
// (ADR-0017, ADR-0032).
//
// Amount is the scalar: a decimal at a fixed scale of four places, which
// covers the widest ISO-4217 minor unit, and never a binary float. It is the
// type used in the two places an amount appears without its currency: the value
// of a NUMERIC(15,4) column, and the price on SkuWrite, where the product
// supplies the currency.
//
// Money pairs an Amount with a Currency and is the value the domain reasons
// about. It owns arithmetic and comparison, and it is the shape of every
// amount in a response.
package money

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Money is an amount and its currency, held as one value (ADR-0032).
//
// A value with one half is not admitted: the zero value is unusable, and
// MarshalJSON refuses it. Every method that takes a second Money reports
// ErrMismatch when the currencies differ (ADR-0032).
//
// A response carries a Money as an object with both halves. A column holds
// the Amount alone, and a read composes the pair from the amount column and
// the product's currency (ADR-0028), so Scan reads a composite and there is
// no Value. A write passes m.Amount. Absence is absence of the whole value,
// and NullMoney holds it (ADR-0031).
type Money struct {
	Amount   Amount
	Currency Currency
}

// moneyJSON is the wire shape. The contract names both fields required.
type moneyJSON struct {
	Amount   Amount   `json:"amount"`
	Currency Currency `json:"currency"`
}

// String returns the amount and the code, as "24.0000 GBP". It is for errors
// and logs, and nothing parses it.
func (m Money) String() string {
	return m.Amount.String() + " " + m.Currency.String()
}

// MarshalJSON writes the pair as an object. The zero value fails, because
// Currency refuses to marshal an empty code.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(moneyJSON{Amount: m.Amount, Currency: m.Currency})
}

// UnmarshalJSON reads the pair from a JSON object. It rejects null, a missing
// field and a null field as ErrNull, because Money holds no absent value and
// a value with one half is not admitted. A nullable field reads into a
// NullMoney (ADR-0031). Anything that is not an object is ErrSyntax.
func (m *Money) UnmarshalJSON(b []byte) error {
	if isJSONNull(b) {
		return fmt.Errorf("money: unmarshal null: %w", ErrNull)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return fmt.Errorf("money: unmarshal %s: %w", b, ErrSyntax)
	}

	amount, ok := fields["amount"]
	if !ok {
		return fmt.Errorf("money: unmarshal %s: no amount: %w", b, ErrNull)
	}
	currency, ok := fields["currency"]
	if !ok {
		return fmt.Errorf("money: unmarshal %s: no currency: %w", b, ErrNull)
	}

	var parsed Money
	if err := parsed.Amount.UnmarshalJSON(amount); err != nil {
		return err
	}
	if err := parsed.Currency.UnmarshalJSON(currency); err != nil {
		return err
	}
	*m = parsed
	return nil
}

// Scan reads the pair from a Postgres composite of (amount, currency), which
// a read composes from the amount column and the product's currency
// (ADR-0028, ADR-0032). The text form is "(24.0000,GBP)".
//
// It rejects a NULL, because Money holds no absent value, and a nullable
// column reads into a NullMoney (ADR-0031). It rejects text that is not a
// composite as ErrSyntax, which is what a bare amount column produces. It
// rejects an empty field as ErrNull, because a value with one half is not
// admitted. Each field then reads through Amount.Scan and Currency.Scan, so
// ErrRange and ErrCurrency come from there.
//
// Postgres double-quotes a field that holds a comma, a parenthesis, a quote,
// a backslash or whitespace, and it doubles a quote inside one. A NUMERIC and
// a three-letter code hold none of those, so this parser reads no quoting.
func (m *Money) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case nil:
		return fmt.Errorf("money: scan null: %w", ErrNull)
	default:
		return fmt.Errorf("money: scan %T: %w", src, ErrSyntax)
	}

	inner, ok := strings.CutPrefix(s, "(")
	if !ok {
		return fmt.Errorf("money: scan %q: %w", s, ErrSyntax)
	}
	inner, ok = strings.CutSuffix(inner, ")")
	if !ok {
		return fmt.Errorf("money: scan %q: %w", s, ErrSyntax)
	}
	amount, currency, ok := strings.Cut(inner, ",")
	if !ok || strings.Contains(currency, ",") {
		return fmt.Errorf("money: scan %q: %w", s, ErrSyntax)
	}
	if amount == "" || currency == "" {
		return fmt.Errorf("money: scan %q: %w", s, ErrNull)
	}

	var parsed Money
	if err := parsed.Amount.Scan(amount); err != nil {
		return err
	}
	if err := parsed.Currency.Scan(currency); err != nil {
		return err
	}
	*m = parsed
	return nil
}

// same reports whether n shares m's currency, and names the pair in the error
// when it does not.
func (m Money) sameCurrency(n Money) error {
	if m.Currency != n.Currency {
		return fmt.Errorf("money: %s and %s: %w", m, n, ErrMismatch)
	}
	return nil
}

// Add returns the sum. Both operands sit inside NUMERIC(15,4), so only the
// result can leave the range.
func (m Money) Add(n Money) (Money, error) {
	if err := m.sameCurrency(n); err != nil {
		return Money{}, err
	}
	amount, err := fromUnits(m.Amount.units + n.Amount.units)
	if err != nil {
		return Money{}, err
	}
	return Money{Amount: amount, Currency: m.Currency}, nil
}

// Sub returns the difference. Both operands sit inside NUMERIC(15,4), so only
// the result can leave the range.
func (m Money) Sub(n Money) (Money, error) {
	if err := m.sameCurrency(n); err != nil {
		return Money{}, err
	}
	amount, err := fromUnits(m.Amount.units - n.Amount.units)
	if err != nil {
		return Money{}, err
	}
	return Money{Amount: amount, Currency: m.Currency}, nil
}

// Neg returns the value with its sign flipped. The range is symmetric, so
// this never leaves it.
func (m Money) Neg() Money {
	return Money{Amount: Amount{units: -m.Amount.units}, Currency: m.Currency}
}

// Mul returns the value taken quantity times, which is how a shipping line
// forms (ADR-0007). Nothing rounds, because the amount is at scale 4 and the
// quantity is a whole number. One operand carries a currency, so nothing can
// mismatch.
func (m Money) Mul(quantity int64) (Money, error) {
	units := m.Amount.units
	if units == 0 || quantity == 0 {
		return Money{Currency: m.Currency}, nil
	}
	if quantity == math.MinInt64 || abs(units) > maxUnits/abs(quantity) {
		return Money{}, fmt.Errorf("money: %s x %d: %w", m, quantity, ErrRange)
	}
	amount, err := fromUnits(units * quantity)
	if err != nil {
		return Money{}, err
	}
	return Money{Amount: amount, Currency: m.Currency}, nil
}

// Compare orders two values. It returns -1 when m is below n, 0 when they
// hold the same amount, and 1 when m is above n. It is the ordering
// base_price derives from (ADR-0027). Two values in different currencies
// have no order, and that is ErrMismatch.
func (m Money) Compare(n Money) (int, error) {
	if err := m.sameCurrency(n); err != nil {
		return 0, err
	}
	switch {
	case m.Amount.units < n.Amount.units:
		return -1, nil
	case m.Amount.units > n.Amount.units:
		return 1, nil
	default:
		return 0, nil
	}
}

// Equal reports whether two values hold the same amount in the same
// currency. Two values in different currencies are never equal, and that is
// a false rather than an error, because inequality is a defined result.
func (m Money) Equal(n Money) bool {
	return m.Currency == n.Currency && m.Amount.units == n.Amount.units
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
