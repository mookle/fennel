# 0030 - Currency must hold an ISO 4217 currency code

Status: Accepted

## Context

The API schema validates `Currency` as three uppercase letters at the service boundary. `ZZZ` satisfies that shape but identifies nothing.

ISO 4217 includes codes for currencies, funds, precious metals, and units of account. It also reserves `XXX` for a transaction with no currency and `XTS` for testing. Only a currency should price a product.

A withdrawn currency is still a currency. `HRK` named a currency until Croatia adopted the euro, and a product listed in `HRK`, along with every `OrderSku` that copied it (ADR-0004), cannot be invalidated after the fact. So the validation rule is about what a code identifies, not whether or not the thing it identifies is still active.

Manually maintaining a list of currency codes is possible, but would require monitoring the ISO 4217 register directly to ensure no drift occurred. Several libraries already exist that perform exactly this work.

## Decision

A `Currency` code must exist in the ISO 4217 register and identify a currency. Codes that relate to funds, precious metals, units of account, and `XXX` and `XTS` are not currencies and must be excluded. This must be enforced as a type invariant, on read and write.

Downstream services `cart` and `order` do not validate currencies. They receive a code `product` has accepted.

`product` takes the register from an external dependency.

## Consequences

- `ZZZ` cannot enter the domain.
- Funds, precious metals, units of account, and `XXX` and `XTS` cannot price a product.
- A seller can list in a withdrawn currency. Not ideal, but nothing in this build takes an action dependent on the currency code (ADR-0016).
- `POST /v1/products` rejects a non currency with a 400. The contract keeps its validation pattern as-is and states the rule in prose.
- The exclusion list is stable, because a metal or a fund cannot be withdrawn.
- Keeping codes up-to-date will require monitoring the dependency for version bumps.
- The currency code list will only be as up-to-date as the dependency allows.
- Currency symbols and minor units stay out of scope (ADR-0017).
