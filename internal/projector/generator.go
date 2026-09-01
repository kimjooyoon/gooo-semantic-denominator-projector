package projector

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Generate(sourcePath, casesPath, outputPath, inventoryRoot string) (GenerationResult, error) {
	if err := EnsureCallerDirectory(outputPath); err != nil {
		return GenerationResult{}, err
	}
	ir, _, err := LoadGraph(sourcePath)
	if err != nil {
		return GenerationResult{}, err
	}
	inputs, err := LoadCases(casesPath, ir.Graph)
	if err != nil {
		return GenerationResult{}, err
	}
	inventory, err := InventoryForRoot(inventoryRoot)
	if err != nil {
		return GenerationResult{}, err
	}
	normalResults := EvaluateCases(ir, inputs)
	perturbedInputs := append([]CaseInput(nil), inputs...)
	reverseCases(perturbedInputs)
	perturbedResults := EvaluateCases(ir, perturbedInputs)
	normalDigest := digestReplay(ir, normalResults)
	perturbedDigest := digestReplay(ir, perturbedResults)
	replay := ReplayReceipt{
		Schema: ReplaySchema, SourceDigest: ir.SourceDigest,
		NormalInputOrder: caseOrder(inputs), OrderPerturbedInputOrder: caseOrder(perturbedInputs),
		NormalDigest: normalDigest, OrderPerturbedDigest: perturbedDigest,
		Match: normalDigest == perturbedDigest, State: DecisionClosed, Reason: "ORDER_PERTURBED_REPLAY_MATCH",
	}
	if !replay.Match {
		replay.State = DecisionRefuted
		replay.Reason = "ORDER_PERTURBED_REPLAY_MISMATCH"
	}
	result := GenerationResult{
		Replay: replay,
		Assertions: buildAssertions(ir, normalResults),
		Events: buildEvents(normalResults),
	}
	result.Denominator = buildDenominator(ir, normalResults, inventory)
	result.Distribution = buildDistribution(ir, normalResults)
	result.Report = renderReport(ir, normalResults, replay)
	if err := writeGeneration(outputPath, result); err != nil {
		return GenerationResult{}, err
	}
	return result, nil
}

func EnsureCallerDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("output directory must be an absolute caller-owned path")
	}
	if err := ensureOutsideRepository(path); err != nil {
		return err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("output path must be a directory")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("caller-owned output directory must be empty")
	}
	return nil
}

func writeGeneration(outputPath string, result GenerationResult) error {
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return err
	}
	outputs, err := outputBytes(result)
	if err != nil {
		return err
	}
	for _, name := range RequiredArtifactNames {
		if err := os.WriteFile(filepath.Join(outputPath, name), outputs[name], 0o644); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(outputPath)
	if err != nil {
		return err
	}
	if len(entries) != len(RequiredArtifactNames) {
		return errors.New("generation output does not contain exactly six artifacts")
	}
	return nil
}

func outputBytes(result GenerationResult) (map[string][]byte, error) {
	denominator, err := jsonWithNewline(result.Denominator)
	if err != nil {
		return nil, err
	}
	distribution, err := jsonWithNewline(result.Distribution)
	if err != nil {
		return nil, err
	}
	assertions, err := jsonWithNewline(struct {
		Schema     string                `json:"schema"`
		Assertions []GeneratedAssertion  `json:"assertions"`
	}{Schema: AssertionSchema, Assertions: result.Assertions})
	if err != nil {
		return nil, err
	}
	replay, err := jsonWithNewline(result.Replay)
	if err != nil {
		return nil, err
	}
	events := bytes.Buffer{}
	for _, event := range result.Events {
		raw, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}
		events.Write(raw)
		events.WriteByte('\n')
	}
	return map[string][]byte{
		"semantic-denominator.json": denominator,
		"semantic-distribution.json": distribution,
		"generated-assertions.json": assertions,
		"projection-events.ndjson": events.Bytes(),
		"replay-receipt.json": replay,
		"report.md": []byte(result.Report),
	}, nil
}

