package hold

import (
	"context"
	"strings"
	"time"
)

// AmendInput is the noon path.
type AmendInput struct {
	Text         string
	Supersede    string
	Reason       string
	Checkpoint   string
	Depth        int
	ExcludeTests bool
}

// AmendResult is printed by `entire hold amend`.
type AmendResult struct {
	Conflicts  []string `json:"conflicts,omitempty"`
	Superseded []string `json:"superseded,omitempty"`
	NewFrozen  []string `json:"new_frozen,omitempty"`
	NewUnbound []string `json:"new_unbound,omitempty"`
	Charter    *Charter `json:"-"`
}

// Amend binds a new sentence onto the charter. Overlap + contradiction →
// CONFLICT (freeze still enforces). --supersede retires a freeze.
func Amend(ctx context.Context, g GraphClient, repo string, c *Charter, in AmendInput) (AmendResult, error) {
	var out AmendResult
	if c == nil {
		return out, ErrNoCharter
	}
	if in.Depth <= 0 {
		in.Depth = DefaultFreezeDepth
	}
	now := time.Now().UTC()
	exs := ExtractConstraints(in.Checkpoint, in.Text)
	if len(exs) == 0 {
		exs = []Extracted{{
			Constraint:   strings.TrimSpace(in.Text),
			Quote:        strings.TrimSpace(in.Text),
			CheckpointID: in.Checkpoint,
			SymbolGuess:  GuessSymbol(in.Text),
		}}
	}

	if in.Supersede != "" {
		for i := range c.Items {
			if c.Items[i].ID == in.Supersede || strings.HasPrefix(c.Items[i].ID, in.Supersede) {
				c.Items[i].Status = StatusSuperseded
				c.Items[i].Reason = in.Reason
				out.Superseded = append(out.Superseded, c.Items[i].ID)
			}
		}
	}

	nextID := nextItemID(c)
	for _, ex := range exs {
		br := Bind(ctx, g, repo, ex, in.Depth, in.ExcludeTests)
		br.Item.ID = nextID
		nextID = incID(nextID)

		if in.Supersede != "" {
			br.Item.Supersedes = in.Supersede
			br.Item.Reason = in.Reason
		}

		if br.Item.Status == StatusFrozen && in.Supersede == "" {
			if conflict := conflictingFreeze(c, br.Item); conflict != "" {
				br.Item.Status = StatusConflict
				br.Item.Reason = "fights still-binding freeze " + conflict
				out.Conflicts = append(out.Conflicts, br.Item.ID+" vs "+conflict)
			}
		}

		c.Items = append(c.Items, br.Item)
		switch br.Item.Status {
		case StatusFrozen:
			out.NewFrozen = append(out.NewFrozen, br.Item.ID)
		case StatusUnbound, StatusOpen:
			out.NewUnbound = append(out.NewUnbound, br.Item.ID)
		case StatusConflict:
			out.Conflicts = append(out.Conflicts, br.Item.ID)
		}
	}

	c.UpdatedAt = now
	c.LastAmend = strings.TrimSpace(in.Text)
	c.LastAmendReason = in.Reason
	out.Charter = c
	return out, nil
}

func conflictingFreeze(c *Charter, neu Item) string {
	if neu.Symbol == nil {
		return ""
	}
	for _, it := range FrozenItems(c) {
		if it.ID == neu.ID {
			continue
		}
		if it.Symbol != nil && canonName(it.Symbol.Name) == canonName(neu.Symbol.Name) {
			if contradicts(it.Constraint, neu.Constraint) {
				return it.ID
			}
		}
	}
	return ""
}

func contradicts(a, b string) bool {
	al, bl := strings.ToLower(a), strings.ToLower(b)
	forbid := strings.Contains(al, "do not") || strings.Contains(al, "don't") || strings.Contains(al, "must not") || strings.Contains(al, "never")
	allow := strings.Contains(bl, "allow") || strings.Contains(bl, "may ") || strings.Contains(bl, "add a") || strings.Contains(bl, "new parameter")
	return forbid && allow
}

func nextItemID(c *Charter) string {
	n := 1
	for _, it := range c.Items {
		if strings.HasPrefix(it.ID, "H") {
			n++
		}
	}
	return "H" + pad3(n)
}

func incID(id string) string {
	n := 0
	for _, r := range id {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return "H" + pad3(n+1)
}

func pad3(n int) string {
	s := itoa(n)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}
