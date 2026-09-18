package money

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

// mustMoney builds a Money from an amount's wire form and a code, or fails
// the test.
func mustMoney(t *testing.T, amount, code string) Money {
	t.Helper()
	c, err := ParseCurrency(code)
	if err != nil {
		t.Fatalf("ParseCurrency(%q): %v", code, err)
	}
	return Money{Amount: mustParseAmount(t, amount), Currency: c}
}

// gbp builds a GBP Money, which is the currency most cases share.
func gbp(t *testing.T, amount string) Money {
	t.Helper()
	return mustMoney(t, amount, "GBP")
}

func TestMoneyString(t *testing.T) {
	if got := gbp(t, "24").String(); got != "24.0000 GBP" {
		t.Errorf("String() = %q, want %q", got, "24.0000 GBP")
	}
}

func TestMoneyJSONRoundTrip(t *testing.T) {
	// The pair is one object on the wire, and both halves are required
	// (ADR-0032).
	b, err := json.Marshal(gbp(t, "24"))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"amount":"24.0000","currency":"GBP"}`
	if string(b) != want {
		t.Errorf("Marshal = %s, want %s", b, want)
	}

	var back Money
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("Unmarshal(%s): %v", b, err)
	}
	if !back.Equal(gbp(t, "24")) {
		t.Errorf("round trip gave %q", back)
	}

	// A request may carry a short amount, and an unknown field is not an
	// error, because the contract does not forbid one.
	if err := json.Unmarshal([]byte(`{"currency":"JPY","amount":"1000","note":1}`), &back); err != nil {
		t.Fatalf("Unmarshal of a short amount: %v", err)
	}
	if !back.Equal(mustMoney(t, "1000.0000", "JPY")) {
		t.Errorf("Unmarshal of a short amount gave %q", back)
	}
}

func TestMoneyMarshalJSONRefusesTheZeroValue(t *testing.T) {
	// A value with one half is not admitted, and the zero value has no
	// currency at all. Currency's own refusal is what fails it.
	if _, err := json.Marshal(Money{Amount: mustParseAmount(t, "24")}); !errors.Is(err, ErrCurrency) {
		t.Errorf("Marshal(no currency) error = %v, want ErrCurrency", err)
	}
	if _, err := json.Marshal(Money{}); !errors.Is(err, ErrCurrency) {
		t.Errorf("Marshal(zero value) error = %v, want ErrCurrency", err)
	}
}

func TestMoneyUnmarshalJSONRejectsAbsence(t *testing.T) {
	// null, a missing half and a null half are all absence, and Money holds
	// none. NullMoney holds the whole value's absence (ADR-0031), and a value
	// with one half is not admitted at all (ADR-0032).
	cases := []string{
		"null",
		"{}",
		`{"amount":"24.0000"}`,
		`{"currency":"GBP"}`,
		`{"amount":null,"currency":"GBP"}`,
		`{"amount":"24.0000","currency":null}`,
	}
	for _, in := range cases {
		var m Money
		if err := json.Unmarshal([]byte(in), &m); !errors.Is(err, ErrNull) {
			t.Errorf("Unmarshal(%s) error = %v, want ErrNull", in, err)
		}
	}
}

func TestMoneyUnmarshalJSONRejectsNonObject(t *testing.T) {
	// The flat string is the old wire form, and a number is the form
	// ADR-0017 keeps money out of. Neither is the pair.
	for _, in := range []string{`"24.0000"`, "24", "true", "[]", `["24.0000","GBP"]`} {
		var m Money
		if err := json.Unmarshal([]byte(in), &m); !errors.Is(err, ErrSyntax) {
			t.Errorf("Unmarshal(%s) error = %v, want ErrSyntax", in, err)
		}
	}
}

func TestMoneyUnmarshalJSONRejectsBadHalves(t *testing.T) {
	// Each half reads through its own decoder, so its own error comes out.
	cases := []struct {
		in   string
		want error
	}{
		{`{"amount":24,"currency":"GBP"}`, ErrSyntax},
		{`{"amount":"24.00000","currency":"GBP"}`, ErrSyntax},
		{`{"amount":"24.0000","currency":"XXX"}`, ErrCurrency},
		{`{"amount":"24.0000","currency":"gbp"}`, ErrCurrency},
		{`{"amount":"24.0000","currency":1}`, ErrSyntax},
	}
	for _, c := range cases {
		var m Money
		if err := json.Unmarshal([]byte(c.in), &m); !errors.Is(err, c.want) {
			t.Errorf("Unmarshal(%s) error = %v, want %v", c.in, err, c.want)
		}
	}
}

func TestMoneyScan(t *testing.T) {
	// A read composes the pair as a composite of the amount column and the
	// product's currency (ADR-0028, ADR-0032).
	var m Money
	if err := m.Scan("(24.0000,GBP)"); err != nil || !m.Equal(gbp(t, "24")) {
		t.Errorf("Scan(string) = %q, %v", m, err)
	}
	if err := m.Scan([]byte("(0.0125,JPY)")); err != nil || !m.Equal(mustMoney(t, "0.0125", "JPY")) {
		t.Errorf("Scan([]byte) = %q, %v", m, err)
	}
	// Postgres writes a NUMERIC at the column's scale, but the composite's
	// amount is the column's text, so a short form still reads.
	if err := m.Scan("(-12.5,EUR)"); err != nil || !m.Equal(mustMoney(t, "-12.5", "EUR")) {
		t.Errorf("Scan(short amount) = %q, %v", m, err)
	}
}

func TestMoneyScanRejects(t *testing.T) {
	cases := []struct {
		src  any
		want error
	}{
		// A NULL is absence, and Money holds none. NullMoney does.
		{nil, ErrNull},
		// A bare amount column is not the composite a read composes.
		{"24.0000", ErrSyntax},
		{"GBP", ErrSyntax},
		{"", ErrSyntax},
		{"(24.0000)", ErrSyntax},
		{"(24.0000,GBP", ErrSyntax},
		{"24.0000,GBP)", ErrSyntax},
		{"(24.0000,GBP,X)", ErrSyntax},
		// A value with one half is not admitted, so an empty field is
		// absence rather than a malformed value.
		{"(,GBP)", ErrNull},
		{"(24.0000,)", ErrNull},
		{"(,)", ErrNull},
		// Each half reads through its own scanner, so its own error comes out.
		{"(24.00000,GBP)", ErrSyntax},
		{"(999999999999.9999,GBP)", ErrSyntax},
		{"(24.0000,ZZZ)", ErrCurrency},
		{"(24.0000,XAU)", ErrCurrency},
		// A driver type this package cannot read.
		{float64(24), ErrSyntax},
		{int64(24), ErrSyntax},
	}
	for _, c := range cases {
		var m Money
		if err := m.Scan(c.src); !errors.Is(err, c.want) {
			t.Errorf("Scan(%#v) error = %v, want %v", c.src, err, c.want)
		}
	}
}

func TestMoneyScanOverwrites(t *testing.T) {
	// Reusing a destination across rows must not leave the previous row's
	// value behind on a failed scan, and a good scan replaces both halves.
	m := gbp(t, "24")
	if err := m.Scan("(24.0000,ZZZ)"); err == nil {
		t.Fatal("Scan of a bad currency passed")
	}
	if !m.Equal(gbp(t, "24")) {
		t.Errorf("a failed Scan changed the value to %q", m)
	}
	if err := m.Scan("(1000.0000,JPY)"); err != nil || !m.Equal(mustMoney(t, "1000", "JPY")) {
		t.Errorf("Scan over a present value gave %q, %v", m, err)
	}
}

func TestMoneyAdd(t *testing.T) {
	sum, err := gbp(t, "12.5").Add(gbp(t, "0.0125"))
	if err != nil || sum.String() != "12.5125 GBP" {
		t.Errorf("Add = %q, %v", sum, err)
	}

	// Both operands sit inside the range, so only the result can leave it.
	if _, err := gbp(t, "99999999999.9999").Add(gbp(t, "0.0001")); !errors.Is(err, ErrRange) {
		t.Errorf("Add past the maximum error = %v, want ErrRange", err)
	}
	if _, err := gbp(t, "-99999999999.9999").Add(gbp(t, "-0.0001")); !errors.Is(err, ErrRange) {
		t.Errorf("Add past the minimum error = %v, want ErrRange", err)
	}
}

func TestMoneySub(t *testing.T) {
	diff, err := gbp(t, "12.5").Sub(gbp(t, "0.0125"))
	if err != nil || diff.String() != "12.4875 GBP" {
		t.Errorf("Sub = %q, %v", diff, err)
	}
	if _, err := gbp(t, "-99999999999.9999").Sub(gbp(t, "0.0001")); !errors.Is(err, ErrRange) {
		t.Errorf("Sub past the minimum error = %v, want ErrRange", err)
	}
}

func TestMoneyMismatch(t *testing.T) {
	// Arithmetic and comparison across two currencies have no defined
	// result (ADR-0017), and the pair is what lets the type refuse them
	// (ADR-0032). The mismatch comes before any range check.
	pounds, yen := gbp(t, "99999999999.9999"), mustMoney(t, "0.0001", "JPY")
	if _, err := pounds.Add(yen); !errors.Is(err, ErrMismatch) {
		t.Errorf("Add across currencies error = %v, want ErrMismatch", err)
	}
	if _, err := pounds.Sub(yen); !errors.Is(err, ErrMismatch) {
		t.Errorf("Sub across currencies error = %v, want ErrMismatch", err)
	}
	if _, err := pounds.Compare(yen); !errors.Is(err, ErrMismatch) {
		t.Errorf("Compare across currencies error = %v, want ErrMismatch", err)
	}

	// Inequality is a defined result, so Equal answers rather than fails.
	if gbp(t, "1").Equal(mustMoney(t, "1", "JPY")) {
		t.Error("Equal held across currencies")
	}
}

func TestMoneyNeg(t *testing.T) {
	if got := gbp(t, "12.5").Neg().String(); got != "-12.5000 GBP" {
		t.Errorf("Neg = %q", got)
	}
	// The range is symmetric, so negating either end stays inside it.
	if got := gbp(t, "-99999999999.9999").Neg().String(); got != "99999999999.9999 GBP" {
		t.Errorf("Neg at the minimum = %q", got)
	}
	if got := gbp(t, "0").Neg(); !got.Equal(gbp(t, "0")) {
		t.Errorf("Neg of zero = %q", got)
	}
}

func TestMoneyMul(t *testing.T) {
	// Nothing rounds: the amount is at scale 4 and the quantity is whole.
	// Three lines at 0.0125 total an amount no one can pay (ADR-0017).
	got, err := gbp(t, "0.0125").Mul(3)
	if err != nil || got.String() != "0.0375 GBP" {
		t.Errorf("Mul = %q, %v", got, err)
	}

	// The zero result keeps its currency, because a value with one half is
	// not admitted.
	if got, err := gbp(t, "12.5").Mul(0); err != nil || !got.Equal(gbp(t, "0")) {
		t.Errorf("Mul by zero = %q, %v", got, err)
	}
	if got, err := gbp(t, "12.5").Mul(-2); err != nil || got.String() != "-25.0000 GBP" {
		t.Errorf("Mul by a negative = %q, %v", got, err)
	}
	if got, err := gbp(t, "0").Mul(math.MinInt64); err != nil || !got.Equal(gbp(t, "0")) {
		t.Errorf("zero Mul MinInt64 = %q, %v", got, err)
	}
}

func TestMoneyMulRange(t *testing.T) {
	cases := []struct {
		amount   string
		quantity int64
	}{
		{"0.0001", 1_000_000_000_000_000 + 1},
		{"99999999999.9999", 2},
		{"-99999999999.9999", 2},
		{"0.0001", math.MinInt64},
		{"0.0001", math.MaxInt64},
	}
	for _, c := range cases {
		if _, err := gbp(t, c.amount).Mul(c.quantity); !errors.Is(err, ErrRange) {
			t.Errorf("Mul(%q, %d) error = %v, want ErrRange", c.amount, c.quantity, err)
		}
	}

	// The check is exact, so the largest product that fits still passes.
	got, err := gbp(t, "0.0001").Mul(999_999_999_999_999)
	if err != nil || got.String() != "99999999999.9999 GBP" {
		t.Errorf("Mul at the maximum = %q, %v", got, err)
	}
}

func TestMoneyCompare(t *testing.T) {
	// Compare is the ordering base_price derives from (ADR-0027).
	low, high := gbp(t, "12.4999"), gbp(t, "12.5")
	cases := []struct {
		m, n Money
		want int
	}{
		{low, high, -1},
		{high, low, 1},
		{low, low, 0},
		// Sign does not confuse the ordering.
		{gbp(t, "-0.0001"), gbp(t, "0"), -1},
		{gbp(t, "0"), gbp(t, "-0.0001"), 1},
	}
	for _, c := range cases {
		got, err := c.m.Compare(c.n)
		if err != nil || got != c.want {
			t.Errorf("%q.Compare(%q) = %d, %v, want %d", c.m, c.n, got, err, c.want)
		}
	}

	if !low.Equal(gbp(t, "12.4999")) || low.Equal(high) {
		t.Errorf("Equal compared %q wrongly", low)
	}
}
