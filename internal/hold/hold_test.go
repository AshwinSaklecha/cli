package hold

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractConstraints_ChargeAndVIP(t *testing.T) {
	t.Parallel()
	blob := `Do not change public charge().
Retries must stay idempotent.
No new runtime dependencies.
VIP customers never receive more than 50% total discount, regardless of overlapping promotions.
Add tests for every caller of charge.`
	got := ExtractConstraints("cp1", blob)
	if len(got) < 3 {
		t.Fatalf("extracted %d, want at least 3: %+v", len(got), got)
	}
	joined := ""
	for _, e := range got {
		joined += strings.ToLower(e.Constraint) + "\n"
	}
	for _, need := range []string{"charge", "idempotent", "dependenc"} {
		if !strings.Contains(joined, need) {
			t.Errorf("missing %q in %s", need, joined)
		}
	}
}

func TestUNBOUNDNeverInFreezeSet(t *testing.T) {
	t.Parallel()
	g := stubGraph{}
	br := Bind(context.Background(), g, ".", Extracted{
		Constraint:  "do not change mysteryFn()",
		SymbolGuess: "mysteryFn",
	}, 2, true)
	if br.Item.Status != StatusUnbound {
		t.Fatalf("status=%s want UNBOUND", br.Item.Status)
	}
	if len(br.Item.FreezeSet) != 0 {
		t.Fatalf("UNBOUND freeze_set=%v", br.Item.FreezeSet)
	}
	if MayFreeze(br.Item.Status) {
		t.Fatal("UNBOUND must not be allowed in freeze_set")
	}
}

func TestCheck_IntersectionFail(t *testing.T) {
	t.Parallel()
	c := fixtureCharter()
	res := RunCheck(c, []Change{{
		Name: "charge",
		File: "demo/charge-api/src/charge.ts",
		Line: 12,
		Kind: "body-changed",
	}}, nil)
	if res.Passed {
		t.Fatal("expected fail")
	}
	if len(res.Intersections) == 0 {
		t.Fatal("expected intersection")
	}
	msg := FormatCheckFailure(c, res)
	if !strings.Contains(msg, "HOLD CHECK FAILED") {
		t.Fatal(msg)
	}
	if !strings.Contains(msg, "AGENTS.md") {
		t.Fatal(msg)
	}
	if !strings.Contains(msg, "processPayment") {
		t.Fatal(msg)
	}
}

func TestCheck_DisjointPass(t *testing.T) {
	t.Parallel()
	c := fixtureCharter()
	res := RunCheck(c, []Change{{
		Name: "holidayBonus",
		File: "demo/charge-api/src/holiday.ts",
		Line: 4,
		Kind: "added",
	}}, nil)
	if !res.Passed {
		t.Fatalf("disjoint change should pass: %+v", res)
	}
}

func TestCheck_DependencyHashFail(t *testing.T) {
	t.Parallel()
	c := fixtureCharter()
	c.DependencyPolicy = &DependencyPolicy{
		Constraint: "no new runtime dependencies",
		Files:      map[string]string{"demo/charge-api/package.json": "abc"},
	}
	res := RunCheck(c, nil, map[string]string{"demo/charge-api/package.json": "def"})
	if res.Passed || len(res.ManifestBreak) == 0 {
		t.Fatalf("want manifest break, got %+v", res)
	}
}

