package projector

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUnknownClaimValidation(t *testing.T) {
	valid := &UnknownClaim{Stage: "STAGE", Step: "STEP", Reason: "REASON", UnknownClass: "CLASS", NextOperation: "NEXT", BlockedBy: []string{}}
	if !valid.Valid() {
		t.Fatal("complete UNKNOWN claim was rejected")
	}
	invalid := &UnknownClaim{Stage: "STAGE", Step: "STEP", Reason: "REASON", UnknownClass: "CLASS", NextOperation: "NEXT"}
	if invalid.Valid() {
		t.Fatal("malformed UNKNOWN claim was accepted")
	}
}

func TestPhysicalLines(t *testing.T) {
	if got := physicalLines([]byte("one\ntwo\n")); got != 2 {
		t.Fatalf("got %d lines, want 2", got)
	}
	if got := physicalLines([]byte("one\ntwo")); got != 2 {
		t.Fatalf("got %d lines, want 2", got)
	}
}

func TestLoadCasesRejectsTrailingJSONValue(t *testing.T) {
	graphPath := filepath.Join("..", "..", ".gooo", "semantic-denominator-projector.gooo")
	ir, _, err := LoadGraph(graphPath)
	if err != nil {
		t.Fatal(err)
	}
	sourceDir := filepath.Join("..", "..", "fixtures", "cases")
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sourceDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if entry.Name() == "01-closed-source-grounding.json" {
			data = append(data, []byte("\n{}\n")...)
		}
		if err := os.WriteFile(filepath.Join(tempDir, entry.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadCases(tempDir, ir.Graph); err == nil {
		t.Fatal("LoadCases accepted a fixture with a trailing JSON value")
	}
}

func TestBuildEventsIsIndependentOfCaseInputOrder(t *testing.T) {
	first := CaseResult{CaseID: "case-b", ActivityMapping: []ActivityProjection{{CellID: "cell-b", ActivityStableID: "activity-b"}}}
	second := CaseResult{CaseID: "case-a", ActivityMapping: []ActivityProjection{{CellID: "cell-a", ActivityStableID: "activity-a"}}}

	forward := buildEvents([]CaseResult{first, second})
	reversed := buildEvents([]CaseResult{second, first})
	if !reflect.DeepEqual(forward, reversed) {
		t.Fatalf("event order depends on case input order: forward=%v reversed=%v", forward, reversed)
	}
	if len(forward) != 2 || forward[0].CaseID != "case-a" || forward[1].CaseID != "case-b" {
		t.Fatalf("events were not canonically ordered: %v", forward)
	}
}