func jsonWithNewline(value any) ([]byte, error) {
	raw, err := JSON(value)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func buildDenominator(ir SemanticIR, results []CaseResult, inventory Inventory) SemanticDenominator {
	expected := expectedStateCounts(ir.Graph.Cases)
	return SemanticDenominator{
		Schema: DenominatorSchema, IRSchema: ir.Schema, SourcePath: ir.SourcePath, SourceDigest: ir.SourceDigest,
		GraphID: ir.Graph.GraphID, Release: ir.Graph.Release,
		Authority: map[string]any{
			"semantic_graph": "RELEASED_GOOO",
			"go_role": []string{"PARSER", "EVALUATOR", "GENERATOR", "RUNTIME"},
			"repository_writes": ir.Graph.RepositoryWrites,
			"external_required_gates": ir.Graph.ExternalRequiredGates,
			"caller_owned_output": true,
		},
		Precedence: ir.Graph.Precedence, ScenarioDenominator: len(ir.Graph.Cases), StateCounts: stateCounts(results), ExpectedStateCounts: expected,
		ProofChoices: graphLabelCounts(ir.Graph, "proof"), IndicatorClasses: graphLabelCounts(ir.Graph, "indicator"),
		Activities: ir.Graph.Activities, Cells: ir.Graph.Cells, Cases: results, OutputArtifacts: append([]string(nil), RequiredArtifactNames...), Inventory: inventory,
	}
}

func buildDistribution(ir SemanticIR, results []CaseResult) SemanticDistribution {
	unknown := 0
	refuted := 0
	for _, result := range results {
		if result.Decision == DecisionUnknown {
			unknown++
		}
		if result.Decision == DecisionRefuted {
			refuted++
		}
	}
	return SemanticDistribution{
		Schema: DistributionSchema, SourceDigest: ir.SourceDigest, ScenarioDenominator: len(ir.Graph.Cases), States: stateCounts(results),
		ProofChoices: graphLabelCounts(ir.Graph, "proof"), IndicatorClasses: graphLabelCounts(ir.Graph, "indicator"),
		UnknownCoverage: StateCounts{Total: unknown, Unknown: unknown}, RefutedCoverage: StateCounts{Total: refuted, Refuted: refuted},
		ActivityMapping: graphActivityMapping(ir.Graph), Improvement: summarizeImprovement(results), Utility: summarizeUtility(results),
	}
}

func graphLabelCounts(graph SemanticGraph, kind string) []LabeledCount {
	labels := []string{}
	counts := map[string]int{}
	for _, cell := range graph.Cells {
		label := cell.Proof
		if kind == "indicator" {
			label = cell.Indicator
		}
		if counts[label] == 0 {
			labels = append(labels, label)
		}
		counts[label]++
	}
	sort.Strings(labels)
	result := make([]LabeledCount, 0, len(labels))
	for _, label := range labels {
		result = append(result, LabeledCount{Label: label, Total: counts[label], Closed: counts[label]})
	}
	return result
}

func expectedStateCounts(cases []CaseContract) StateCounts {
	counts := StateCounts{Total: len(cases)}
	for _, value := range cases {
		switch value.Expected {
		case DecisionClosed:
			counts.Closed++
		case DecisionUnknown:
			counts.Unknown++
		case DecisionRefuted:
			counts.Refuted++
		}
	}
	return counts
}

func stateCounts(results []CaseResult) StateCounts {
	counts := StateCounts{Total: len(results)}
	for _, value := range results {
		switch value.Decision {
		case DecisionClosed:
			counts.Closed++
		case DecisionUnknown:
			counts.Unknown++
		case DecisionRefuted:
			counts.Refuted++
		}
	}
	return counts
}

func buildAssertions(ir SemanticIR, results []CaseResult) []GeneratedAssertion {
	contracts := map[string]CaseContract{}
	for _, value := range ir.Graph.Cases {
		contracts[value.StableID] = value
	}
	assertions := make([]GeneratedAssertion, 0, len(results))
	for _, result := range results {
		contract := contracts[result.CaseID]
		ids := []string{}
		for _, node := range result.StableIDs {
			ids = append(ids, node.StableID)
		}
		sort.Strings(ids)
		assertions = append(assertions, GeneratedAssertion{
			Ordinal: result.Ordinal, CaseID: result.CaseID, Expected: result.Expected, Actual: result.Decision,
			Pass: result.Expected == result.Decision, Source: contract.Source, Reason: result.Reason,
			Unknown: result.Unknown, StableIDs: ids,
		})
	}
	return assertions
}

func buildEvents(results []CaseResult) []ProjectionEvent {
	events := []ProjectionEvent{}
	for _, result := range results {
		for _, mapping := range result.ActivityMapping {
			events = append(events, ProjectionEvent{
				Schema: EventSchema, CaseOrdinal: result.Ordinal, CaseID: result.CaseID, CellOrdinal: mapping.CellOrdinal,
				CellID: mapping.CellID, ActivityStableID: mapping.ActivityStableID, Activity: mapping.Activity,
				Proof: mapping.Proof, Indicator: mapping.Indicator, Decision: result.Decision, Source: mapping.Source,
			})
		}
	}
	return events
}

type replayCase struct {
	Ordinal  int      `json:"ordinal"`
	CaseID   string   `json:"case_id"`
	Decision string   `json:"decision"`
	Issues   []Issue  `json:"issues"`
	StableIDs []string `json:"stable_ids"`
}

func digestReplay(ir SemanticIR, results []CaseResult) string {
	canonical := make([]replayCase, 0, len(results))
	for _, result := range results {
		ids := []string{}
		for _, node := range result.StableIDs {
			ids = append(ids, node.StableID)
		}
		sort.Strings(ids)
		canonical = append(canonical, replayCase{Ordinal: result.Ordinal, CaseID: result.CaseID, Decision: result.Decision, Issues: result.Issues, StableIDs: ids})
	}
	sort.Slice(canonical, func(left, right int) bool { return canonical[left].Ordinal < canonical[right].Ordinal })
	payload := struct {
		Schema       string       `json:"schema"`
		SourceDigest string       `json:"source_digest"`
		Cases        []replayCase `json:"cases"`
	}{Schema: IRSchema, SourceDigest: ir.SourceDigest, Cases: canonical}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return DigestBytes(raw)
}

func reverseCases(inputs []CaseInput) {
	for left, right := 0, len(inputs)-1; left < right; left, right = left+1, right-1 {
		inputs[left], inputs[right] = inputs[right], inputs[left]
	}
}

func caseOrder(inputs []CaseInput) []string {
	result := make([]string, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, input.CaseID)
	}
	return result
}

