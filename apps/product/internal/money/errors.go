package money

import "errors"

var (
	// ErrSyntax reports a value that is not in a form this package reads:
	// the contract's string for an amount or a currency, the contract's
	// object for a pair, or the composite text a read composes for a pair.
	ErrSyntax = errors.New("not in a form this package reads")

	// ErrRange reports a value outside NUMERIC(15,4).
	ErrRange = errors.New("outside NUMERIC(15,4)")

	// ErrCurrency reports a code that does not name an ISO 4217 currency.
	ErrCurrency = errors.New("does not name a currency")

	// ErrMismatch reports arithmetic or comparison across two currencies.
	// Money refuses both operations (ADR-0032).
	ErrMismatch = errors.New("currencies differ")

	// ErrNull reports a SQL NULL read into a type that holds no absent value.
	ErrNull = errors.New("null into a value that cannot be absent")

	// ErrPrecision reports a driver value that lost precision before this
	// package saw it, so no amount read from it is exact.
	ErrPrecision = errors.New("value lost precision before it was read")
)