func TestAmend_ConflictAndSupersede(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	g := stubGraph{
		defs: map[string][]Symbol{
			"charge": {{Name: "charge", File: "demo/charge-api/src/charge.ts", Line: 12, Kind: "function"}},
		},
		impact: map[string]ImpactResult{
			"charge": {Callers: []string{"processPayment"}, Neighborhood: []FreezeEntry{{Name: "charge", File: "demo/charge-api/src/charge.ts", Line: 12}}},
		},
	}
	c := fixtureCharter()
	conflict, err := Amend(ctx, g, ".", c, AmendInput{Text: "allow a new parameter on charge()"})
	if err != nil {
		t.Fatal(err)
	}
	if len(conflict.Conflicts) == 0 {
		t.Fatalf("expected CONFLICT, got %+v", conflict)
	}
	still := false
	for _, it := range c.Items {
		if it.ID == "H001" && it.Status == StatusFrozen {
			still = true
		}
	}
	if !still {
		t.Fatal("original freeze must still enforce during CONFLICT")
	}

	c2 := fixtureCharter()
	sup, err := Amend(ctx, g, ".", c2, AmendInput{
		Text:      "allow a new parameter on charge() but keep retries idempotent",
		Supersede: "H001",
		Reason:    "noon curveball",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sup.Superseded) == 0 {
		t.Fatalf("expected SUPERSEDED, got %+v", sup)
	}
	found := false
	for _, it := range c2.Items {
		if it.ID == "H001" && it.Status == StatusSuperseded {
			found = true
		}
	}
	if !found {
		t.Fatal("H001 should be SUPERSEDED")
	}
}

func TestIntentRegression_VIPHoliday(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
	  "dependent": {"name":"processPayment","file":"demo/charge-api/src/processPayment.ts","line":40},
	  "quote": "VIP customers never receive more than 50% total discount, regardless of overlapping promotions.",
	  "checkpoint_id": "01HOLDVIP",
	  "changed": [{"name":"holidayBonus","file":"demo/charge-api/src/holiday.ts","kind":"added"}]
	}`)
	var spec struct {
		Dependent    Symbol   `json:"dependent"`
		Quote        string   `json:"quote"`
		CheckpointID string   `json:"checkpoint_id"`
		Changed      []Change `json:"changed"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	hits := EvaluateIntent(spec.Dependent, spec.Quote, spec.CheckpointID, spec.Changed, []Symbol{
		{Name: "vipDiscount"}, {Name: "holidayBonus"}, {Name: "processPayment"},
	})
	if len(hits) != 1 {
		t.Fatalf("want 1 INTENT_REGRESSION, got %d", len(hits))
	}
	msg := FormatProveFailure(hits[0])
	if !strings.Contains(msg, "INTENT REGRESSION") || !strings.Contains(msg, "60%") {
		t.Fatal(msg)
	}

	c := fixtureCharter()
	c.Items = append(c.Items, Item{
		ID:           "Hvip",
		Status:       StatusOpen,
		Constraint:   spec.Quote,
		CheckpointID: spec.CheckpointID,
		Evidence:     Evidence{Quote: spec.Quote},
	})
	pr := RunProve(c, ProveInput{
		VerifyOK:    true,
		Changed:     spec.Changed,
		CurrentDiff: spec.Changed,
		Neighbors:   []Symbol{{Name: "holidayBonus"}, {Name: "vipDiscount"}},
	})
	if pr.Passed || len(pr.IntentRegressions) == 0 {
		t.Fatalf("prove should fail Leg B: %+v", pr)
	}
}

func TestParseGraphDiff_StubJSON(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"changes":[{"symbol_name":"charge","file_path":"demo/charge-api/src/charge.ts","start_line":12,"change":"body-changed"}]}`)
	got := ParseGraphDiffChanges(raw)
	if len(got) == 0 || got[0].Name != "charge" {
		t.Fatalf("%+v", got)
	}
}

func TestParseWhyBlame_StubJSON(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"hits":[{"checkpoint_id":"abc","file":"src/processPayment.ts","line":40}]}`)
	got := collectCheckpointIDs(raw)
	if len(got) != 1 || got[0].ID != "abc" {
		t.Fatalf("%+v", got)
	}
}

func fixtureCharter() *Charter {
	return &Charter{
		Version: CharterVersion,
		Items: []Item{{
			ID:           "H001",
			Status:       StatusFrozen,
			Constraint:   "do not change charge()",
			CheckpointID: "01HABC",
			Evidence:     Evidence{Quote: "do not change charge()"},
			Symbol:       &Symbol{Name: "charge", File: "demo/charge-api/src/charge.ts", Line: 12, Kind: "function"},
			FreezeSet: []FreezeEntry{
				{Name: "charge", File: "demo/charge-api/src/charge.ts", Line: 12},
			},
			Impact: ImpactInfo{Callers: []string{"processPayment", "webhookRetry", "invoiceCharge", "scheduledCharge", "refundRetry", "adminCharge"}},
		}},
	}
}

type stubGraph struct {
	defs   map[string][]Symbol
	search map[string][]Symbol
	impact map[string]ImpactResult
}

func (s stubGraph) Def(_ context.Context, symbol, _ string) ([]Symbol, error) {
	return s.defs[symbol], nil
}
func (s stubGraph) Search(_ context.Context, query, _ string) ([]Symbol, error) {
	return s.search[query], nil
}
func (s stubGraph) Impact(_ context.Context, symbol, _ string, _ int, _ bool) (ImpactResult, error) {
	return s.impact[symbol], nil
}
func (s stubGraph) Diff(context.Context, string, string, string) ([]Change, error) {
	return nil, nil
}
func (s stubGraph) Neighbors(context.Context, string, string, string, string) ([]Symbol, error) {
	return nil, nil
}
func (s stubGraph) Doctor(context.Context) (GraphPluginInfo, error) {
	return GraphPluginInfo{Version: "test", DoctorOK: true}, nil
}
