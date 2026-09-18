# 0031 - An absent amount is a separate type

Status: Accepted

## Context

`base_price` is the lowest price among a product's SKUs, and a product with no SKUs has none (ADR-0027). The contract omits the field in that state, and the column holds NULL.

An amount is a fixed-scale decimal (ADR-0017). Its zero value is 0.0000, and 0.0000 is a real amount: a free shipping line carries it, and so does a price a seller sets to zero. One amount therefore cannot separate an absent price from a zero one, and the state ADR-0027 calls valid has no representation.

Four ways to carry the absence:

1. A pointer to an amount. The encoder writes null for a nil pointer without the amount type having a say, and every read dereferences. A nil that escapes its check reaches a format verb as a placeholder rather than as an error.

2. A reserved amount that means absent. It has to sit outside NUMERIC(15,4) to avoid colliding with a real price, which breaks the single representation ADR-0017 set, and every arithmetic path then tests for it.

3. A presence flag on the amount itself. Absence spreads to every amount in the build, and each arithmetic, comparison and encoding path pays a check so that one field can be absent.

4. A second type that pairs an amount with a presence flag. The database driver interfaces already read and write that shape. The absent case is named once, in the type, rather than at each call site.

## Decision

Absence lives in a separate type. It pairs an amount with a flag that states whether the amount is there, and only a field the contract marks optional uses it. `base_price` is the only such field in this build.

The amount type holds no absent value. It refuses a SQL NULL and a JSON null, and reports each as its own error rather than reading a zero amount.

## Consequences

- `base_price` carries a type that says it can be absent, and no other amount carries it.
- Arithmetic and comparison stay free of a presence check.
- A NULL read into an amount fails at the read, rather than becoming a zero that surfaces later as a wrong price.
- A caller reads the flag before the amount. Nothing enforces that order, and a caller that skips it reads 0.0000.
- An absent `base_price` omits the field on the wire, because the contract does not mark it nullable. The column holds NULL.
- A second optional amount reuses the type. Nothing in this build needs one.
