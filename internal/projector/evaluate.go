package projector

import (
	"sort"
)

func EvaluateCases(ir SemanticIR, inputs []CaseInput) []CaseResult {
	results := make([]CaseResult, 0, len(inputs))
	for _, input := range inputs {
		results = append(results, evaluateCase(ir, input))
	}
	sort.Slice(results, func(left, right int) bool {
		return results[left].Ordinal < results[right].Ordinal
	})
	return results
}

func evaluateCase(ir SemanticIR, input CaseInput) CaseResult {
	contract := caseContract(ir.Graph, input.CaseID)
	mapping := graphActivityMapping(ir.Graph)
	result := CaseResult{
		Ordinal:         contract.Ordinal,
		CaseID:          input.CaseID,
		Expected:        contract.Expected,
		Decision:        DecisionClosed,
		Reason:          "SEMANTIC_GRAPH_EVIDENCE_CLOSED",
		Issues:          []Issue{},
		StableIDs:       append([]SemanticNodeInput(nil), input.SemanticNodes...),
		ActivityMapping: mapping,
		FixturePath:     input.FixturePath,
	}
	addIssue := func(ruleID string, blockedBy []string) {
		for _, issue := range result.Issues {
			if issue.RuleID == ruleID {
				return
			}
		}
		rule := ir.Graph.Rules[ruleID]
		result.Issues = append(result.Issues, Issue{RuleID: ruleID, State: rule.State, Reason: rule.Reason, BlockedBy: sortedCopy(blockedBy)})
	}

	if input.ExpectedDecision != "" && input.ExpectedDecision != contract.Expected {
		addIssue("contradictory_state", []string{"expected_decision"})
	}
	if input.ObservedDenominator == nil {
		addIssue("missing_activity", []string{"observed_denominator"})
	} else if *input.ObservedDenominator != len(ir.Graph.Cells) {
		addIssue("stale_expected_denominator", []string{"observed_denominator", "DENOMINATOR_ESTABLISHED"})
	}
	missingActivities := missingActivityNames(ir.Graph, input.ObservedActivities)
	if len(missingActivities) > 0 {
		addIssue("missing_activity", missingActivities)
	}
	if duplicateIDs(input.SemanticNodes) {
		addIssue("duplicate_stable_id", duplicateIDList(input.SemanticNodes))
	}
	if input.ContradictoryState {
		addIssue("contradictory_state", []string{"top_level_decision", "claim.state"})
	}
	if input.UnboundedInput {
		addIssue("unbounded_input", []string{"input_scope"})
	}
	if input.ExternalUserUtility == nil || !*input.ExternalUserUtility {
		addIssue("external_user_utility_absent", []string{"external_user_utility_evidence"})
	}
	if input.Improvement == nil || input.Improvement.Before == nil || input.Improvement.After == nil {
		addIssue("improvement_pair_absent", []string{"improvement.before", "improvement.after"})
	}
	switch input.TopLevelDecision {
	case DecisionClosed, DecisionUnknown, DecisionRefuted:
		if input.TopLevelDecision != contract.Expected {
			addIssue("contradictory_state", []string{"top_level_decision", "graph_case_expectation"})
		}
	case "":
		addIssue("unknown_top_level_decision", []string{"top_level_decision"})
	default:
		addIssue("unknown_top_level_decision", []string{"top_level_decision"})
	}
	if input.TopLevelDecision == DecisionUnknown && !input.Claim.Valid() {
		addIssue("malformed_unknown", []string{"claim"})
	}
	if input.MalformedUnknown {
		addIssue("malformed_unknown", []string{"claim"})
	}

	sort.Slice(result.Issues, func(left, right int) bool {
		return result.Issues[left].RuleID < result.Issues[right].RuleID
	})
	if len(result.Issues) > 0 {
		winner := result.Issues[0]
		for _, issue := range result.Issues[1:] {
			if precedenceIndex(ir.Graph.Precedence, issue.State) < precedenceIndex(ir.Graph.Precedence, winner.State) {
				winner = issue
			}
		}
		result.Decision = winner.State
		result.Reason = winner.Reason
		if result.Decision == DecisionUnknown {
			result.Unknown = unknownForRule(ir.Graph, winner.RuleID, winner.BlockedBy)
		}
	}
	if result.Decision != contract.Expected {
		addIssue("contradictory_state", []string{"projected_decision", "graph_case_expectation"})
		result.Decision = DecisionRefuted
		result.Reason = ir.Graph.Rules["contradictory_state"].Reason
		result.Unknown = nil
	}
	result.Improvement = evaluateImprovement(ir, input, result.Decision)
	result.Utility = evaluateUtility(ir, input)
	return result
}

