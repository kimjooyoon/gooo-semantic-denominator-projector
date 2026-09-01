# Gooo semantic denominator projector

This repository is a read-only Gooo semantic graph projector. The released
graph in `.gooo/semantic-denominator-projector.gooo` is the only authority for
the denominator, semantic state vocabulary, precedence, proof choices,
indicator classes, activity bindings, and output contract. Go is only the
parser, evaluator, generator, and runtime for that graph.

The projector builds a source-located intermediate representation with stable
semantic IDs, then derives all six caller-owned output artifacts from that IR:

- `semantic-denominator.json`
- `semantic-distribution.json`
- `generated-assertions.json`
- `projection-events.ndjson`
- `replay-receipt.json`
- `report.md`

The canonical denominator contains exactly twelve cases: four `CLOSED`, four
`UNKNOWN`, and four `REFUTED`. `REFUTED > UNKNOWN > CLOSED` is fail-closed. An
`UNKNOWN` record always contains `stage`, `step`, `reason`, `unknown_class`,
`next_operation`, and `blocked_by`. Proof and indicator distributions are
counted by their declared graph labels; no score is used.

The runtime requires an absolute, empty, caller-owned output directory outside
the repository. It never writes the source repository. The repository has no
cross-project required gates (`0`). ROOT `README.md` is excluded from the
inventory.

Generate into a temporary directory with:

```text
go run ./cmd/projector generate \
  --source .gooo/semantic-denominator-projector.gooo \
  --cases fixtures/cases \
  --output /absolute/path/to/empty/caller-output
```

Local validation commands are intentionally delegated to GitHub Actions. The
workflow records exact integer wall time and peak RSS for compile, build, test,
conformance, and integration, plus test accounting and source inventory.
