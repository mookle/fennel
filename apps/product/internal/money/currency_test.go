package money

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/moov-io/iso4217"
)

// pinnedTable is every alphabetic code github.com/moov-io/iso4217 v0.4.0
// carries, in order. notCurrency classifies this exact set, so a dependency
// version bump that adds, drops or renames a code has to be read by a person
// before it reaches the domain.
const pinnedTable = `
AED AFN ALL AMD ANG AOA ARS AUD AWG AZN BAM BBD BDT BGN BHD BIF BMD BND BOB
BOV BRL BSD BTN BWP BYN BZD CAD CDF CHE CHF CHW CLF CLP CNH CNY COP COU CRC
CUC CUP CVE CZK DJF DKK DOP DZD EGP ERN ETB EUR FJD FKP GBP GEL GHS GIP GMD
GNF GTQ GYD HKD HNL HRK HTG HUF IDR ILS INR IQD IRR ISK JMD JOD JPY KES KGS
KHR KMF KPW KRW KWD KYD KZT LAK LBP LKR LRD LSL LYD MAD MDL MGA MKD MMK MNT
MOP MRU MUR MVR MWK MXN MXV MYR MZN NAD NGN NIO NOK NPR NZD OMR PAB PEN PGK
PHP PKR PLN PYG QAR RON RSD RUB RWF SAR SBD SCR SDG SEK SGD SHP SLE SLL SOS
SRD SSP STN SVC SYP SZL THB TJS TMT TND TOP TRY TTD TWD TZS UAH UGX USD USN
UYI UYU UYW UZS VED VES VND VUV WST XAD XAF XAG XAU XBA XBB XBC XBD XCD XCG
XDR XOF XPD XPF XPT XSU XTS XUA XXX YER ZAR ZMW ZWG ZWL
`

// tableCodes reads the dependency's table. The package exports no iterator
// over it, so this asks it about every three-letter code there is. Lookup
// matches numeric codes too, and this range holds none of them.
func tableCodes() []string {
	var codes []string
	var b [3]byte
	for i := range 26 {
		b[0] = byte('A' + i)
		for j := range 26 {
			b[1] = byte('A' + j)
			for k := range 26 {
				b[2] = byte('A' + k)
				if _, ok := iso4217.Lookup(string(b[:])); ok {
					codes = append(codes, string(b[:]))
				}
			}
		}
	}
	return codes
}

func TestTableIsPinned(t *testing.T) {
	got, want := tableCodes(), strings.Fields(pinnedTable)
	if slices.Equal(got, want) {
		return
	}

	for _, code := range got {
		if !slices.Contains(want, code) {
			entry, _ := iso4217.Lookup(code)
			t.Errorf("the dependency added %q (%s). Classify it, then add it to pinnedTable, and add it to notCurrency if it names no currency", code, entry.Name)
		}
	}
	for _, code := range want {
		if !slices.Contains(got, code) {
			t.Errorf("the dependency dropped %q. Remove it from pinnedTable, and from notCurrency if it is there", code)
		}
	}
}

func TestNotCurrencyHoldsNoGhosts(t *testing.T) {
	// A code the dependency does not carry is already rejected, so its entry
	// here does nothing and hides the fact that the guard has moved.
	for code := range notCurrency {
		if _, ok := iso4217.Lookup(code); !ok {
			t.Errorf("notCurrency holds %q, which the dependency does not carry", code)
		}
	}
}

func TestEveryPinnedCodeIsClassified(t *testing.T) {
	// Each code in the table either names a currency and parses, or sits in
	// notCurrency and does not. There is no third state.
	for _, code := range strings.Fields(pinnedTable) {
		_, excluded := notCurrency[code]
		_, err := ParseCurrency(code)
		switch {
		case excluded && err == nil:
			t.Errorf("ParseCurrency(%q) accepted an excluded code", code)
		case !excluded && err != nil:
			t.Errorf("ParseCurrency(%q) rejected a code no exclusion covers: %v", code, err)
		}
	}
}

func TestParseCurrencyRejectsNonCurrencies(t *testing.T) {
	// The cases ADR-0030 names, spelled out so the rule does not rest on the
	// table pin alone.
	cases := []string{
		"XAU", "XAG", "XPT", "XPD", // precious metals
		"XBA", "XBB", "XBC", "XBD", // bond market units
		"XDR", "XSU", "XUA", "XAD", // units of account
		"XXX", "XTS", // no currency, and testing
		"CLF", "UYI", "UYW", "USN", "BOV", "COU", "MXV", // funds and index units
		"CHE", "CHW", // WIR Bank complementary currencies
		"CNH",        // an offshore market convention
		"ZZZ", "AAA", // the shape the contract's pattern admits and nothing else
	}
	for _, code := range cases {
		if _, err := ParseCurrency(code); !errors.Is(err, ErrCurrency) {
			t.Errorf("ParseCurrency(%q) error = %v, want ErrCurrency", code, err)
		}
	}
}

