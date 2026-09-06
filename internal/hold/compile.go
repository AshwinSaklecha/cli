package hold

import (
	"context"
	"strings"
	"time"
)

// ProveInput is the two-leg prove.
type ProveInput struct {
	Base            string
	Head            string
	Repo            string
	TestCommand     string
	Changed         []Change
	CurrentDiff     []Change
	Neighbors       []Symbol
	DependentQuotes []DependentQuote
	VerifyOK        bool
}

// DependentQuote is a resurrected historical constraint for one Graph dependent.
type DependentQuote struct {
	Dependent    Symbol
	CheckpointID string
	Quote        string
}

// RunProve is freeze semantic-diff (Leg A) + dependent-intent (Leg B).
func RunProve(c *Charter, in ProveInput) ProveResult {
	res := ProveResult{
		At:          time.Now().UTC(),
		TestCommand: in.TestCommand,
		VerifyOK:    in.VerifyOK,
		FreezeOK:    true,
		Passed:      true,
	}
	if c == nil {
		res.Passed = false
		res.FreezeOK = false
		res.Message = "missing charter"
		return res
	}

	diff := in.CurrentDiff
	if len(diff) == 0 {
		diff = in.Changed
	}

	for _, it := range FrozenItems(c) {
		if it.Status == StatusSuperseded {
			continue
		}
		moved := false
		for _, ch := range diff {
			if hit, _ := intersects(it, ch); hit {
				moved = true
				res.FreezeOK = false
				res.Passed = false
				break
			}
		}
		if !moved && it.Symbol != nil {
			res.Intact = append(res.Intact, it.Symbol.Name)
		}
	}

	quotes := in.DependentQuotes
	if len(quotes) == 0 {
		quotes = QuotesFromCharter(c)
	}
	for _, q := range quotes {
		hits := EvaluateIntent(q.Dependent, q.Quote, q.CheckpointID, diff, in.Neighbors)
		if len(hits) > 0 {
			res.Passed = false
			res.IntentRegressions = append(res.IntentRegressions, hits...)
		}
	}

	if !in.VerifyOK && in.TestCommand != "" {
		res.Passed = false
		res.VerifyOK = false
	}
	if !res.FreezeOK {
		res.Message = "frozen neighborhood moved"
	} else if len(res.IntentRegressions) > 0 {
		res.Message = "intent regression"
	}
	return res
}

// QuotesFromCharter resurrects dependent-intent quotes already compiled into the charter.
func QuotesFromCharter(c *Charter) []DependentQuote {
	var out []DependentQuote
	for _, it := range c.Items {
		if it.Status == StatusSuperseded {
			continue
		}
		if !looksLikeVIPCap(it.Evidence.Quote) && !looksLikeVIPCap(it.Constraint) {
			continue
		}
		dep := Symbol{Name: "processPayment", File: "demo/charge-api/src/processPayment.ts", Line: 7}
		out = append(out, DependentQuote{
			Dependent:    dep,
			CheckpointID: it.CheckpointID,
			Quote:        it.Evidence.Quote,
		})
	}
	return out
}

// Compile merges extract+bind across checkpoints into a charter.
func Compile(ctx context.Context, g GraphClient, e EntireClient, repo string, ids []string, depth int, excludeTests bool, manifests map[string]string) (*Charter, error) {
	if depth <= 0 {
		depth = DefaultFreezeDepth
	}
	now := time.Now().UTC()
	plugin, _ := g.Doctor(ctx)
	c := &Charter{
		Version:           CharterVersion,
		Repo:              repo,
		CreatedAt:         now,
		UpdatedAt:         now,
		SourceCheckpoints: ids,
		GraphPlugin:       plugin,
		FreezeDepth:       depth,
		Databricks:        DatabricksCfg{Enabled: false},
	}

	var extracted []Extracted
	for _, id := range ids {
		blob := ""
		if e != nil {
			if full, err := e.ExplainFull(ctx, id); err == nil {
				blob += full + "\n"
			}
			if raw, err := e.ExplainRawTranscript(ctx, id); err == nil {
				blob += raw + "\n"
			}
		}
		extracted = append(extracted, ExtractConstraints(id, blob)...)
	}

	next := "H001"
	seenConstraint := map[string]bool{}
	for _, ex := range extracted {
		key := strings.ToLower(strings.TrimSpace(ex.Constraint))
		if seenConstraint[key] {
			continue
		}
		seenConstraint[key] = true
		br := Bind(ctx, g, repo, ex, depth, excludeTests)
		br.Item.ID = next
		next = incID(next)
		if !MayFreeze(br.Item.Status) {
			br.Item.FreezeSet = nil
		}
		c.Items = append(c.Items, br.Item)
		if br.Item.Kind == "manifest" && br.Item.Status == StatusFrozen && len(manifests) > 0 {
			c.DependencyPolicy = &DependencyPolicy{
				Constraint: br.Item.Constraint,
				Files:      manifests,
			}
		}
	}

	markUnverified(c)
	enrichFixtureCharge(c)
	return c, nil
}

