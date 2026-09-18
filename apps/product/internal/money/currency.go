package money

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/moov-io/iso4217"
)

// notCurrency holds a registry of ISO 4217 codes that represent something
// other than a national currency: the precious metals, the fund codes and
// units of account, and the two special codes for a transaction with no
// currency and for testing.
var notCurrency = map[string]struct{}{
	// Precious metals
	"XAG": {}, "XAU": {}, "XPD": {}, "XPT": {},

	// Bond market units
	"XBA": {}, "XBB": {}, "XBC": {}, "XBD": {},

	// Units of account and special codes
	"UYW": {}, "XAD": {}, "XDR": {}, "XSU": {}, "XTS": {}, "XUA": {}, "XXX": {},

	// Funds and index units
	"BOV": {}, "CHE": {}, "CHW": {}, "CLF": {},
	"CNH": {}, "COU": {}, "MXV": {}, "USN": {}, "UYI": {},
}

// Currency is an ISO 4217 code that names a currency, current or withdrawn
// (ADR-0030). It is the currency half of a Money (ADR-0032), and it stands
// alone on a product as the shop's stand-in (ADR-0029). Arithmetic between
// two of them has no defined result (ADR-0017).
//
// ParseCurrency is the only way to populate one, but Go always admits the
// zero value, and an empty code is not a currency. No currency field is
// optional in the contract, so the zero value is a bug rather than an
// absent value, and both MarshalJSON and Value refuse it.
type Currency struct {
	code string
}

// ParseCurrency reads a code. It is the only place a Currency is
// constructed, and where we check that the code names a currency.
func ParseCurrency(s string) (Currency, error) {
	entry, ok := iso4217.Lookup(s)
	if !ok || entry.Code != s {
		return Currency{}, fmt.Errorf("money: parse currency %q: %w", s, ErrCurrency)
	}
	if _, excluded := notCurrency[s]; excluded {
		return Currency{}, fmt.Errorf("money: parse currency %q: %w", s, ErrCurrency)
	}
	return Currency{code: s}, nil
}

// String returns the code.
func (c Currency) String() string {
	return c.code
}

// MarshalJSON writes the code as a string. It refuses the zero value.
func (c Currency) MarshalJSON() ([]byte, error) {
	if c.code == "" {
		return nil, fmt.Errorf("money: marshal currency: %w", ErrCurrency)
	}
	return strconv.AppendQuote(nil, c.code), nil
}

// UnmarshalJSON reads the code from a JSON string. It rejects null, because
// no currency field in the contract is optional.
func (c *Currency) UnmarshalJSON(b []byte) error {
	if isJSONNull(b) {
		return fmt.Errorf("money: unmarshal currency null: %w", ErrNull)
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("money: unmarshal %s: %w", b, ErrSyntax)
	}

	parsed, err := ParseCurrency(s)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

// Value writes the code to a text column. It refuses the zero value.
func (c Currency) Value() (driver.Value, error) {
	if c.code == "" {
		return nil, fmt.Errorf("money: value currency: %w", ErrCurrency)
	}
	return c.code, nil
}

// Scan reads the code from a text column. It rejects a NULL, because no
// currency column in this build is nullable.
func (c *Currency) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case nil:
		return fmt.Errorf("money: scan currency null: %w", ErrNull)
	default:
		return fmt.Errorf("money: scan %T: %w", src, ErrSyntax)
	}

	parsed, err := ParseCurrency(s)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
