package money

import "bytes"

// NullMoney is a value that may be absent (ADR-0031).
//
// Money cannot carry absence, because its zero value is 0.0000 and that is a
// real amount. A product with no SKUs has no `base_price` at all (ADR-0027),
// and this type is where that state lives. A nullable column scans into it,
// and it writes null back.
//
// Valid states whether the value is there. Money is the zero value when
// Valid is false, and a caller reads Money only after it has read Valid.
type NullMoney struct {
	Money Money
	Valid bool
}

// MarshalJSON writes the amount as a string, or null when it is absent.
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

// UnmarshalJSON reads the amount from a JSON string, or from null.
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

// Scan reads the amount from a nullable NUMERIC(15,4) column.
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