func summarizeImprovement(results []CaseResult) UtilityResult {
	for _, result := range results {
		if result.Improvement.State != DecisionClosed {
			return UtilityResult{State: DecisionUnknown, Unknown: result.Improvement.Unknown}
		}
	}
	return UtilityResult{State: DecisionClosed}
}

func summarizeUtility(results []CaseResult) UtilityResult {
	for _, result := range results {
		if result.Utility.State != DecisionClosed {
			return UtilityResult{State: DecisionUnknown, Unknown: result.Utility.Unknown}
		}
	}
	return UtilityResult{State: DecisionClosed}
}

func InventoryForRoot(root string) (Inventory, error) {
	if root == "" {
		return Inventory{}, errors.New("inventory root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return Inventory{}, err
	}
	inventory := Inventory{RootREADMEExcluded: true}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != root && entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			inventory.DescendantDirs++
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		inventory.RegularFiles++
		if relative == "README.md" {
			return nil
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".go" && ext != ".gooo" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := physicalLines(raw)
		if ext == ".go" {
			inventory.GoFiles++
			inventory.GoPhysicalLines += lines
		} else {
			inventory.GoooFiles++
			inventory.GoooPhysicalLines += lines
		}
		return nil
	})
	return inventory, err
}

func physicalLines(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	lines := bytes.Count(raw, []byte{'\n'})
	if raw[len(raw)-1] != '\n' {
		lines++
	}
	return lines
}

func ensureOutsideRepository(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	root := findRepositoryRoot()
	if root == "" {
		return nil
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return fmt.Errorf("caller-owned output must be outside repository: %s", absolute)
	}
	return nil
}

func findRepositoryRoot() string {
	current, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		info, err := os.Stat(filepath.Join(current, ".git"))
		if err == nil && info.IsDir() {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}
