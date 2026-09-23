package projector

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestRuleIDsCannotAliasOtherSemanticNodes(t *testing.T) {
	path := filepath.Join("..", "..", ".gooo", "semantic-denominator-projector.gooo")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Replace(raw,
		[]byte("rule id=improvement_pair_absent"),
		[]byte("rule id=gooo.activity.semantic-denominator-projector.bind-semantic-evidence"),
		1,
	)
	if bytes.Equal(raw, mutated) {
		t.Fatal("test mutation did not find the rule declaration")
	}
	if _, err := parseGraph("test.gooo", mutated); err == nil {
		t.Fatal("cross-kind stable ID collision was accepted")
	}
}
