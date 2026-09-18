package money

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNullMoneyZeroValueIsAbsent(t *testing.T) {
	// Absence is the zero value, so a NullMoney nobody has written reads as
	// absent rather than as 0.0000.
	var n NullMoney
	if n.Valid {
		t.Error("the zero value of NullMoney reports as present")
	}
}

func TestNullMoneyJSON(t *testing.T) {
	present := NullMoney{Money: mustParse(t, "12.5"), Valid: true}
	b, err := json.Marshal(present)
	if err != nil || string(b) != `"12.5000"` {
		t.Errorf("Marshal(present) = %s, %v", b, err)
	}

	b, err = json.Marshal(NullMoney{})
	if err != nil || string(b) != "null" {
		t.Errorf("Marshal(absent) = %s, %v", b, err)
	}

	var back NullMoney
	if err := json.Unmarshal([]byte(`"12.5000"`), &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !back.Valid || back.Money != present.Money {
		t.Errorf("Unmarshal of a string gave %+v", back)
	}

	back = present
	if err := json.Unmarshal([]byte("null"), &back); err != nil {
		t.Fatalf("Unmarshal(null): %v", err)
	}
	if back.Valid || back.Money != (Money{}) {
		t.Errorf("Unmarshal(null) gave %+v, want the zero value", back)
	}
}

func TestNullMoneyUnmarshalJSONRejectsNonString(t *testing.T) {
	// The nullable form relaxes presence, not the wire shape.
	var n NullMoney
	if err := json.Unmarshal([]byte("12.50"), &n); !errors.Is(err, ErrSyntax) {
		t.Errorf("Unmarshal(12.50) error = %v, want ErrSyntax", err)
	}
}

func TestNullMoneyScan(t *testing.T) {
	// The absent base_price of a product with no SKUs arrives here as a NULL
	// (ADR-0027).
	var n NullMoney
	if err := n.Scan(nil); err != nil || n.Valid {
		t.Errorf("Scan(nil) = %+v, %v", n, err)
	}

	if err := n.Scan("12.5"); err != nil || !n.Valid || n.Money != mustParse(t, "12.5") {
		t.Errorf("Scan(string) = %+v, %v", n, err)
	}

	// A present value in the wrong form is still an error.
	if err := n.Scan("12.50000"); !errors.Is(err, ErrSyntax) {
		t.Errorf("Scan(malformed) error = %v, want ErrSyntax", err)
	}
}

func TestNullMoneyScanOverwritesAPresentValue(t *testing.T) {
	// Reusing a destination across rows must not leave the previous row's
	// value behind when this row is NULL.
	n := NullMoney{Money: mustParse(t, "12.5"), Valid: true}
	if err := n.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if n.Valid || n.Money != (Money{}) {
		t.Errorf("Scan(nil) over a present value gave %+v", n)
	}
}
