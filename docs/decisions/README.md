# Architecture Decision Records

The big decisions for Fennel. The format is lightweight MADR: Status, Context, Decision, Consequences.

A superseded ADR stays in the tree. It records the reasoning and the rejected alternatives, and both stay useful when someone picks up the deferred work. The number is the stable reference. A cross-reference uses `ADR-NNNN`, never a filename. A slug tracks the current title, so a retitled ADR gets a new filename and a new link in this table.

**An ADR records one decision at one moment, and later work never edits it.** A cross-reference points backwards, to a record that already existed. The one forward pointer is the Status line, which names whatever superseded or amended this decision, and this table repeats it. Nothing else in the body points forward.

**A fact belongs in an ADR only if the decision rests on it.** Cite the record that owns the fact and move on. Never restate another ADR's rule, because two copies of one rule drift apart. Never report what later happened to this decision, because that is the job of the record that changed it.

When a decision changes, write a new ADR. Amend the Status line of the old one, and leave its body alone.

| ADR | Title | Status |
|---|---|---|
| [0001](0001-partial-scope-three-domains.md) | Partial scope: three domains across three deployables | Accepted |
