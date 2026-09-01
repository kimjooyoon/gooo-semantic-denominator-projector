package projector

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	GraphSchema       = "gooo/semantic-denominator-projector/semantic-graph/v1"
	IRSchema          = "gooo/semantic-denominator-projector/semantic-ir/v1"
	DenominatorSchema = "gooo/semantic-denominator-projector/semantic-denominator/v1"
	DistributionSchema = "gooo/semantic-denominator-projector/semantic-distribution/v1"
	AssertionSchema   = "gooo/semantic-denominator-projector/generated-assertions/v1"
	EventSchema       = "gooo/semantic-denominator-projector/projection-event/v1"
	ReplaySchema      = "gooo/semantic-denominator-projector/replay-receipt/v1"
	DecisionClosed   = "CLOSED"
	DecisionUnknown  = "UNKNOWN"
	DecisionRefuted  = "REFUTED"
	ToolchainVersion = "go1.27.0"
)

var RequiredArtifactNames = []string{
	"semantic-denominator.json",
	"semantic-distribution.json",
	"generated-assertions.json",
	"projection-events.ndjson",
	"replay-receipt.json",
	"report.md",
}

var RequiredRuleIDs = []string{
	"duplicate_stable_id",
	"missing_activity",
	"stale_expected_denominator",
	"contradictory_state",
	"malformed_unknown",
	"unknown_top_level_decision",
	"external_user_utility_absent",
	"unbounded_input",
	"improvement_pair_absent",
}

type SourceLocation struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type ArtifactDecl struct {
	Ordinal int            `json:"ordinal"`
	Name    string         `json:"name"`
	Source  SourceLocation `json:"source_location"`
}

type ActivityDecl struct {
	Ordinal   int            `json:"ordinal"`
	StableID  string         `json:"stable_id"`
	Name      string         `json:"name"`
	Proof     string         `json:"proof_choice"`
	Indicator string         `json:"indicator_class"`
	Artifact  string         `json:"artifact"`
	Authority string         `json:"authority"`
	Source    SourceLocation `json:"source_location"`
}

type CellDecl struct {
	Ordinal   int            `json:"ordinal"`
	StableID  string         `json:"stable_id"`
	Activity  string         `json:"activity"`
	Stage     string         `json:"stage"`
	Step      string         `json:"step"`
	Proof     string         `json:"proof_choice"`
	Indicator string         `json:"indicator_class"`
	DependsOn []string       `json:"depends_on"`
	Source    SourceLocation `json:"source_location"`
}

type RuleDecl struct {
	StableID      string         `json:"stable_id"`
	Cell          string         `json:"cell"`
	State         string         `json:"state"`
	Reason        string         `json:"reason"`
	UnknownClass  string         `json:"unknown_class"`
	NextOperation string         `json:"next_operation"`
	Source        SourceLocation `json:"source_location"`
}

type CaseContract struct {
	Ordinal  int            `json:"ordinal"`
	StableID string         `json:"stable_id"`
	Expected string         `json:"expected_decision"`
	Source   SourceLocation `json:"source_location"`
}

type SemanticGraph struct {
	Schema                 string                    `json:"schema"`
	GraphID                string                    `json:"graph_id"`
	Release                string                    `json:"release"`
	Precedence             []string                  `json:"precedence"`
	ExternalRequiredGates  int                       `json:"external_required_gates"`
	RepositoryWrites       int                       `json:"repository_writes"`
	Artifacts              []ArtifactDecl            `json:"artifacts"`
	Activities             []ActivityDecl            `json:"activities"`
	Cells                  []CellDecl                `json:"cells"`
	Rules                  map[string]RuleDecl       `json:"rules"`
	Cases                  []CaseContract             `json:"cases"`
}

type SemanticIR struct {
	Schema       string        `json:"schema"`
	SourcePath   string        `json:"source_path"`
	SourceDigest string        `json:"source_digest"`
	Graph        SemanticGraph `json:"graph"`
}

