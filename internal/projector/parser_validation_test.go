package projector

import (
	"os"
	"testing"
)

func TestValidateGraphRejectsRuleBoundToMissingCell(t *testing.T) {
	raw, err := os.ReadFile("../../.gooo/semantic-denominator-projector.gooo")
	if err != nil {
		t.Fatal(err)
	}
	graph, err := parseGraph("semantic-denominator-projector.gooo", raw)
	if err != nil {
		t.Fatal(err)
	}
	rule := graph.Rules["duplicate_stable_id"]
	rule.Cell = "missing-cell"
	graph.Rules["duplicate_stable_id"] = rule
	if err := validateGraph(graph); err == nil {
		t.Fatal("expected rule bound to missing cell to be rejected")
	}
}
