package projector

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func LoadGraph(path string) (SemanticIR, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return SemanticIR{}, nil, err
	}
	graph, err := parseGraph(path, raw)
	if err != nil {
		return SemanticIR{}, nil, err
	}
	ir := SemanticIR{
		Schema:       IRSchema,
		SourcePath:   filepath.ToSlash(path),
		SourceDigest: DigestBytes(raw),
		Graph:        graph,
	}
	return ir, raw, nil
}

func parseGraph(path string, raw []byte) (SemanticGraph, error) {
	graph := SemanticGraph{
		Schema:     GraphSchema,
		Artifacts:  []ArtifactDecl{},
		Activities: []ActivityDecl{},
		Cells:      []CellDecl{},
		Rules:      map[string]RuleDecl{},
		Cases:      []CaseContract{},
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	lineNumber := 0
	seenTop := map[string]bool{}
	seenStableIDs := map[string]string{}
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "package" || fields[0] == "namespace" {
			if len(fields) != 2 || fields[1] == "" {
				return SemanticGraph{}, fmt.Errorf("line %d: invalid %s declaration", lineNumber, fields[0])
			}
			continue
		}
		values, err := keyValues(fields[1:])
		if err != nil {
			return SemanticGraph{}, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		location := SourceLocation{Path: filepath.ToSlash(path), Line: lineNumber, Column: 1}
		switch fields[0] {
		case "graph":
			if seenTop["graph"] || values["id"] == "" || values["release"] == "" {
				return SemanticGraph{}, fmt.Errorf("line %d: invalid graph declaration", lineNumber)
			}
			precedence := splitList(values["precedence"])
			gates, err := integerValue(values, "external_required_gates")
			if err != nil {
				return fmt.Errorf("line %d: %w", lineNumber, err)
			}
			writes, err := integerValue(values, "repository_writes")
			if err != nil {
				return fmt.Errorf("line %d: %w", lineNumber, err)
			}
			graph.GraphID = values["id"]
			graph.Release = values["release"]
			graph.Precedence = precedence
			graph.ExternalRequiredGates = gates
			graph.RepositoryWrites = writes
			seenTop["graph"] = true
		case "artifact":
			ordinal, err := integerValue(values, "ordinal")
			if err != nil || ordinal < 1 {
				return SemanticGraph{}, fmt.Errorf("line %d: invalid artifact ordinal", lineNumber)
			}
			name := values["name"]
			if name == "" {
				return SemanticGraph{}, fmt.Errorf("line %d: artifact name is required", lineNumber)
			}
			graph.Artifacts = append(graph.Artifacts, ArtifactDecl{Ordinal: ordinal, Name: name, Source: location})
		case "activity":
			activity, err := parseActivity(values, location, lineNumber)
			if err != nil {
				return SemanticGraph{}, err
			}
			if err := registerStableID(seenStableIDs, activity.StableID, "activity", lineNumber); err != nil {
				return SemanticGraph{}, err
			}
			graph.Activities = append(graph.Activities, activity)
		case "cell":
			cell, err := parseCell(values, location, lineNumber)
			if err != nil {
				return SemanticGraph{}, err
			}
			if err := registerStableID(seenStableIDs, cell.StableID, "cell", lineNumber); err != nil {
				return SemanticGraph{}, err
			}
			graph.Cells = append(graph.Cells, cell)
		case "rule":
			rule, err := parseRule(values, location, lineNumber)
			if err != nil {
				return SemanticGraph{}, err
			}
			if _, exists := graph.Rules[rule.StableID]; exists {
				return SemanticGraph{}, fmt.Errorf("line %d: duplicate rule %s", lineNumber, rule.StableID)
			}
			graph.Rules[rule.StableID] = rule
		case "case":
			caseContract, err := parseCase(values, location, lineNumber)
			if err != nil {
				return SemanticGraph{}, err
			}
			if err := registerStableID(seenStableIDs, caseContract.StableID, "case", lineNumber); err != nil {
				return SemanticGraph{}, err
			}
			graph.Cases = append(graph.Cases, caseContract)
		default:
			return SemanticGraph{}, fmt.Errorf("line %d: unsupported declaration %s", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return SemanticGraph{}, err
	}
	if err := validateGraph(graph); err != nil {
		return SemanticGraph{}, err
	}
	return graph, nil
}

func parseActivity(values map[string]string, location SourceLocation, line int) (ActivityDecl, error) {
	ordinal, err := integerValue(values, "ordinal")
	if err != nil || ordinal < 1 {
		return ActivityDecl{}, fmt.Errorf("line %d: invalid activity ordinal", line)
	}
	activity := ActivityDecl{
		Ordinal: ordinal, StableID: values["id"], Name: values["name"], Proof: values["proof"],
		Indicator: values["indicator"], Artifact: values["artifact"], Authority: values["authority"], Source: location,
	}
	if activity.StableID == "" || activity.Name == "" || activity.Proof == "" || activity.Indicator == "" || activity.Artifact == "" || activity.Authority == "" {
		return ActivityDecl{}, fmt.Errorf("line %d: incomplete activity", line)
	}
	return activity, nil
}

func parseCell(values map[string]string, location SourceLocation, line int) (CellDecl, error) {
	ordinal, err := integerValue(values, "ordinal")
	if err != nil || ordinal < 1 {
		return CellDecl{}, fmt.Errorf("line %d: invalid cell ordinal", line)
	}
	cell := CellDecl{
		Ordinal: ordinal, StableID: values["id"], Activity: values["activity"], Stage: values["stage"], Step: values["step"],
		Proof: values["proof"], Indicator: values["indicator"], DependsOn: splitList(values["depends_on"]), Source: location,
	}
	if cell.StableID == "" || cell.Activity == "" || cell.Stage == "" || cell.Step == "" || cell.Proof == "" || cell.Indicator == "" {
		return CellDecl{}, fmt.Errorf("line %d: incomplete cell", line)
	}
	return cell, nil
}

func parseRule(values map[string]string, location SourceLocation, line int) (RuleDecl, error) {
	rule := RuleDecl{
		StableID: values["id"], Cell: values["cell"], State: values["state"], Reason: values["reason"],
		UnknownClass: values["unknown_class"], NextOperation: values["next_operation"], Source: location,
	}
	if rule.StableID == "" || rule.Cell == "" || rule.State == "" || rule.Reason == "" || rule.UnknownClass == "" || rule.NextOperation == "" {
		return RuleDecl{}, fmt.Errorf("line %d: incomplete rule", line)
	}
	if rule.State != DecisionUnknown && rule.State != DecisionRefuted && rule.State != DecisionClosed {
		return RuleDecl{}, fmt.Errorf("line %d: unknown rule state %s", line, rule.State)
	}
	return rule, nil
}

func parseCase(values map[string]string, location SourceLocation, line int) (CaseContract, error) {
	ordinal, err := integerValue(values, "ordinal")
	if err != nil || ordinal < 1 {
		return CaseContract{}, fmt.Errorf("line %d: invalid case ordinal", line)
	}
	caseContract := CaseContract{Ordinal: ordinal, StableID: values["id"], Expected: values["expected"], Source: location}
	if caseContract.StableID == "" || caseContract.Expected == "" {
		return CaseContract{}, fmt.Errorf("line %d: incomplete case", line)
	}
	if caseContract.Expected != DecisionClosed && caseContract.Expected != DecisionUnknown && caseContract.Expected != DecisionRefuted {
		return CaseContract{}, fmt.Errorf("line %d: unknown case decision %s", line, caseContract.Expected)
	}
	return caseContract, nil
}

func validateGraph(graph SemanticGraph) error {
	if graph.GraphID == "" || graph.Release == "" || len(graph.Precedence) != 3 || graph.ExternalRequiredGates != 0 || graph.RepositoryWrites != 0 {
		return errors.New("graph metadata is incomplete or violates the read-only contract")
	}
	if !sameStrings(graph.Precedence, []string{DecisionRefuted, DecisionUnknown, DecisionClosed}) {
		return errors.New("precedence must be REFUTED,UNKNOWN,CLOSED")
	}
	if len(graph.Artifacts) != len(RequiredArtifactNames) {
		return fmt.Errorf("graph must declare exactly %d artifacts", len(RequiredArtifactNames))
	}
	for index, artifact := range graph.Artifacts {
		if artifact.Ordinal != index+1 || artifact.Name != RequiredArtifactNames[index] {
			return fmt.Errorf("artifact ordinal %d is not the fixed output contract", index+1)
		}
	}
	if len(graph.Activities) != 12 || len(graph.Cells) != 12 || len(graph.Cases) != 12 {
		return errors.New("graph must declare exactly twelve activities, cells, and cases")
	}
	activityByName := map[string]ActivityDecl{}
	activityOrdinals := map[int]bool{}
	proofCounts := map[string]int{}
	indicatorCounts := map[string]int{}
	for index, activity := range graph.Activities {
		if activity.Ordinal != index+1 || activityOrdinals[activity.Ordinal] {
			return errors.New("activity ordinals must be unique and contiguous")
		}
		activityOrdinals[activity.Ordinal] = true
		if activity.Authority != "READ_ONLY" {
			return fmt.Errorf("activity %s is not read-only", activity.Name)
		}
		if _, exists := activityByName[activity.Name]; exists {
			return fmt.Errorf("activity %s is declared more than once", activity.Name)
		}
		activityByName[activity.Name] = activity
	}
	cellOrdinals := map[int]bool{}
	boundActivities := map[string]int{}
	for index, cell := range graph.Cells {
		if cell.Ordinal != index+1 || cellOrdinals[cell.Ordinal] {
			return errors.New("cell ordinals must be unique and contiguous")
		}
		cellOrdinals[cell.Ordinal] = true
		activity, exists := activityByName[cell.Activity]
		if !exists {
			return fmt.Errorf("cell %s refers to missing activity %s", cell.StableID, cell.Activity)
		}
		if activity.Proof != cell.Proof || activity.Indicator != cell.Indicator || activity.Artifact == "" {
			return fmt.Errorf("cell %s disagrees with its activity binding", cell.StableID)
		}
		boundActivities[cell.Activity]++
		proofCounts[cell.Proof]++
		indicatorCounts[cell.Indicator]++
	}
	for _, activity := range graph.Activities {
		if boundActivities[activity.Name] != 1 {
			return fmt.Errorf("activity %s must be bound by exactly one cell", activity.Name)
		}
	}
	for _, label := range []string{"FOUNDATION", "COHERENCE", "REGRESSION"} {
		if proofCounts[label] != 4 {
			return fmt.Errorf("proof choice %s must occur exactly four times", label)
		}
	}
	for _, label := range []string{"DRIVER", "OUTCOME", "GUARDRAIL"} {
		if indicatorCounts[label] != 4 {
			return fmt.Errorf("indicator class %s must occur exactly four times", label)
		}
	}
	caseOrdinals := map[int]bool{}
	caseIDs := map[string]bool{}
	caseCounts := map[string]int{}
	for index, caseContract := range graph.Cases {
		if caseContract.Ordinal != index+1 || caseOrdinals[caseContract.Ordinal] || caseIDs[caseContract.StableID] {
			return errors.New("case ordinals and IDs must be unique and contiguous")
		}
		caseOrdinals[caseContract.Ordinal] = true
		caseIDs[caseContract.StableID] = true
		caseCounts[caseContract.Expected]++
	}
	if caseCounts[DecisionClosed] != 4 || caseCounts[DecisionUnknown] != 4 || caseCounts[DecisionRefuted] != 4 {
		return errors.New("case denominator must be CLOSED=4, UNKNOWN=4, REFUTED=4")
	}
	for _, ruleID := range RequiredRuleIDs {
		rule, exists := graph.Rules[ruleID]
		if !exists || rule.Cell == "" {
			return fmt.Errorf("graph is missing rule %s", ruleID)
		}
	}
	return nil
}

func LoadCases(directory string, graph SemanticGraph) ([]CaseInput, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		paths = append(paths, entry.Name())
	}
	sort.Strings(paths)
	if len(paths) != len(graph.Cases) {
		return nil, fmt.Errorf("case directory must contain exactly %d JSON fixtures", len(graph.Cases))
	}
	byID := map[string]CaseContract{}
	for _, caseContract := range graph.Cases {
		byID[caseContract.StableID] = caseContract
	}
	seen := map[string]bool{}
	result := make([]CaseInput, 0, len(paths))
	for _, name := range paths {
		raw, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return nil, err
		}
		var input CaseInput
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			return nil, fmt.Errorf("decode %s: %w", name, err)
		}
		if input.CaseID == "" {
			return nil, fmt.Errorf("fixture %s has no case_id", name)
		}
		if _, exists := byID[input.CaseID]; !exists {
			return nil, fmt.Errorf("fixture %s is not in the graph contract", input.CaseID)
		}
		if seen[input.CaseID] {
			return nil, fmt.Errorf("fixture %s duplicates case %s", name, input.CaseID)
		}
		seen[input.CaseID] = true
		input.FixturePath = filepath.ToSlash(name)
		result = append(result, input)
	}
	if len(seen) != len(graph.Cases) {
		return nil, errors.New("case fixtures do not cover the graph denominator")
	}
	return result, nil
}

func stripComment(line string) string {
	if index := strings.Index(line, "//"); index >= 0 {
		return line[:index]
	}
	return line
}

func keyValues(fields []string) (map[string]string, error) {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, errors.New("invalid key/value")
		}
		if _, exists := values[parts[0]]; exists {
			return nil, fmt.Errorf("duplicate key %s", parts[0])
		}
		values[parts[0]] = strings.Trim(parts[1], "\"'")
	}
	return values, nil
}

func integerValue(values map[string]string, key string) (int, error) {
	value := values[key]
	if value == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return parsed, nil
}

func splitList(value string) []string {
	if value == "" || value == "-" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func registerStableID(seen map[string]string, id, kind string, line int) error {
	if previous, exists := seen[id]; exists {
		return fmt.Errorf("line %d: duplicate stable ID %s (%s already declared)", line, id, previous)
	}
	seen[id] = kind
	return nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