type SemanticNodeInput struct {
	StableID string         `json:"stable_id"`
	Kind     string         `json:"kind"`
	Source   SourceLocation `json:"source_location"`
}

type UnknownClaim struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

func (u *UnknownClaim) Valid() bool {
	return u != nil && u.Stage != "" && u.Step != "" && u.Reason != "" &&
		u.UnknownClass != "" && u.NextOperation != "" && u.BlockedBy != nil
}

type ImprovementInput struct {
	Before         *int   `json:"before"`
	After          *int   `json:"after"`
	Scenario       string `json:"scenario"`
	SourceDigest   string `json:"source_digest"`
	ContractDigest string `json:"contract_digest"`
	Fixture        string `json:"fixture"`
	Toolchain      string `json:"toolchain"`
	Runner         string `json:"runner"`
}

type CaseInput struct {
	CaseID                    string              `json:"case_id"`
	ExpectedDecision          string              `json:"expected_decision"`
	TopLevelDecision          string              `json:"top_level_decision"`
	ObservedDenominator       *int                `json:"observed_denominator"`
	ObservedActivities        []string            `json:"observed_activities"`
	SemanticNodes             []SemanticNodeInput `json:"semantic_nodes"`
	Claim                     *UnknownClaim       `json:"claim"`
	ContradictoryState        bool                `json:"contradictory_state"`
	MalformedUnknown          bool                `json:"malformed_unknown"`
	UnboundedInput            bool                `json:"unbounded_input"`
	ExternalUserUtility       *bool               `json:"external_user_utility_evidence"`
	Improvement               *ImprovementInput   `json:"improvement"`
	FixturePath               string              `json:"-"`
}

type Issue struct {
	RuleID    string   `json:"rule_id"`
	State     string   `json:"state"`
	Reason    string   `json:"reason"`
	BlockedBy []string `json:"blocked_by"`
}

type ActivityProjection struct {
	CellOrdinal       int            `json:"cell_ordinal"`
	CellID            string         `json:"cell_id"`
	ActivityStableID  string         `json:"activity_stable_id"`
	Activity          string         `json:"activity"`
	Artifact          string         `json:"artifact"`
	Proof             string         `json:"proof_choice"`
	Indicator         string         `json:"indicator_class"`
	Source            SourceLocation `json:"source_location"`
}

type CaseResult struct {
	Ordinal           int                  `json:"ordinal"`
	CaseID            string               `json:"case_id"`
	Expected          string               `json:"expected_decision"`
	Decision          string               `json:"decision"`
	Reason            string               `json:"reason"`
	Unknown           *UnknownClaim        `json:"unknown,omitempty"`
	Issues            []Issue              `json:"issues"`
	StableIDs         []SemanticNodeInput  `json:"semantic_nodes"`
	ActivityMapping   []ActivityProjection `json:"activity_mapping"`
	Improvement       ImprovementResult    `json:"improvement"`
	Utility           UtilityResult        `json:"utility"`
	FixturePath       string               `json:"fixture_path"`
}

type ImprovementResult struct {
	State   string        `json:"state"`
	Before  *int          `json:"before"`
	After   *int          `json:"after"`
	Unknown *UnknownClaim `json:"unknown,omitempty"`
}

type UtilityResult struct {
	State   string        `json:"state"`
	Unknown *UnknownClaim `json:"unknown,omitempty"`
}

type StateCounts struct {
	Total   int `json:"total"`
	Closed  int `json:"closed"`
	Unknown int `json:"unknown"`
	Refuted int `json:"refuted"`
}

type LabeledCount struct {
	Label  string `json:"label"`
	Total  int    `json:"total"`
	Closed int    `json:"closed"`
}

type Inventory struct {
	DescendantDirs    int  `json:"descendant_dirs"`
	RegularFiles      int  `json:"regular_files"`
	GoFiles           int  `json:"go_files"`
	GoPhysicalLines   int  `json:"go_physical_lines"`
	GoooFiles         int  `json:"gooo_files"`
	GoooPhysicalLines int  `json:"gooo_physical_lines"`
	GeneratedFiles    int  `json:"generated_files"`
	GeneratedBytes    int  `json:"generated_bytes"`
	RootREADMEExcluded bool `json:"root_readme_excluded"`
}

