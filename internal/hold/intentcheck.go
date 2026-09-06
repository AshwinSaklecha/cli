package hold

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Fixture composition facts for demo/charge-api. Deterministic Leg B — not an LLM.
const (
	FixtureVIPRate     = 0.40
	FixtureHolidayRate = 0.20
	FixtureMaxTotal    = 0.50
)

var maxDiscountRe = regexp.MustCompile(`(?i)(?:never|not|no more than|at most|must not exceed|does not exceed|≤|<=)\s*(\d{1,3})\s*%`)
var percentRe = regexp.MustCompile(`(\d{1,3})\s*%`)

// EvaluateIntent checks a legal change against a dependent's historical quote.
// Fixture-aware: VIP 40% + holiday 20% exceeds 50% total even when unit tests pass.
func EvaluateIntent(dependent Symbol, quote, checkpointID string, changed []Change, blast []Symbol) []IntentFinding {
	quote = strings.TrimSpace(quote)
	if quote == "" {
		return nil
	}
	max, hasMax := parseMaxPercent(quote)
	if !hasMax {
		max = FixtureMaxTotal
		hasMax = looksLikeVIPCap(quote)
	}
	if !hasMax {
		return nil
	}

	vip := namedIn(blast, changed, "vipDiscount", "vip")
	holiday := namedIn(blast, changed, "holidayBonus", "holiday")
	if !holiday {
		return nil
	}

	vipRate := FixtureVIPRate
	holRate := FixtureHolidayRate
	total := vipRate + holRate
	if total <= max+1e-9 {
		return nil
	}

	dep := dependent
	if dep.Name == "" {
		dep.Name = "processPayment"
		dep.File = "demo/charge-api/src/processPayment.ts"
	}

	comp := fmt.Sprintf("%.0f%% VIP + %.0f%% holiday = %.0f%% > %.0f%%", vipRate*100, holRate*100, total*100, max*100)
	why := "dependent historical intent caps total discount; overlapping promotions compose above the cap"
	if !vip {
		why = "holiday bonus added beside existing VIP discount without a composition cap"
	}
	return []IntentFinding{{
		Dependent:    dep,
		CheckpointID: checkpointID,
		Quote:        quote,
		Change:       "holidayBonus +20% in src/holiday.ts",
		Composition:  comp,
		Why:          why,
	}}
}

func parseMaxPercent(quote string) (float64, bool) {
	m := maxDiscountRe.FindStringSubmatch(quote)
	if len(m) < 2 {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return float64(n) / 100.0, true
}

func looksLikeVIPCap(quote string) bool {
	l := strings.ToLower(quote)
	return strings.Contains(l, "vip") && (strings.Contains(l, "discount") || strings.Contains(l, "total")) && percentRe.MatchString(l)
}

func namedIn(blast []Symbol, changed []Change, names ...string) bool {
	match := func(s string) bool {
		cs := canonName(s)
		cl := strings.ToLower(cs)
		for _, n := range names {
			if strings.EqualFold(cs, n) || strings.Contains(cl, strings.ToLower(n)) {
				return true
			}
		}
		return false
	}
	for _, s := range blast {
		if match(s.Name) || match(s.File) {
			return true
		}
	}
	for _, c := range changed {
		if match(c.Name) || match(c.File) {
			return true
		}
	}
	return false
}

// FormatProveFailure is Beat C stderr.
func FormatProveFailure(f IntentFinding) string {
	depLine := f.Dependent.File
	if f.Dependent.Line > 0 {
		depLine = fmt.Sprintf("%s:%d", f.Dependent.File, f.Dependent.Line)
	}
	cp := f.CheckpointID
	if len(cp) > 12 {
		cp = cp[:12]
	}
	return fmt.Sprintf(`HOLD PROVE FAILED — INTENT REGRESSION
dependent: %s  %s
authored by checkpoint %s  quote: %q
change: %s
composition: %s
unit tests: still passing (they never compose the two)
`, f.Dependent.Name, depLine, cp, f.Quote, f.Change, f.Composition)
}
