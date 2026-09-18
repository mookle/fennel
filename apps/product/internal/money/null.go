package money

import "bytes"

// NullMoney is a value that may be absent (ADR-0031).
//
// Money cannot carry absence, because its zero amount is 0.0000 and that is a
// real amount. A product with no SKUs has no `base_price` at all (ADR-0027),
// and this type is where that state lives. Absence is absence of the whole
// value, never of one half (ADR-0032): a read routes an absent base_price to
// a plain NULL, which scans into this type, and it writes null back.
//
// Valid states whether the value is there. Money is the zero value when
// Valid is false, and a caller reads Money only after it has read Valid.
type NullMoney struct {
	Money Money
	Valid bool
}

// MarshalJSON writes the pair as an object, or null when it is absent.
func (n NullMoney) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.Money.MarshalJSON()
}

// isJSONNull reports whether b is the JSON null literal. encoding/json reads
// null into a string as a no-op rather than as an error, so every decoder in
// this package has to rule it out before it decodes.
func isJSONNull(b []byte) bool {
	return bytes.Equal(bytes.TrimSpace(b), []byte("null"))
}

// UnmarshalJSON reads the pair from a JSON object, or from null.
func (n *NullMoney) UnmarshalJSON(b []byte) error {
	if isJSONNull(b) {
		*n = NullMoney{}
		return nil
	}

	var m Money
	if err := m.UnmarshalJSON(b); err != nil {
		return err
	}
	*n = NullMoney{Money: m, Valid: true}
	return nil
}

// Scan reads the pair from a nullable composite of (amount, currency). A
// read of base_price routes the absent case to a plain NULL rather than to a
// composite with an empty amount, so the NULL is the only absent form this
// method sees.
func (n *NullMoney) Scan(src any) error {
	if src == nil {
		*n = NullMoney{}
		return nil
	}

	var m Money
	if err := m.Scan(src); err != nil {
		return err
	}
	*n = NullMoney{Money: m, Valid: true}
	return nil
}