func TestParseCurrencyAcceptsCurrencies(t *testing.T) {
	// HRK is withdrawn and still names a currency, so a product listed in it
	// stays valid (ADR-0030). XAF and XCD carry the X prefix and are legal
	// tender.
	for _, code := range []string{"GBP", "EUR", "USD", "JPY", "HRK", "XAF", "XCD", "XCG", "XOF", "XPF"} {
		got, err := ParseCurrency(code)
		if err != nil {
			t.Errorf("ParseCurrency(%q): %v", code, err)
			continue
		}
		if got.String() != code {
			t.Errorf("ParseCurrency(%q).String() = %q", code, got)
		}
	}
}

func TestParseCurrencyIsCaseSensitive(t *testing.T) {
	// The dependency folds case and pads a short code, and the contract's
	// pattern admits neither. The identity check is what keeps the type as
	// strict as the contract.
	for _, code := range []string{"usd", "Usd", "uSD", " USD", "USD ", "", "US", "USDD"} {
		if _, err := ParseCurrency(code); !errors.Is(err, ErrCurrency) {
			t.Errorf("ParseCurrency(%q) error = %v, want ErrCurrency", code, err)
		}
	}
}

func TestParseCurrencyRejectsNumericCodes(t *testing.T) {
	// The dependency looks up a numeric code as readily as an alphabetic one,
	// and the contract carries only the alphabetic form.
	for _, code := range []string{"784", "826", "978", "999"} {
		if _, err := ParseCurrency(code); !errors.Is(err, ErrCurrency) {
			t.Errorf("ParseCurrency(%q) error = %v, want ErrCurrency", code, err)
		}
	}
}

func TestCurrencyRefusesTheZeroValue(t *testing.T) {
	// No currency field in the contract is optional, so an empty code is a
	// bug rather than an absent value.
	var c Currency
	if _, err := c.MarshalJSON(); !errors.Is(err, ErrCurrency) {
		t.Errorf("MarshalJSON of the zero value error = %v, want ErrCurrency", err)
	}
	if _, err := c.Value(); !errors.Is(err, ErrCurrency) {
		t.Errorf("Value of the zero value error = %v, want ErrCurrency", err)
	}
}

func TestCurrencyJSONRoundTrip(t *testing.T) {
	c, err := ParseCurrency("GBP")
	if err != nil {
		t.Fatalf("ParseCurrency: %v", err)
	}

	b, err := json.Marshal(c)
	if err != nil || string(b) != `"GBP"` {
		t.Fatalf("Marshal = %s, %v", b, err)
	}

	var back Currency
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if back != c {
		t.Errorf("round trip gave %q, want %q", back, c)
	}
}

func TestCurrencyUnmarshalJSONRejects(t *testing.T) {
	cases := []struct {
		in   string
		want error
	}{
		{"null", ErrNull},
		{"123", ErrSyntax},
		{"[]", ErrSyntax},
		{"`GBP`", ErrSyntax}, // a Go raw string, which JSON never produces
		{`"ZZZ"`, ErrCurrency},
		{`"gbp"`, ErrCurrency},
	}
	for _, c := range cases {
		var got Currency
		if err := got.UnmarshalJSON([]byte(c.in)); !errors.Is(err, c.want) {
			t.Errorf("UnmarshalJSON(%s) error = %v, want %v", c.in, err, c.want)
		}
	}
}

func TestCurrencyScan(t *testing.T) {
	var c Currency
	if err := c.Scan("GBP"); err != nil || c.String() != "GBP" {
		t.Errorf("Scan(string) = %q, %v", c, err)
	}
	if err := c.Scan([]byte("EUR")); err != nil || c.String() != "EUR" {
		t.Errorf("Scan([]byte) = %q, %v", c, err)
	}

	// No currency column in this build is nullable.
	if err := c.Scan(nil); !errors.Is(err, ErrNull) {
		t.Errorf("Scan(nil) error = %v, want ErrNull", err)
	}
	if err := c.Scan(int64(826)); !errors.Is(err, ErrSyntax) {
		t.Errorf("Scan(int64) error = %v, want ErrSyntax", err)
	}
	if err := c.Scan("ZZZ"); !errors.Is(err, ErrCurrency) {
		t.Errorf("Scan(%q) error = %v, want ErrCurrency", "ZZZ", err)
	}
}

func TestCurrencyValue(t *testing.T) {
	c, err := ParseCurrency("GBP")
	if err != nil {
		t.Fatalf("ParseCurrency: %v", err)
	}
	v, err := c.Value()
	if err != nil || v != "GBP" {
		t.Errorf("Value() = %#v, %v", v, err)
	}
}
