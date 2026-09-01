# Semantic denominator projector v1

## Authority

The released `.gooo` graph is the semantic authority. It declares twelve
activities, twelve cells, six generated artifact names, case expectations,
proof choices, indicator classes, precedence, and the fail-closed rules. The
Go runtime may parse and evaluate those declarations, but it does not replace
them with a second hand-written contract.

Every IR object carries a stable semantic ID and source location. The
denominator, distributions, activity mapping, assertions, NDJSON events, and
report are projections of the same IR, so a number cannot drift independently
in a report or README.

## State rules

`REFUTED` means the evidence contradicts the graph contract. Missing,
ambiguous, stale, unbounded, or unavailable evidence is `UNKNOWN` and retains
all six unknown fields. Resolution precedence is exactly `REFUTED > UNKNOWN >
CLOSED`.

The improvement claim is independent of the scenario decision. It remains
`UNKNOWN` until an exact integer before/after pair has matching scenario,
source, contract, fixture, toolchain, and runner identities. External user
utility evidence is `UNKNOWN` when absent.

## Effects and replay

Generation writes only to an empty caller-owned output directory outside the
repository. Normal and order-perturbed replay both normalize the IR by stable
ordinal and ID before hashing; a digest mismatch is `REFUTED` and is preserved
in the replay receipt. No consumer branch or cross-project gate is required.
