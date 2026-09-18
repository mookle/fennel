package money

import "errors"

var (
	// ErrSyntax reports a value that is not in a form this package reads
	ErrSyntax = errors.New("not in a form this package reads")

	// ErrRange reports a value outside NUMERIC(15,4).
	ErrRange = errors.New("outside NUMERIC(15,4)")

	// ErrNull reports a SQL NULL read into a type that holds no absent value.
	ErrNull = errors.New("null into a value that cannot be absent")

	// ErrPrecision reports a driver value that lost precision before this
	// package saw it, so no amount read from it is exact.
	ErrPrecision = errors.New("value lost precision before it was read")
)