type SemanticDenominator struct {
	Schema                string               `json:"schema"`
	IRSchema              string               `json:"ir_schema"`
	SourcePath            string               `json:"source_path"`
	SourceDigest          string               `json:"source_digest"`
	GraphID               string               `json:"graph_id"`
	Release               string               `json:"release"`
	Authority             map[string]any       `json:"authority"`
	Precedence            []string             `json:"precedence"`
	ScenarioDenominator   int                  `json:"scenario_denominator"`
	StateCounts           StateCounts          `json:"state_counts"`
	ExpectedStateCounts   StateCounts          `json:"expected_state_counts"`
	ProofChoices          []LabeledCount       `json:"proof_choices"`
	IndicatorClasses      []LabeledCount       `json:"indicator_classes"`
	Activities            []ActivityDecl       `json:"activities"`
	Cells                 []CellDecl           `json:"cells"`
	Cases                 []CaseResult         `json:"cases"`
	OutputArtifacts       []string             `json:"output_artifacts"`
	Inventory             Inventory            `json:"inventory"`
}

type SemanticDistribution struct {
	Schema              string               `json:"schema"`
	SourceDigest        string               `json:"source_digest"`
	ScenarioDenominator int                  `json:"scenario_denominator"`
	States              StateCounts          `json:"states"`
	ProofChoices        []LabeledCount       `json:"proof_choices"`
	IndicatorClasses    []LabeledCount       `json:"indicator_classes"`
	UnknownCoverage     StateCounts          `json:"unknown_coverage"`
	RefutedCoverage     StateCounts          `json:"refuted_coverage"`
	ActivityMapping     []ActivityProjection `json:"activity_mapping"`
	Improvement         UtilityResult        `json:"improvement"`
	Utility             UtilityResult        `json:"external_user_utility"`
}

type GeneratedAssertion struct {
	Ordinal       int           `json:"ordinal"`
	CaseID        string        `json:"case_id"`
	Expected      string        `json:"expected_decision"`
	Actual        string        `json:"actual_decision"`
	Pass          bool          `json:"pass"`
	Source        SourceLocation `json:"source_location"`
	Reason        string        `json:"reason"`
	Unknown       *UnknownClaim `json:"unknown,omitempty"`
	StableIDs     []string      `json:"stable_ids"`
}

type ReplayReceipt struct {
	Schema                 string   `json:"schema"`
	SourceDigest           string   `json:"source_digest"`
	NormalInputOrder       []string `json:"normal_input_order"`
	OrderPerturbedInputOrder []string `json:"order_perturbed_input_order"`
	NormalDigest           string   `json:"normal_digest"`
	OrderPerturbedDigest   string   `json:"order_perturbed_digest"`
	Match                  bool     `json:"match"`
	State                  string   `json:"state"`
	Reason                 string   `json:"reason"`
}

type ProjectionEvent struct {
	Schema            string         `json:"schema"`
	CaseOrdinal       int            `json:"case_ordinal"`
	CaseID            string         `json:"case_id"`
	CellOrdinal       int            `json:"cell_ordinal"`
	CellID            string         `json:"cell_id"`
	ActivityStableID  string         `json:"activity_stable_id"`
	Activity          string         `json:"activity"`
	Proof             string         `json:"proof_choice"`
	Indicator         string         `json:"indicator_class"`
	Decision          string         `json:"decision"`
	Source            SourceLocation `json:"source_location"`
}

type GenerationResult struct {
	Denominator  SemanticDenominator
	Distribution  SemanticDistribution
	Assertions    []GeneratedAssertion
	Events        []ProjectionEvent
	Replay        ReplayReceipt
	Report        string
}

func DigestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func JSON(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}
