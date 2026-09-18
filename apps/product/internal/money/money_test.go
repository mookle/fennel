package money

import (
	"encoding/json"
	"errors"
	"testing"
)

// mustParse builds an Money from the wire form, or fails the test.
func mustParse(t *testing.T, s string) Money {
	t.Helper()
	a, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return a
}

func TestParseAccepts(t *testing.T) {
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
		got := mustParse(t, c.in)
		if got.String() != c.want {
			t.Errorf("Parse(%q).String() = %q, want %q", c.in, got.String(), c.want)
		}
	}
}

func TestParseRejects(t *testing.T) {
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
		if _, err := Parse(in); !errors.Is(err, ErrSyntax) {
			t.Errorf("Parse(%q) error = %v, want ErrSyntax", in, err)
		}
	}
}

func TestMoneyStringPadsToScale(t *testing.T) {
	// A yen amount reads with four places, because padding a currency's own
	// minor unit would round away a price quoted finer than it (ADR-0017).
	if got := mustParse(t, "1000").String(); got != "1000.0000" {
		t.Errorf("String() = %q, want %q", got, "1000.0000")
	}
	if got := (Money{}).String(); got != "0.0000" {
		t.Errorf("zero Money String() = %q, want %q", got, "0.0000")
	}
	if got := mustParse(t, "-0.5").String(); got != "-0.5000" {
		t.Errorf("String() = %q, want %q", got, "-0.5000")
	}
}

func TestMoneyJSONRoundTrip(t *testing.T) {
	for _, in := range []string{"0.0000", "12.5000", "-0.0001", "99999999999.9999"} {
		b, err := json.Marshal(mustParse(t, in))
		if err != nil {
			t.Fatalf("Marshal(%q): %v", in, err)
		}
		if string(b) != `"`+in+`"` {
			t.Errorf("Marshal(%q) = %s, want %q", in, b, in)
		}

		var back Money
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("Unmarshal(%s): %v", b, err)
		}
		if back != mustParse(t, in) {
			t.Errorf("round trip of %q gave %q", in, back)
		}
	}
}

func TestMoneyUnmarshalJSONRejectsNonString(t *testing.T) {
	// A JSON number is the form ADR-0017 keeps money out of.
	for _, in := range []string{"12.50", "12", "true", "[]", "{}", "1e3"} {
		var a Money
		if err := a.UnmarshalJSON([]byte(in)); !errors.Is(err, ErrSyntax) {
			t.Errorf("UnmarshalJSON(%s) error = %v, want ErrSyntax", in, err)
		}
	}
}

func TestMoneyUnmarshalJSONRejectsNull(t *testing.T) {
	// encoding/json reads null into a string as a no-op, so this has to be
	// ruled out on its own. Money holds no absent value, and the error points
	// at the type that does (ADR-0031).
	var a Money
	if err := a.UnmarshalJSON([]byte("null")); !errors.Is(err, ErrNull) {
		t.Errorf("UnmarshalJSON(null) error = %v, want ErrNull", err)
	}
}

func TestMoneyUnmarshalJSONRejectsGoSyntax(t *testing.T) {
	// A raw string literal and a rune literal are Go forms, not JSON ones.
	// encoding/json never produces either, and neither may slip past.
	for _, in := range []string{"`12.50`", `'a'`} {
		var a Money
		if err := a.UnmarshalJSON([]byte(in)); err == nil {
			t.Errorf("UnmarshalJSON(%s) = %q, want an error", in, a)
		}
	}
}

func TestMoneyScan(t *testing.T) {
	var a Money
	if err := a.Scan("12.5"); err != nil || a.String() != "12.5000" {
		t.Errorf("Scan(string) = %q, %v", a, err)
	}
	if err := a.Scan([]byte("0.0125")); err != nil || a.String() != "0.0125" {
		t.Errorf("Scan([]byte) = %q, %v", a, err)
	}
}

func TestMoneyScanRejects(t *testing.T) {
	cases := []struct {
		src  any
		want error
	}{
		// A NULL is absence, and Money holds none. NullMoney does.
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
		var a Money
		if err := a.Scan(c.src); !errors.Is(err, c.want) {
			t.Errorf("Scan(%#v) error = %v, want %v", c.src, err, c.want)
		}
	}
}

func TestMoneyValue(t *testing.T) {
	v, err := mustParse(t, "12.5").Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if v != "12.5000" {
		t.Errorf("Value() = %#v, want %q", v, "12.5000")
	}
}
