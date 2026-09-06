package hold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// MarshalCharter is pretty JSON for charter.json.
func MarshalCharter(c *Charter) ([]byte, error) {
	if c == nil {
		return nil, ErrNoCharter
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	return b, nil
}

// ParseCharter loads charter.json.
func ParseCharter(data []byte) (*Charter, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, ErrNoCharter
	}
	var c Charter
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse charter: %w", err)
	}
	if c.Version == "" {
		c.Version = CharterVersion
	}
	return &c, nil
}

// RenderPermit is briefing only — check never parses this file.
func RenderPermit(c *Charter) string {
	if c == nil {
		return "# Hold permit\n\nNo charter. Run `entire hold compile`.\n"
	}
	var b strings.Builder
	b.WriteString("# Hold continuation permit\n\n")
	b.WriteString("Briefing for the next agent. **Not enforced.** Enforcement is `entire hold check` (exit 1).\n\n")
	b.WriteString("## Frozen\n\n")
	b.WriteString("| id | symbol | file:line | constraint | checkpoint |\n")
	b.WriteString("|----|--------|-----------|------------|------------|\n")
	for _, it := range c.Items {
		if it.Status != StatusFrozen && it.Status != StatusConflict {
			continue
		}
		sym, loc := "-", "-"
		if it.Symbol != nil {
			sym = it.Symbol.Name
			loc = fmt.Sprintf("%s:%d", it.Symbol.File, it.Symbol.Line)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", it.ID, sym, loc, clip(it.Constraint, 80), shortID(it.CheckpointID))
	}
	b.WriteString("\n## OPEN\n\n")
	writeStatusList(&b, c, StatusOpen)
	b.WriteString("\n## UNVERIFIED\n\n")
	writeStatusList(&b, c, StatusUnverified)
	if c.LastAmend != "" {
		fmt.Fprintf(&b, "\n## Last amend\n\n%s\n", c.LastAmend)
		if c.LastAmendReason != "" {
			fmt.Fprintf(&b, "Reason: %s\n", c.LastAmendReason)
		}
	}
	b.WriteString("\n## How to check\n\n```\nentire hold check\nentire hold prove --test \"npm test\"\n```\n\n`HOLD_BYPASS=1` skips the pre-commit hook. Do not use it in the demo.\n")
	fmt.Fprintf(&b, "\n_Updated %s_\n", c.UpdatedAt.UTC().Format(time.RFC3339))
	return b.String()
}

func writeStatusList(b *strings.Builder, c *Charter, status string) {
	n := 0
	for _, it := range c.Items {
		if it.Status != status {
			continue
		}
		fmt.Fprintf(b, "- `%s` %s\n", it.ID, clip(it.Constraint, 100))
		n++
	}
	if n == 0 {
		b.WriteString("(none)\n")
	}
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "|", "/")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// FrozenCount is used by compile's exit-2 gate.
func FrozenCount(c *Charter) int {
	n := 0
	for _, it := range c.Items {
		if it.Status == StatusFrozen {
			n++
		}
	}
	return n
}

// PreCommitScript is a POSIX hook Git for Windows runs via sh. It does not
// replace Entire's prepare-commit-msg / post-commit / pre-push hooks.
func PreCommitScript(entireBin string) string {
	if strings.TrimSpace(entireBin) == "" {
		entireBin = "entire"
	}
	// Quote for sh. Windows paths stay as Git Bash can exec them.
	quoted := "'" + strings.ReplaceAll(entireBin, "'", `'\''`) + "'"
	return `#!/bin/sh
# ` + HoldHookMarker + `
# Runs entire hold check. Entire's own git hooks are a different set
# (prepare-commit-msg, commit-msg, post-commit, post-rewrite, pre-push).
if [ -n "$` + BypassEnv + `" ]; then
  exit 0
fi
if [ -f ` + quoted + ` ] || command -v entire >/dev/null 2>&1; then
  if [ -f ` + quoted + ` ]; then
    ` + quoted + ` hold check
  else
    entire hold check
  fi
  status=$?
  if [ "$status" -ne 0 ]; then
    exit "$status"
  fi
fi
if [ -x "$0.pre-hold" ]; then
  "$0.pre-hold" "$@"
fi
exit 0
`
}
