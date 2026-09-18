package money

import (
	"encoding/json"
	"errors"
	"testing"
)

// mustParseAmount builds an Amount from the wire form, or fails the test.
func mustParseAmount(t *testing.T, s string) Amount {
	t.Helper()
	a, err := ParseAmount(s)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", s, err)
	}
	return a
}

func TestParseAmountAccepts(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0", "0.0000"},
		{"12.5", "12.5000"},
		{"12.5000", "12.5000"},
		{"12.0125", "12.0125"},
		{"-0.0001", "-0.0001"},
		{"-12.5", "-12.5000"},
		{"1000", "1000.0000"},
		{"00000000012.5", "12.5000"},
		{"99999999999.9999", "99999999999.9999"},
		{"-99999999999.9999", "-99999999999.9999"},
		{"-0", "0.0000"},
		{"-0.0000", "0.0000"},
	}
	for _, c := range cases {
		got := mustParseAmount(t, c.in)
		if got.String() != c.want {
			t.Errorf("ParseAmount(%q).String() = %q, want %q", c.in, got.String(), c.want)
		}
	}
}

func TestParseAmountRejects(t *testing.T) {
	// Every one of these fails the contract's pattern, so every one is a
	// syntax error rather than a range error.
	cases := []string{
		"",
		"-",
		"+1",
		".5",
		"12.",
		"12.50000",          // five decimal places
		"999999999999",      // twelve integer digits
		"999999999999.9999", // twelve integer digits with a fraction
		"1e3",
		"12,50",
		" 12.50",
		"12.50 ",
		"1_000",
		"١٢",
		"NaN",
		"Inf",
		"--1",
		"1.2.3",
	}
	for _, in := range cases {
		if _, err := ParseAmount(in); !errors.Is(err, ErrSyntax) {
			t.Errorf("ParseAmount(%q) error = %v, want ErrSyntax", in, err)
		}
	}
}

func TestAmountStringPadsToScale(t *testing.T) {
	// A yen amount reads with four places, because padding a currency's own
	// minor unit would round away a price quoted finer than it (ADR-0017).
	if got := mustParseAmount(t, "1000").String(); got != "1000.0000" {
		t.Errorf("String() = %q, want %q", got, "1000.0000")
	}
	if got := (Amount{}).String(); got != "0.0000" {
		t.Errorf("zero Amount String() = %q, want %q", got, "0.0000")
	}
	if got := mustParseAmount(t, "-0.5").String(); got != "-0.5000" {
		t.Errorf("String() = %q, want %q", got, "-0.5000")
	}
}

func TestAmountJSONRoundTrip(t *testing.T) {
	for _, in := range []string{"0.0000", "12.5000", "-0.0001", "99999999999.9999"} {
		b, err := json.Marshal(mustParseAmount(t, in))
		if err != nil {
			t.Fatalf("Marshal(%q): %v", in, err)
		}
		if string(b) != `"`+in+`"` {
			t.Errorf("Marshal(%q) = %s, want %q", in, b, in)
		}

		var back Amount
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("Unmarshal(%s): %v", b, err)
		}
		if back != mustParseAmount(t, in) {
			t.Errorf("round trip of %q gave %q", in, back)
		}
	}
}

func TestAmountUnmarshalJSONRejectsNonString(t *testing.T) {
	// A JSON number is the form ADR-0017 keeps money out of.
	for _, in := range []string{"12.50", "12", "true", "[]", "{}", "1e3"} {
		var a Amount
		if err := a.UnmarshalJSON([]byte(in)); !errors.Is(err, ErrSyntax) {
			t.Errorf("UnmarshalJSON(%s) error = %v, want ErrSyntax", in, err)
		}
	}
}

func TestAmountUnmarshalJSONRejectsNull(t *testing.T) {
	// encoding/json reads null into a string as a no-op, so this has to be
	// ruled out on its own. Amount holds no absent value, and the error points
	// at the type that does (ADR-0031).
	var a Amount
	if err := a.UnmarshalJSON([]byte("null")); !errors.Is(err, ErrNull) {
		t.Errorf("UnmarshalJSON(null) error = %v, want ErrNull", err)
	}
}

func TestAmountUnmarshalJSONRejectsGoSyntax(t *testing.T) {
	// A raw string literal and a rune literal are Go forms, not JSON ones.
	// encoding/json never produces either, and neither may slip past.
	for _, in := range []string{"`12.50`", `'a'`} {
		var a Amount
		if err := a.UnmarshalJSON([]byte(in)); err == nil {
			t.Errorf("UnmarshalJSON(%s) = %q, want an error", in, a)
		}
	}
}

func TestAmountScan(t *testing.T) {
	var a Amount
	if err := a.Scan("12.5"); err != nil || a.String() != "12.5000" {
		t.Errorf("Scan(string) = %q, %v", a, err)
	}
	if err := a.Scan([]byte("0.0125")); err != nil || a.String() != "0.0125" {
		t.Errorf("Scan([]byte) = %q, %v", a, err)
	}
}

func TestAmountScanRejects(t *testing.T) {
	cases := []struct {
		src  any
		want error
	}{
		// A NULL is absence, and Amount holds none. NullMoney does.
		{nil, ErrNull},
		// A float reached this point having already lost precision, so the
		// text was never the problem.
		{float64(12.5), ErrPrecision},
		{float32(12.5), ErrPrecision},
		// A driver type this package cannot read.
		{int64(12), ErrSyntax},
		{true, ErrSyntax},
		// Text in the wrong form.
		{"12.50000", ErrSyntax},
	}
	for _, c := range cases {
		var a Amount
		if err := a.Scan(c.src); !errors.Is(err, c.want) {
			t.Errorf("Scan(%#v) error = %v, want %v", c.src, err, c.want)
		}
	}
}

func TestAmountValue(t *testing.T) {
	v, err := mustParseAmount(t, "12.5").Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if v != "12.5000" {
		t.Errorf("Value() = %#v, want %q", v, "12.5000")
	}
}
