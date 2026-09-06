package hold

import (
	"fmt"
	"path"
	"strings"
)

// ErrNoCharter is compile/check/prove when charter.json is missing.
var ErrNoCharter = fmt.Errorf("hold charter not found")

// RunCheck intersects graph diff changes with freeze_set. No LLM.
func RunCheck(c *Charter, changes []Change, currentManifests map[string]string) CheckResult {
	res := CheckResult{Passed: true}
	if c == nil {
		res.Passed = false
		res.Message = "missing charter"
		return res
	}
	for _, it := range FrozenItems(c) {
		for _, ch := range changes {
			if hit, fe := intersects(it, ch); hit {
				res.Passed = false
				res.Intersections = append(res.Intersections, Intersection{
					Frozen:       fe,
					Changed:      ch,
					ItemID:       it.ID,
					Callers:      it.Impact.Callers,
					Quote:        it.Evidence.Quote,
					CheckpointID: it.CheckpointID,
				})
			}
		}
	}
	if c.DependencyPolicy != nil {
		for file, want := range c.DependencyPolicy.Files {
			got := currentManifests[file]
			if got == "" {
				got = currentManifests[slash(file)]
			}
			if got != "" && !strings.EqualFold(got, want) {
				res.Passed = false
				res.ManifestBreak = append(res.ManifestBreak, file)
			}
		}
	}
	return res
}

func intersects(it Item, ch Change) (bool, FreezeEntry) {
	for _, fe := range it.FreezeSet {
		if matchFreeze(fe, ch) {
			return true, fe
		}
	}
	if it.Symbol != nil {
		fe := FreezeEntry{Name: it.Symbol.Name, File: it.Symbol.File, Line: it.Symbol.Line, Kind: it.Symbol.Kind}
		if matchFreeze(fe, ch) {
			return true, fe
		}
	}
	return false, FreezeEntry{}
}

func matchFreeze(fe FreezeEntry, ch Change) bool {
	fn, cn := canonName(fe.Name), canonName(ch.Name)
	ff, cf := slash(fe.File), slash(ch.File)
	if fn != "" && cn != "" && fn == cn {
		if ff == "" || cf == "" || ff == cf || strings.HasSuffix(cf, "/"+path.Base(ff)) || strings.HasSuffix(ff, "/"+path.Base(cf)) {
			return true
		}
	}
	if ff != "" && cf != "" && (ff == cf || strings.HasSuffix(cf, "/"+path.Base(ff))) {
		if fe.Line > 0 && ch.Line > 0 {
			d := fe.Line - ch.Line
			if d < 0 {
				d = -d
			}
			if d <= 8 {
				return true
			}
		}
	}
	return false
}

func canonName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "()")
	if i := strings.LastIndex(s, "."); i >= 0 && i < len(s)-1 {
		s = s[i+1:]
	}
	return s
}

func slash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// FormatCheckFailure is the Beat A stderr. AGENTS.md cannot compute this.
func FormatCheckFailure(c *Charter, res CheckResult) string {
	var b strings.Builder
	b.WriteString("HOLD CHECK FAILED\n")
	if len(res.Intersections) > 0 {
		x := res.Intersections[0]
		frozenLine := fmt.Sprintf("%s:%d", x.Frozen.File, x.Frozen.Line)
		changedLine := fmt.Sprintf("%s:%d", x.Changed.File, x.Changed.Line)
		cp := x.CheckpointID
		if len(cp) > 12 {
			cp = cp[:12]
		}
		fmt.Fprintf(&b, "frozen:  %s  %s  (checkpoint %s  %q)\n", x.Frozen.Name, frozenLine, cp, x.Quote)
		fmt.Fprintf(&b, "changed: %s  %s  (%s)\n", x.Changed.Name, changedLine, orKind(x.Changed.Kind))
		callers := x.Callers
		if len(callers) == 0 && c != nil {
			for _, it := range c.Items {
				if it.ID == x.ItemID {
					callers = it.Impact.Callers
				}
			}
		}
		fmt.Fprintf(&b, "graph impact: %d callers", len(callers))
		if len(callers) > 0 {
			b.WriteString(" — ")
			b.WriteString(strings.Join(limit(callers, 8), ", "))
		}
		b.WriteByte('\n')
	}
	for _, m := range res.ManifestBreak {
		fmt.Fprintf(&b, "dependency_policy: %s hash changed\n", m)
	}
	b.WriteString("A static AGENTS.md cannot compute this intersection.\n")
	return b.String()
}

func orKind(k string) string {
	if k == "" {
		return "body-changed"
	}
	return k
}

func limit(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
