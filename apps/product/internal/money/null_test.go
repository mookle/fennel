package money

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNullMoneyZeroValueIsAbsent(t *testing.T) {
	// Absence is the zero value, so a NullMoney nobody has written reads as
	// absent rather than as 0.0000 in no currency.
	var n NullMoney
	if n.Valid {
		t.Error("the zero value of NullMoney reports as present")
	}
}

func TestNullMoneyJSON(t *testing.T) {
	present := NullMoney{Money: gbp(t, "12.5"), Valid: true}
	b, err := json.Marshal(present)
	if err != nil || string(b) != `{"amount":"12.5000","currency":"GBP"}` {
		t.Errorf("Marshal(present) = %s, %v", b, err)
	}

	b, err = json.Marshal(NullMoney{})
	if err != nil || string(b) != "null" {
		t.Errorf("Marshal(absent) = %s, %v", b, err)
	}

	var back NullMoney
	if err := json.Unmarshal([]byte(`{"amount":"12.5000","currency":"GBP"}`), &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !back.Valid || !back.Money.Equal(present.Money) {
		t.Errorf("Unmarshal of an object gave %+v", back)
	}

	back = present
	if err := json.Unmarshal([]byte("null"), &back); err != nil {
		t.Fatalf("Unmarshal(null): %v", err)
	}
	if back.Valid || back.Money != (Money{}) {
		t.Errorf("Unmarshal(null) gave %+v, want the zero value", back)
	}
}

func TestNullMoneyUnmarshalJSONRejectsHalfAValue(t *testing.T) {
	// The nullable form relaxes presence of the whole value, not the wire
	// shape and not the presence of one half (ADR-0032).
	cases := []struct {
		in   string
		want error
	}{
		{`"12.5000"`, ErrSyntax},
		{"12.50", ErrSyntax},
		{`{"amount":"12.5000"}`, ErrNull},
		{`{"amount":"12.5000","currency":null}`, ErrNull},
	}
	for _, c := range cases {
		var n NullMoney
		if err := json.Unmarshal([]byte(c.in), &n); !errors.Is(err, c.want) {
			t.Errorf("Unmarshal(%s) error = %v, want %v", c.in, err, c.want)
		}
	}
}

func TestNullMoneyScan(t *testing.T) {
	// The absent base_price of a product with no SKUs arrives here as a NULL
	// (ADR-0027), because the read routes it there rather than composing a
	// pair with an empty amount.
	var n NullMoney
	if err := n.Scan(nil); err != nil || n.Valid {
		t.Errorf("Scan(nil) = %+v, %v", n, err)
	}

	if err := n.Scan("(12.5,GBP)"); err != nil || !n.Valid || !n.Money.Equal(gbp(t, "12.5")) {
		t.Errorf("Scan(composite) = %+v, %v", n, err)
	}

	// A present value that is not a pair is still an error, and so is a
	// pair with one half.
	if err := n.Scan("12.5"); !errors.Is(err, ErrSyntax) {
		t.Errorf("Scan(bare amount) error = %v, want ErrSyntax", err)
	}
	if err := n.Scan("(,GBP)"); !errors.Is(err, ErrNull) {
		t.Errorf("Scan(empty amount) error = %v, want ErrNull", err)
	}
}

func TestNullMoneyScanOverwritesAPresentValue(t *testing.T) {
	// Reusing a destination across rows must not leave the previous row's
	// value behind when this row is NULL.
	n := NullMoney{Money: gbp(t, "12.5"), Valid: true}
	if err := n.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if n.Valid || n.Money != (Money{}) {
		t.Errorf("Scan(nil) over a present value gave %+v", n)
	}
}
