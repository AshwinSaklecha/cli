// Package hold is the checkpoint-native continuation gate: extract promises
// from Entire checkpoints, bind them onto Graph symbols, freeze a neighborhood,
// and fail later edits that intersect it. prove additionally resurrects
// dependent historical intent.
package hold

import "time"

// CharterVersion is the charter.json schema version.
const CharterVersion = "1"

// Closed item statuses. UNBOUND never enters freeze_set.
const (
	StatusUnbound          = "UNBOUND"
	StatusFrozen           = "FROZEN"
	StatusOpen             = "OPEN"
	StatusUnverified       = "UNVERIFIED"
	StatusConflict         = "CONFLICT"
	StatusSuperseded       = "SUPERSEDED"
	StatusIntentRegression = "INTENT_REGRESSION"
	StatusIntactProven     = "INTACT_PROVEN"
)

// Artifact paths under the worktree, repo-relative.
const (
	CharterRelPath   = ".entire/hold/charter.json"
	PermitRelPath    = ".entire/hold/permit.md"
	LastCheckRelPath = ".entire/hold/last-check.json"
	LastProveRelPath = ".entire/hold/last-prove.json"
	BaselinesRelDir  = ".entire/hold/baselines"
)

// HoldHookMarker identifies the Hold pre-commit hook so install can replace it
// without touching Entire's managed git hooks (which do not include pre-commit).
const HoldHookMarker = "Entire Hold gate"

// BypassEnv skips the pre-commit check. Documented, never used in the demo.
const BypassEnv = "HOLD_BYPASS"

// DefaultFreezeDepth is graph impact depth for the 6-caller fixture.
const DefaultFreezeDepth = 2

// Charter is .entire/hold/charter.json — the gate source of truth.
type Charter struct {
	Version           string            `json:"version"`
	Repo              string            `json:"repo"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	SourceCheckpoints []string          `json:"source_checkpoints"`
	GraphPlugin       GraphPluginInfo   `json:"graph_plugin"`
	Items             []Item            `json:"items"`
	DependencyPolicy  *DependencyPolicy `json:"dependency_policy,omitempty"`
	Databricks        DatabricksCfg     `json:"databricks"`
	FreezeDepth       int               `json:"freeze_depth,omitempty"`
	LastAmend         string            `json:"last_amend,omitempty"`
	LastAmendReason   string            `json:"last_amend_reason,omitempty"`
}

// GraphPluginInfo records the graph used at compile time.
type GraphPluginInfo struct {
	Version  string `json:"version"`
	DoctorOK bool   `json:"doctor_ok"`
}

// DatabricksCfg is optional recording; check/prove must not require it.
type DatabricksCfg struct {
	Enabled bool `json:"enabled"`
}

// DependencyPolicy hashes manifests named by a FROZEN "no new runtime deps" item.
type DependencyPolicy struct {
	Constraint string            `json:"constraint"`
	Files      map[string]string `json:"files"` // path → sha256 hex
}

// Item is one extracted constraint.
type Item struct {
	ID           string        `json:"id"`
	Status       string        `json:"status"`
	Constraint   string        `json:"constraint"`
	CheckpointID string        `json:"checkpoint_id"`
	Evidence     Evidence      `json:"evidence"`
	Symbol       *Symbol       `json:"symbol,omitempty"`
	FreezeSet    []FreezeEntry `json:"freeze_set,omitempty"`
	Impact       ImpactInfo    `json:"impact"`
	Supersedes   string        `json:"supersedes,omitempty"`
	Reason       string        `json:"reason,omitempty"`
	Kind         string        `json:"kind,omitempty"` // function, behavioral, manifest
	Candidates   []Symbol      `json:"candidates,omitempty"`
}

// Evidence cites the transcript.
type Evidence struct {
	Quote         string `json:"quote"`
	TranscriptRef string `json:"transcript_ref"`
}

// Symbol is a Graph-cited entity.
type Symbol struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
	Kind string `json:"kind,omitempty"`
}

// FreezeEntry is one member of a frozen neighborhood.
type FreezeEntry struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
	Kind string `json:"kind,omitempty"`
}

// ImpactInfo stores callers captured at freeze time.
type ImpactInfo struct {
	Callers []string `json:"callers,omitempty"`
}

// Change is one entity from graph diff (or a worktree overlay).
type Change struct {
	Name   string `json:"name"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Kind   string `json:"kind,omitempty"` // body-changed, added, removed, renamed
	Detail string `json:"detail,omitempty"`
}

// CheckResult is written to last-check.json.
type CheckResult struct {
	Passed        bool           `json:"passed"`
	At            time.Time      `json:"at"`
	Base          string         `json:"base,omitempty"`
	Head          string         `json:"head,omitempty"`
	Intersections []Intersection `json:"intersections,omitempty"`
	ManifestBreak []string       `json:"manifest_break,omitempty"`
	Message       string         `json:"message,omitempty"`
}

// Intersection is a freeze_set hit.
type Intersection struct {
	Frozen       FreezeEntry `json:"frozen"`
	Changed      Change      `json:"changed"`
	ItemID       string      `json:"item_id"`
	Callers      []string    `json:"callers,omitempty"`
	Quote        string      `json:"quote,omitempty"`
	CheckpointID string      `json:"checkpoint_id,omitempty"`
}

// ProveResult is written to last-prove.json.
type ProveResult struct {
	Passed            bool            `json:"passed"`
	At                time.Time       `json:"at"`
	FreezeOK          bool            `json:"freeze_ok"`
	VerifyOK          bool            `json:"verify_ok"`
	TestCommand       string          `json:"test_command,omitempty"`
	IntentRegressions []IntentFinding `json:"intent_regressions,omitempty"`
	Intact            []string        `json:"intact,omitempty"`
	Message           string          `json:"message,omitempty"`
}

// IntentFinding is a Leg B fail.
type IntentFinding struct {
	Dependent    Symbol `json:"dependent"`
	CheckpointID string `json:"checkpoint_id"`
	Quote        string `json:"quote"`
	Change       string `json:"change"`
	Composition  string `json:"composition,omitempty"`
	Why          string `json:"why"`
}

// Extracted is a constraint before bind.
type Extracted struct {
	Constraint    string
	Quote         string
	CheckpointID  string
	TranscriptRef string
	SymbolGuess   string
}

// MayFreeze reports whether status is allowed in freeze_set.
func MayFreeze(status string) bool {
	return status == StatusFrozen || status == StatusConflict
}

// FrozenItems returns items that still enforce.
func FrozenItems(c *Charter) []Item {
	if c == nil {
		return nil
	}
	out := make([]Item, 0, len(c.Items))
	for _, it := range c.Items {
		if MayFreeze(it.Status) {
			out = append(out, it)
		}
	}
	return out
}
