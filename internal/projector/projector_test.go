package projector

import "testing"

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
