package hold

import (
	"encoding/json"
	"regexp"
	"strings"
)

var constraintLineRe = regexp.MustCompile(`(?i)(do not|don't|does not|must not|must stay|never|keep .{0,60}idempotent|idempotent|no new (runtime )?(dep|dependency|dependencies)|never (more|exceed)|without changing|regardless of overlapping).{0,240}`)

var identTickRe = regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_]*)`")
var identCallRe = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\(\)`)

// ExtractConstraints pulls gate sentences from checkpoint explain text and/or
// raw JSONL. Local regex only — never POST prompts or transcripts off-machine.
// Prefer UNBOUND later over a hallucinated freeze.
func ExtractConstraints(checkpointID, blob string) []Extracted {
	if strings.TrimSpace(blob) == "" {
		return nil
	}
	text := blobToPlain(blob)
	seen := map[string]bool{}
	var out []Extracted
	add := func(sentence, quote string) {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" || !isHoldSignal(sentence) {
			return
		}
		key := strings.ToLower(sentence)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, Extracted{
			Constraint:    sentence,
			Quote:         quoteOrSelf(quote, sentence),
			CheckpointID:  checkpointID,
			TranscriptRef: checkpointID,
			SymbolGuess:   GuessSymbol(sentence),
		})
	}

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "-*•"))
		if line == "" {
			continue
		}
		if constraintLineRe.MatchString(line) {
			add(clipSentence(line), line)
		}
	}
	for _, line := range firstPromptLines(text) {
		if constraintLineRe.MatchString(line) {
			add(clipSentence(line), line)
		}
	}
	return out
}

func isHoldSignal(sentence string) bool {
	l := strings.ToLower(sentence)
	if strings.Contains(l, "do not redo") || strings.Contains(l, "do not research") || strings.Contains(l, "do not invent") || strings.Contains(l, "do not skip") || strings.Contains(l, "do not fake") {
		return false
	}
	if isPrivacyBoundaryText(sentence) {
		return true
	}
	switch {
	case strings.Contains(l, "charge"):
		return true
	case strings.Contains(l, "idempotent") || strings.Contains(l, "retry"):
		return true
	case strings.Contains(l, "dependenc"):
		return true
	case strings.Contains(l, "vip") && strings.Contains(l, "discount"):
		return true
	case strings.Contains(l, "caller") && strings.Contains(l, "test"):
		return true
	default:
		return false
	}
}

// GuessSymbol picks a likely identifier from a constraint sentence.
func GuessSymbol(sentence string) string {
	if isPrivacyBoundaryText(sentence) {
		return ""
	}
	if m := identTickRe.FindStringSubmatch(sentence); len(m) > 1 {
		return m[1]
	}
	if m := identCallRe.FindStringSubmatch(sentence); len(m) > 1 {
		return m[1]
	}
	lower := strings.ToLower(sentence)
	switch {
	case strings.Contains(lower, "charge"):
		return "charge"
	case strings.Contains(lower, "retry"):
		return "retryCharge"
	case strings.Contains(lower, "idempotent"):
		return "retryCharge"
	case strings.Contains(lower, "dependenc"):
		return "package.json"
	case strings.Contains(lower, "vip"):
		return "vipDiscount"
	}
	return ""
}

func quoteOrSelf(quote, sentence string) string {
	quote = strings.TrimSpace(quote)
	if quote == "" {
		return sentence
	}
	if len(quote) > 240 {
		return quote[:237] + "..."
	}
	return quote
}

func clipSentence(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 280 {
		s = s[:277] + "..."
	}
	return s
}

func blobToPlain(blob string) string {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return ""
	}
	// JSONL transcript: pull string fields that look like prompts.
	if strings.Contains(blob, `"role"`) || strings.Contains(blob, `"prompt"`) || strings.HasPrefix(blob, "{") {
		var b strings.Builder
		for _, line := range strings.Split(blob, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var m map[string]any
			if json.Unmarshal([]byte(line), &m) != nil {
				b.WriteString(line)
				b.WriteByte('\n')
				continue
			}
			b.WriteString(flattenPrompt(m))
			b.WriteByte('\n')
		}
		if b.Len() > 0 {
			return b.String()
		}
	}
	return blob
}

func flattenPrompt(m map[string]any) string {
	for _, k := range []string{"prompt", "text", "content", "message"} {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	if role, _ := m["role"].(string); strings.EqualFold(role, "user") {
		if s, ok := m["content"].(string); ok {
			return s
		}
	}
	return ""
}

// ContainsRedaction reports Entire's replacement tokens (REDACTED,
// [REDACTED_LABEL]). It does not match the English word "redacted".
func ContainsRedaction(s string) bool {
	if strings.Contains(s, "[REDACTED") {
		return true
	}
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], "REDACTED")
		if j < 0 {
			return false
		}
		j += i
		beforeOK := j == 0 || !isIdentByte(s[j-1])
		after := j + len("REDACTED")
		afterOK := after >= len(s) || !isIdentByte(s[after])
		if beforeOK && afterOK {
			return true
		}
		i = j + 1
	}
	return false
}

func isIdentByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}

// HolePreventsFreeze is true when a quote/guess is a redacted hole or empty.
// Those items stay UNBOUND — never FROZEN.
func HolePreventsFreeze(ex Extracted) bool {
	return ContainsRedaction(ex.Constraint) || ContainsRedaction(ex.Quote) || ContainsRedaction(ex.SymbolGuess)
}

// ClassifyCheckpointBlob is missing / redacted / ok. Empty text is missing.
func ClassifyCheckpointBlob(blob string) string {
	if strings.TrimSpace(blob) == "" {
		return "missing"
	}
	if ContainsRedaction(blob) {
		return "redacted"
	}
	return "ok"
}

func isPrivacyBoundaryText(s string) bool {
	l := strings.ToLower(s)
	if !(strings.Contains(l, "transcript") || strings.Contains(l, "prompt") || strings.Contains(l, "external service")) {
		return false
	}
	return strings.Contains(l, "must not") || strings.Contains(l, "must not be sent") || strings.Contains(l, "redact") || strings.Contains(l, "incomplete") || strings.Contains(l, "unbound")
}

func firstPromptLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "-*•"))
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) > 40 {
			break
		}
	}
	return lines
}
