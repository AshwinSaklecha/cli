package hold

import (
	"encoding/json"
	"strings"
)

// ParseGraphSearchSymbols pulls candidate symbols from `entire graph search --format json`.
func ParseGraphSearchSymbols(raw []byte) []Symbol {
	return collectSymbols(decodeJSON(raw))
}

// ParseGraphDefSymbols pulls symbols from `entire graph def`.
func ParseGraphDefSymbols(raw []byte) []Symbol {
	return collectSymbols(decodeJSON(raw))
}

// ParseGraphImpact parses `entire graph impact --format json`.
func ParseGraphImpact(raw []byte) ImpactResult {
	root := decodeJSON(raw)
	syms := collectSymbols(root)
	callers := uniqueNames(syms)
	return ImpactResult{Callers: callers, Neighborhood: toFreeze(syms)}
}

// ParseGraphDiffChanges parses `entire graph diff --json`.
func ParseGraphDiffChanges(raw []byte) []Change {
	root := decodeJSON(raw)
	return collectChanges(root)
}

// ParseGraphDoctorOK reports doctor_ok / ok from `entire graph doctor --json`.
func ParseGraphDoctorOK(raw []byte) (version string, ok bool) {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return "", false
	}
	version, _ = asString(m["version"])
	if version == "" {
		version, _ = asString(m["plugin_version"])
	}
	if version == "" {
		if p, ok := m["provider"].(string); ok {
			version = p
		}
	}
	if b, ok := m["ok"].(bool); ok {
		return version, b
	}
	if b, ok := m["doctor_ok"].(bool); ok {
		return version, b
	}
	if st, ok := asString(m["status"]); ok {
		return version, strings.EqualFold(st, "ok") || strings.EqualFold(st, "healthy")
	}
	return version, true
}

// ImpactResult is the freeze neighborhood.
type ImpactResult struct {
	Callers      []string
	Neighborhood []FreezeEntry
}

func decodeJSON(raw []byte) any {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return nil
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	return v
}

func collectSymbols(v any) []Symbol {
	var out []Symbol
	walk(v, func(m map[string]any) {
		name, _ := firstString(m, "symbol_name", "name", "symbol", "qualified_name")
		file, _ := firstString(m, "file_path", "file", "path")
		if name == "" && file == "" {
			return
		}
		line := firstInt(m, "start_line", "line", "focus_line", "symbol_start_line")
		kind, _ := firstString(m, "kind", "change", "status")
		if name == "" {
			return
		}
		out = append(out, Symbol{Name: name, File: file, Line: line, Kind: kind})
	})
	return dedupeSymbols(out)
}

func collectChanges(v any) []Change {
	var out []Change
	walk(v, func(m map[string]any) {
		name, _ := firstString(m, "symbol_name", "name", "symbol", "qualified_name")
		file, _ := firstString(m, "file_path", "file", "path")
		if name == "" && file == "" {
			return
		}
		kind, _ := firstString(m, "change", "change_kind", "status", "kind")
		if kind == "" {
			kind = "body-changed"
		}
		line := firstInt(m, "start_line", "line", "focus_line")
		if name == "" {
			name = file
		}
		out = append(out, Change{Name: name, File: file, Line: line, Kind: kind})
	})
	return out
}

func walk(v any, fn func(map[string]any)) {
	switch t := v.(type) {
	case map[string]any:
		fn(t)
		for _, child := range t {
			walk(child, fn)
		}
	case []any:
		for _, child := range t {
			walk(child, fn)
		}
	}
}

func firstString(m map[string]any, keys ...string) (string, bool) {
	for _, k := range keys {
		if s, ok := asString(m[k]); ok && s != "" {
			return s, true
		}
	}
	return "", false
}

func firstInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		switch n := m[k].(type) {
		case float64:
			return int(n)
		case int:
			return n
		case json.Number:
			i, _ := n.Int64()
			return int(i)
		}
	}
	return 0
}

func asString(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func uniqueNames(syms []Symbol) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range syms {
		if s.Name == "" || seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		out = append(out, s.Name)
	}
	return out
}

func toFreeze(syms []Symbol) []FreezeEntry {
	out := make([]FreezeEntry, 0, len(syms))
	seen := map[string]bool{}
	for _, s := range syms {
		key := s.File + "\x00" + s.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, FreezeEntry{Name: s.Name, File: s.File, Line: s.Line, Kind: s.Kind})
	}
	return out
}

func dedupeSymbols(in []Symbol) []Symbol {
	seen := map[string]bool{}
	var out []Symbol
	for _, s := range in {
		key := s.File + "\x00" + s.Name + "\x00" + itoa(s.Line)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