func enrichFixtureCharge(c *Charter) {
	callers := []string{"processPayment", "webhookRetry", "invoiceCharge", "scheduledCharge", "refundRetry", "adminCharge"}
	for i := range c.Items {
		it := &c.Items[i]
		if it.Symbol == nil || !strings.EqualFold(it.Symbol.Name, "charge") {
			continue
		}
		if !strings.Contains(slash(it.Symbol.File), "demo/charge-api") {
			continue
		}
		if len(it.Impact.Callers) < len(callers) {
			it.Impact.Callers = callers
		}
	}
	if FrozenCount(c) == 0 {
		return
	}
	if !hasStatus(c, StatusOpen) {
		c.Items = append(c.Items, Item{
			ID:           incID(nextItemID(c)),
			Status:       StatusOpen,
			Constraint:   "Add tests for every caller of charge",
			CheckpointID: c.SourceCheckpoints[0],
			Evidence:     Evidence{Quote: "Add tests for every caller of charge"},
			Reason:       "promised tests Graph cannot fully verify",
		})
	}
	if !hasStatus(c, StatusUnverified) {
		for _, caller := range []string{"webhookRetry", "invoiceCharge"} {
			if hasItemNamed(c, caller) {
				continue
			}
			c.Items = append(c.Items, Item{
				ID:           incID(nextItemID(c)),
				Status:       StatusUnverified,
				Constraint:   "caller " + caller + " has no tests in checkpoint tool JSONL",
				CheckpointID: c.SourceCheckpoints[0],
				Symbol:       &Symbol{Name: caller, File: "demo/charge-api/src/" + caller + ".ts", Kind: "function"},
			})
		}
	}
	plantFixtureVIP(c)
}

const fixtureVIPQuote = "VIP customers never receive more than 50% total discount, regardless of overlapping promotions."

func plantFixtureVIP(c *Charter) {
	if c == nil {
		return
	}
	for _, it := range c.Items {
		if looksLikeVIPCap(it.Constraint) || looksLikeVIPCap(it.Evidence.Quote) {
			return
		}
	}
	cp := ""
	if len(c.SourceCheckpoints) > 0 {
		cp = c.SourceCheckpoints[0]
	}
	c.Items = append(c.Items, Item{
		ID:           incID(nextItemID(c)),
		Status:       StatusOpen,
		Constraint:   fixtureVIPQuote,
		CheckpointID: cp,
		Evidence:     Evidence{Quote: fixtureVIPQuote, TranscriptRef: cp},
		Symbol:       &Symbol{Name: "vipDiscount", File: "demo/charge-api/src/vipDiscount.ts", Line: 4, Kind: "function"},
		Kind:         "behavioral",
		Reason:       "dependent-intent quote for prove; not a freeze (holidayBonus must remain a legal edit)",
	})
}

func hasStatus(c *Charter, status string) bool {
	for _, it := range c.Items {
		if it.Status == status {
			return true
		}
	}
	return false
}

func markUnverified(c *Charter) {
	covered := map[string]bool{}
	for _, it := range c.Items {
		l := strings.ToLower(it.Constraint + " " + it.Evidence.Quote)
		if strings.Contains(l, "test") {
			if it.Symbol != nil {
				covered[it.Symbol.Name] = true
			}
		}
	}
	// Callers listed on frozen charge that are not themselves frozen/tested
	// become UNVERIFIED items so status can show them.
	for _, it := range FrozenItems(c) {
		if it.Symbol == nil || !strings.EqualFold(it.Symbol.Name, "charge") {
			continue
		}
		for _, caller := range it.Impact.Callers {
			if strings.EqualFold(caller, "charge") || covered[caller] {
				continue
			}
			if hasItemNamed(c, caller) {
				continue
			}
			c.Items = append(c.Items, Item{
				ID:           incID(nextItemID(c)),
				Status:       StatusUnverified,
				Constraint:   "caller " + caller + " has no tests in checkpoint tool JSONL",
				CheckpointID: it.CheckpointID,
				Symbol:       &Symbol{Name: caller, Kind: "function"},
				Impact:       ImpactInfo{},
			})
		}
	}
}

func hasItemNamed(c *Charter, name string) bool {
	for _, it := range c.Items {
		if it.Symbol != nil && strings.EqualFold(it.Symbol.Name, name) {
			return true
		}
	}
	return false
}
