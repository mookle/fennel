# 0017 - Money is NUMERIC(15,4), and a fixed-scale decimal string on the wire

Status: Accepted

## Context

Money is stored in PostgreSQL, crosses services in events and via REST responses, and appears in the OpenAPI contracts. All three need a single, consistent representation.

Binary floating point cannot represent most decimal fractions exactly, which often creates rounding errors and precision loss. It is unsuitable for money.

Scaled integers are exact, but require a scale that sits alongside the value; both value and scale must be interpreted to produce the correct monetary value. A separate scale column also allows per-row variable scale, which defeats a plain index on the value, blocks aggregation, and prevents ordering.

A scale based on a currency's decimal precision constrains the unit price to real money. Sellers dealing in low-value, high-volume commodities often carry unit prices less than a currency's minor unit, relying on minimum quantities to bring a line item cost back to something real. 

Any scale must cover the widest minor unit. The range defined in ISO 4217 starts at 0 (e.g. JPY) and extends to 4 (e.g. CLF).

JSON numbers are commonly parsed as IEEE 754 doubles, so the wire representation must avoid JSON numeric types.

## Decision

- Store money as fixed-scale decimals (`NUMERIC(15,4)`) alongside an ISO-4217 currency code.
- Carry it on the wire as a decimal string, never a JSON number. A response emits exactly four decimal places, and an input may carry zero to four.
- Calculate in exact decimal types, never binary floating point.
- Arithmetic between two currencies has no defined result.

## Consequences

- Storage, the wire and memory share one representation of money.
- The OpenAPI contract holds the canonical wire form as a single pattern, and the contract-derived doubles inherit it (ADR-0015).
- The wire form is generic. A yen amount reads `"1000.0000"`. The alternative of padding the currency's own minor unit would round away any price quoted finer than it, so a response never does.
- Four places exceeds the minor unit of most currencies, so a line can total an amount no one can pay (e.g. 3 x 0.0125 GBP = 0.0375).
- Display and settlement both require ISO-4217 metadata (currency symbols, minor-unit precision, etc).
- Multi-currency orders stay undefined and need their own decision.
