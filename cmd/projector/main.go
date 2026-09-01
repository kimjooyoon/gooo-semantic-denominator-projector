package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-semantic-denominator-projector/internal/projector"
)

func main() {
	if len(os.Args) < 2 {
		fatal("command is required: check, generate, or conformance")
	}
	switch os.Args[1] {
	case "check":
		check(os.Args[2:])
	case "generate":
		generate(os.Args[2:], false)
	case "conformance":
		generate(os.Args[2:], true)
	default:
		fatal("unknown command %q", os.Args[1])
	}
}

type flags struct {
	source, cases, output, root string
}

func parseFlags(command string, args []string, outputRequired bool) flags {
	set := flag.NewFlagSet(command, flag.ExitOnError)
	values := flags{}
	set.StringVar(&values.source, "source", ".gooo/semantic-denominator-projector.gooo", "released .gooo semantic graph")
	set.StringVar(&values.cases, "cases", "fixtures/cases", "canonical case fixture directory")
	set.StringVar(&values.output, "output", "", "absolute empty caller-owned output directory")
	set.StringVar(&values.root, "root", ".", "repository root for inventory")
	set.Parse(args)
	if outputRequired && values.output == "" {
		fatal("%s requires --output", command)
	}
	if command == "generate" && values.output == "" {
		fatal("generate requires --output")
	}
	return values
}

func check(args []string) {
	values := parseFlags("check", args, false)
	ir, _, err := projector.LoadGraph(values.source)
	if err != nil {
		fatal(err.Error())
	}
	cases, err := projector.LoadCases(values.cases, ir.Graph)
	if err != nil {
		fatal(err.Error())
	}
	printJSON(struct {
		Schema     string `json:"schema"`
		Activities int    `json:"activities"`
		Cells      int    `json:"cells"`
		Cases      int    `json:"cases"`
		Artifacts  int    `json:"artifacts"`
	}{ir.Schema, len(ir.Graph.Activities), len(ir.Graph.Cells), len(cases), len(ir.Graph.Artifacts)})
}

func generate(args []string, conformance bool) {
	values := parseFlags("generate", args, conformance)
	result, err := projector.Generate(values.source, values.cases, values.output, values.root)
	if err != nil {
		fatal(err.Error())
	}
	if conformance {
		if err := verifyConformance(values.output, result); err != nil {
			fatal(err.Error())
		}
	}
	printJSON(struct {
		Decision         string `json:"decision"`
		ScenarioDenominator int `json:"scenario_denominator"`
		Closed           int    `json:"closed"`
		Unknown          int    `json:"unknown"`
		Refuted          int    `json:"refuted"`
		ReplayMatch      bool   `json:"replay_match"`
		OutputDirectory  string `json:"output_directory"`
	}{
		Decision: "CONFORMANT",
		ScenarioDenominator: result.Denominator.ScenarioDenominator,
		Closed: result.Denominator.StateCounts.Closed,
		Unknown: result.Denominator.StateCounts.Unknown,
		Refuted: result.Denominator.StateCounts.Refuted,
		ReplayMatch: result.Replay.Match,
		OutputDirectory: filepath.Clean(values.output),
	})
}

func verifyConformance(output string, result projector.GenerationResult) error {
	if result.Denominator.StateCounts != result.Denominator.ExpectedStateCounts {
		return fmt.Errorf("projected state counts do not match graph expectations: got %+v expected %+v", result.Denominator.StateCounts, result.Denominator.ExpectedStateCounts)
	}
	if !result.Replay.Match {
		return fmt.Errorf("order-perturbed replay digest mismatch")
	}
	for _, assertion := range result.Assertions {
		if !assertion.Pass {
			return fmt.Errorf("assertion failed for case %s", assertion.CaseID)
		}
	}
	for _, result := range result.Denominator.Cases {
		if result.Decision == projector.DecisionUnknown && !result.Unknown.Valid() {
			return fmt.Errorf("UNKNOWN case %s does not contain all six fields", result.CaseID)
		}
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		return err
	}
	if len(entries) != len(projector.RequiredArtifactNames) {
		return fmt.Errorf("conformance output has %d files, expected %d", len(entries), len(projector.RequiredArtifactNames))
	}
	return nil
}

func printJSON(value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println(string(raw))
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
