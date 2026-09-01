package projector

import (
	"fmt"
	"strings"
)

func renderReport(ir SemanticIR, results []CaseResult, replay ReplayReceipt) string {
	var builder strings.Builder
	counts := stateCounts(results)
	expected := expectedStateCounts(ir.Graph.Cases)
	builder.WriteString("# Gooo semantic denominator projector report\n\n")
	fmt.Fprintf(&builder, "Source graph: `%s`\n\n", ir.Graph.GraphID)
	fmt.Fprintf(&builder, "Released graph digest: `%s`\n\n", ir.SourceDigest)
	fmt.Fprintf(&builder, "The semantic graph is the authority. Go acts as parser, evaluator, generator, and runtime. Repository writes: `%d`; cross-project required gates: `%d`.\n\n", ir.Graph.RepositoryWrites, ir.Graph.ExternalRequiredGates)
	builder.WriteString("## Fixed denominator\n\n")
	fmt.Fprintf(&builder, "Scenario denominator: **%d**. Expected state counts are `CLOSED=%d`, `UNKNOWN=%d`, `REFUTED=%d`; projected counts are `CLOSED=%d`, `UNKNOWN=%d`, `REFUTED=%d`. Precedence is `%s`.\n\n", len(ir.Graph.Cases), expected.Closed, expected.Unknown, expected.Refuted, counts.Closed, counts.Unknown, counts.Refuted, strings.Join(ir.Graph.Precedence, " > "))
	builder.WriteString("| proof choice | total |\n|---|---:|\n")
	for _, value := range graphLabelCounts(ir.Graph, "proof") {
		fmt.Fprintf(&builder, "| %s | %d |\n", value.Label, value.Total)
	}
	builder.WriteString("\n| indicator class | total |\n|---|---:|\n")
	for _, value := range graphLabelCounts(ir.Graph, "indicator") {
		fmt.Fprintf(&builder, "| %s | %d |\n", value.Label, value.Total)
	}
	builder.WriteString("\nProof and indicator distributions are direct counts of graph declarations; no score or percentage is used.\n\n")
	builder.WriteString("## Scenario assertions\n\n")
	builder.WriteString("| ordinal | case | expected | projected | assertion | reason |\n|---:|---|---|---|---|---|\n")
	for _, result := range results {
		fmt.Fprintf(&builder, "| %d | `%s` | `%s` | `%s` | %t | `%s` |\n", result.Ordinal, result.CaseID, result.Expected, result.Decision, result.Expected == result.Decision, result.Reason)
	}
	builder.WriteString("\n")
	builder.WriteString("## UNKNOWN and REFUTED coverage\n\n")
	builder.WriteString("Every UNKNOWN projection below is generated with all six fields: `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`. REFUTED takes precedence over UNKNOWN, which takes precedence over CLOSED.\n\n")
	for _, result := range results {
		if result.Decision == DecisionUnknown && result.Unknown != nil {
			fmt.Fprintf(&builder, "- UNKNOWN `%s`: stage `%s`, step `%s`, reason `%s`, class `%s`, next `%s`, blocked by `%s`.\n", result.CaseID, result.Unknown.Stage, result.Unknown.Step, result.Unknown.Reason, result.Unknown.UnknownClass, result.Unknown.NextOperation, strings.Join(result.Unknown.BlockedBy, ","))
		}
		if result.Decision == DecisionRefuted {
			fmt.Fprintf(&builder, "- REFUTED `%s`: `%s`.\n", result.CaseID, result.Reason)
		}
	}
	builder.WriteString("\n")
	builder.WriteString("## Activity mapping\n\n")
	builder.WriteString("| cell | activity stable ID | activity | proof | indicator | artifact |\n|---|---|---|---|---|---|\n")
	for _, cell := range ir.Graph.Cells {
		for _, activity := range ir.Graph.Activities {
			if activity.Name == cell.Activity {
				fmt.Fprintf(&builder, "| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` |\n", cell.StableID, activity.StableID, activity.Name, cell.Proof, cell.Indicator, activity.Artifact)
				break
			}
		}
	}
	builder.WriteString("\n")
	builder.WriteString("## Replay and improvement\n\n")
	fmt.Fprintf(&builder, "Normal replay digest: `%s`\n\nOrder-perturbed replay digest: `%s`\n\nReplay state: `%s` (`%s`).\n\n", replay.NormalDigest, replay.OrderPerturbedDigest, replay.State, replay.Reason)
	fmt.Fprintf(&builder, "Improvement state: `%s` because an exact before/after integer pair must match scenario, source, contract, fixture, toolchain, and runner. External user utility is UNKNOWN when no evidence is supplied.\n\n", summarizeImprovement(results).State)
	builder.WriteString("## Generated artifacts\n\n")
	for _, name := range RequiredArtifactNames {
		fmt.Fprintf(&builder, "- `%s`\n", name)
	}
	builder.WriteString("\nROOT `README.md` is excluded from inventory. All generated files are written only to the caller-owned output directory.\n")
	return builder.String()
}
