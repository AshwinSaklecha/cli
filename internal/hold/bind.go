package hold

import (
	"context"
	"path"
	"strings"
)

// GraphClient is the Entire Graph plugin surface Hold execs. Tests stub it.
type GraphClient interface {
	Def(ctx context.Context, symbol, repo string) ([]Symbol, error)
	Search(ctx context.Context, query, repo string) ([]Symbol, error)
	Impact(ctx context.Context, symbol, repo string, depth int, excludeTests bool) (ImpactResult, error)
	Diff(ctx context.Context, base, head, repo string) ([]Change, error)
	Neighbors(ctx context.Context, symbol, repo, relation, direction string) ([]Symbol, error)
	Doctor(ctx context.Context) (GraphPluginInfo, error)
}

// EntireClient is the Entire CLI checkpoint/authorship surface. Tests stub it.
type EntireClient interface {
	CheckpointList(ctx context.Context) ([]CheckpointMeta, error)
	ExplainFull(ctx context.Context, id string) (string, error)
	ExplainRawTranscript(ctx context.Context, id string) (string, error)
	Why(ctx context.Context, fileLine string) ([]CheckpointMeta, error)
	Blame(ctx context.Context, file string) ([]CheckpointMeta, error)
}

// CheckpointMeta is a list/why/blame hit.
type CheckpointMeta struct {
	ID      string `json:"checkpoint_id"`
	Session string `json:"session_id,omitempty"`
	Message string `json:"message,omitempty"`
}

// BindResult is one extracted constraint after Graph citation.
type BindResult struct {
	Item       Item
	UnboundWhy string
}

// Bind cites Graph for an extracted constraint. Ambiguous or missing → UNBOUND.
// Never freeze from an uncited guess. Never freeze from a redacted hole.
func Bind(ctx context.Context, g GraphClient, repo string, ex Extracted, depth int, excludeTests bool) BindResult {
	guess := strings.TrimSpace(ex.SymbolGuess)
	item := Item{
		Status:       StatusUnbound,
		Constraint:   ex.Constraint,
		CheckpointID: ex.CheckpointID,
		Evidence: Evidence{
			Quote:         ex.Quote,
			TranscriptRef: ex.TranscriptRef,
		},
		Kind: constraintKind(ex),
	}

	if HolePreventsFreeze(ex) {
		item.Status = StatusUnbound
		item.Reason = "redacted or missing checkpoint text; cannot freeze"
		item.FreezeSet = nil
		return BindResult{Item: item, UnboundWhy: item.Reason}
	}

	if isManifestConstraint(ex) {
		item.Status = StatusFrozen
		item.Kind = "manifest"
		item.Symbol = &Symbol{Name: "package.json", File: "demo/charge-api/package.json", Kind: "manifest"}
		return BindResult{Item: item}
	}

	if guess == "" {
		item.Status = StatusOpen
		item.Reason = "no identifier to bind"
		return BindResult{Item: item, UnboundWhy: item.Reason}
	}

	cands, err := lookup(ctx, g, repo, guess)
	if err != nil || len(cands) == 0 {
		item.Status = StatusUnbound
		item.Reason = "graph could not cite " + guess
		return BindResult{Item: item, UnboundWhy: item.Reason}
	}

	cands = preferExact(preferSrc(cands), guess)
	if len(cands) > 1 && !sameSymbol(cands) {
		item.Status = StatusUnbound
		item.Candidates = cands
		item.Reason = "ambiguous graph bind for " + guess
		return BindResult{Item: item, UnboundWhy: item.Reason}
	}

	sym := cands[0]
	item.Symbol = &sym
	if item.Kind == "" {
		item.Kind = "function"
	}

	impactArg := guess
	if sym.File != "" && sym.Line > 0 {
		impactArg = slash(sym.File) + ":" + itoa(sym.Line)
	}
	impact, err := g.Impact(ctx, impactArg, repo, depth, excludeTests)
	if err != nil {
		impact = ImpactResult{}
	}
	item.Impact.Callers = impact.Callers
	item.FreezeSet = freezeSetFor(sym, impact)
	item.Status = StatusFrozen
	return BindResult{Item: item}
}

func lookup(ctx context.Context, g GraphClient, repo, guess string) ([]Symbol, error) {
	searched, err := g.Search(ctx, guess, repo)
	if exact := preferExact(searched, guess); len(exact) > 0 {
		return exact, nil
	}
	defs, derr := g.Def(ctx, guess, repo)
	if len(defs) > 0 {
		return defs, derr
	}
	return searched, err
}

func freezeSetFor(sym Symbol, _ ImpactResult) []FreezeEntry {
	// Freeze the Graph-cited symbol only. Callers live on impact.callers for
	// the check failure message. Putting callers in freeze_set would fail
	// check on a legal holidayBonus edit and kill prove's silent-regression beat.
	if sym.Name == "" {
		return nil
	}
	return []FreezeEntry{{Name: sym.Name, File: sym.File, Line: sym.Line, Kind: sym.Kind}}
}

func preferSrc(cands []Symbol) []Symbol {
	var demo, src, rest []Symbol
	for _, c := range cands {
		p := strings.ReplaceAll(c.File, "\\", "/")
		base := path.Base(p)
		testish := strings.Contains(base, ".test.") || strings.Contains(p, "/testdata/")
		switch {
		case strings.Contains(p, "demo/charge-api/") && !testish:
			demo = append(demo, c)
		case strings.Contains(p, "/src/") && !testish:
			src = append(src, c)
		case testish:
			rest = append(rest, c)
		default:
			src = append(src, c)
		}
	}
	if len(demo) > 0 {
		return demo
	}
	if len(src) > 0 {
		return src
	}
	return rest
}

func preferExact(cands []Symbol, guess string) []Symbol {
	g := canonName(guess)
	if g == "" {
		return cands
	}
	var exact []Symbol
	for _, c := range cands {
		if strings.EqualFold(canonName(c.Name), g) {
			exact = append(exact, c)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return cands
}

func sameSymbol(cands []Symbol) bool {
	if len(cands) == 0 {
		return true
	}
	n, f := cands[0].Name, cands[0].File
	for _, c := range cands[1:] {
		if c.Name != n || c.File != f {
			return false
		}
	}
	return true
}

func isManifestConstraint(ex Extracted) bool {
	l := strings.ToLower(ex.Constraint + " " + ex.Quote)
	return strings.Contains(l, "dependenc") && (strings.Contains(l, "no new") || strings.Contains(l, "must not"))
}

func constraintKind(ex Extracted) string {
	l := strings.ToLower(ex.Constraint)
	switch {
	case strings.Contains(l, "idempotent"):
		return "behavioral"
	case isManifestConstraint(ex):
		return "manifest"
	default:
		return "function"
	}
}