func evaluateImprovement(ir SemanticIR, input CaseInput, decision string) ImprovementResult {
	if input.Improvement == nil {
		return unknownImprovement(ir, nil, nil)
	}
	value := input.Improvement
	result := ImprovementResult{State: DecisionUnknown, Before: value.Before, After: value.After}
	if value.Before == nil || value.After == nil || decision != DecisionClosed {
		unknown := unknownForRule(ir.Graph, "improvement_pair_absent", []string{"improvement.before", "improvement.after"})
		result.Unknown = unknown
		return result
	}
	if value.Scenario != input.CaseID || value.SourceDigest != ir.SourceDigest || value.ContractDigest != ir.SourceDigest ||
		value.Fixture != input.FixturePath || value.Toolchain != ToolchainVersion || value.Runner != "github-actions/ubuntu-latest" {
		unknown := unknownForRule(ir.Graph, "improvement_pair_absent", []string{"scenario", "source_digest", "contract_digest", "fixture", "toolchain", "runner"})
		result.Unknown = unknown
		return result
	}
	result.State = DecisionClosed
	return result
}

func unknownImprovement(ir SemanticIR, before, after *int) ImprovementResult {
	return ImprovementResult{
		State: DecisionUnknown,
		Before: before,
		After: after,
		Unknown: unknownForRule(ir.Graph, "improvement_pair_absent", []string{"improvement.before", "improvement.after"}),
	}
}

func evaluateUtility(ir SemanticIR, input CaseInput) UtilityResult {
	if input.ExternalUserUtility != nil && *input.ExternalUserUtility {
		return UtilityResult{State: DecisionClosed}
	}
	return UtilityResult{
		State: DecisionUnknown,
		Unknown: unknownForRule(ir.Graph, "external_user_utility_absent", []string{"external_user_utility_evidence"}),
	}
}

func unknownForRule(graph SemanticGraph, ruleID string, blockedBy []string) *UnknownClaim {
	rule := graph.Rules[ruleID]
	cell := cellByID(graph, rule.Cell)
	return &UnknownClaim{
		Stage:         cell.Stage,
		Step:          cell.Step,
		Reason:        rule.Reason,
		UnknownClass:  rule.UnknownClass,
		NextOperation: rule.NextOperation,
		BlockedBy:     sortedCopy(blockedBy),
	}
}

func graphActivityMapping(graph SemanticGraph) []ActivityProjection {
	activities := map[string]ActivityDecl{}
	for _, activity := range graph.Activities {
		activities[activity.Name] = activity
	}
	mapping := make([]ActivityProjection, 0, len(graph.Cells))
	for _, cell := range graph.Cells {
		activity := activities[cell.Activity]
		mapping = append(mapping, ActivityProjection{
			CellOrdinal: cell.Ordinal, CellID: cell.StableID, ActivityStableID: activity.StableID,
			Activity: activity.Name, Artifact: activity.Artifact, Proof: cell.Proof, Indicator: cell.Indicator, Source: activity.Source,
		})
	}
	return mapping
}

func missingActivityNames(graph SemanticGraph, observed []string) []string {
	observedSet := map[string]bool{}
	for _, name := range observed {
		observedSet[name] = true
	}
	missing := []string{}
	for _, activity := range graph.Activities {
		if !observedSet[activity.Name] {
			missing = append(missing, activity.Name)
		}
	}
	return missing
}

func duplicateIDs(nodes []SemanticNodeInput) bool {
	seen := map[string]bool{}
	for _, node := range nodes {
		if node.StableID == "" {
			continue
		}
		if seen[node.StableID] {
			return true
		}
		seen[node.StableID] = true
	}
	return false
}

func duplicateIDList(nodes []SemanticNodeInput) []string {
	counts := map[string]int{}
	for _, node := range nodes {
		if node.StableID != "" {
			counts[node.StableID]++
		}
	}
	duplicates := []string{}
	for id, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, id)
		}
	}
	sort.Strings(duplicates)
	return duplicates
}

func precedenceIndex(precedence []string, state string) int {
	for index, value := range precedence {
		if value == state {
			return index
		}
	}
	return len(precedence) + 1
}

func sortedCopy(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func caseContract(graph SemanticGraph, id string) CaseContract {
	for _, value := range graph.Cases {
		if value.StableID == id {
			return value
		}
	}
	return CaseContract{}
}

func cellByID(graph SemanticGraph, id string) CellDecl {
	for _, cell := range graph.Cells {
		if cell.StableID == id {
			return cell
		}
	}
	return CellDecl{}
}
