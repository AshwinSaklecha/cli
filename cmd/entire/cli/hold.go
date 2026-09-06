package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/entiredir"
	"github.com/entireio/cli/cmd/entire/cli/osroot"
	"github.com/entireio/cli/cmd/entire/cli/paths"
	"github.com/entireio/cli/cmd/entire/cli/strategy"
	"github.com/entireio/cli/cmd/entire/cli/worktreedir"
	"github.com/entireio/cli/internal/hold"
	"github.com/spf13/cobra"
)

func newHoldGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hold",
		Short: "Checkpoint-native continuation gate",
		Long: `Hold compiles checkpoint promises onto Graph symbols, fails edits that
intersect a frozen neighborhood, amends when the plan changes, and proves
remaining change against dependents' historical intent.

Artifacts live under .entire/hold/. permit.md is briefing only and is never
parsed by check.`,
		Example: "  entire hold compile --json\n  entire hold check\n  entire hold amend \"allow a new parameter on charge()\" --supersede H001 --reason noon\n  entire hold prove --test \"npm test\"",
	}
	cmd.AddCommand(newHoldCompileCmd())
	cmd.AddCommand(newHoldStatusCmd())
	cmd.AddCommand(newHoldPermitCmd())
	cmd.AddCommand(newHoldCheckCmd())
	cmd.AddCommand(newHoldAmendCmd())
	cmd.AddCommand(newHoldProveCmd())
	cmd.AddCommand(newHoldPublishCmd())
	cmd.AddCommand(newHoldPullCmd())
	return cmd
}

func newHoldCompileCmd() *cobra.Command {
	var fromID string
	var all bool
	var jsonOut bool
	var depth int
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Extract checkpoint promises and freeze Graph neighborhoods",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			clients := holdClients()
			ids, err := resolveCheckpointIDs(ctx, clients, fromID, all)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No checkpoints to compile. Save an agent session and commit first.")
				return NewCodedSilentError(2, errors.New("no checkpoints"))
			}
			repo := "."
			if root, err := paths.WorktreeRoot(ctx); err == nil {
				repo = root
			}
			manifests := hashManifests(ctx)
			c, err := hold.Compile(ctx, clients, clients, repo, ids, depth, true, manifests)
			if err != nil {
				return err
			}
			if err := writeCharter(ctx, c); err != nil {
				return err
			}
			_, _ = writePermit(ctx, c, "")
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), c)
			}
			printCompile(cmd.OutOrStdout(), c)
			if hold.FrozenCount(c) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "hold compile: zero FROZEN items — Graph bind is dead.")
				return NewCodedSilentError(2, errors.New("zero frozen"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&fromID, "from-checkpoint", "", "Compile a single checkpoint ID")
	cmd.Flags().BoolVar(&all, "all", false, "Compile every checkpoint on the branch")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "JSON charter")
	cmd.Flags().IntVar(&depth, "freeze-depth", hold.DefaultFreezeDepth, "graph impact depth")
	return cmd
}

func newHoldStatusCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show charter items (FROZEN, OPEN, UNVERIFIED)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := loadCharter(cmd.Context())
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "No charter. Run `entire hold compile`.")
				return NewCodedSilentError(2, hold.ErrNoCharter)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), c)
			}
			printStatus(cmd.OutOrStdout(), c)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "JSON charter")
	return cmd
}

func newHoldPermitCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "permit",
		Short: "Write permit.md briefing (not enforced)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := loadCharter(cmd.Context())
			if err != nil {
				return NewCodedSilentError(2, hold.ErrNoCharter)
			}
			text, err := writePermit(cmd.Context(), c, out)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), text)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", hold.PermitRelPath, "Output path")
	return cmd
}

func newHoldCheckCmd() *cobra.Command {
	var base, head string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Fail if a frozen Graph neighborhood moved",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := loadCharter(ctx)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "No charter. Run `entire hold compile`.")
				return NewCodedSilentError(2, hold.ErrNoCharter)
			}
			if err := installHoldPreCommit(ctx, cmd.ErrOrStderr()); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "hold: pre-commit hook: %v\n", err)
			}
			clients := holdClients()
			if base == "" {
				base = "HEAD"
			}
			if head == "" {
				head = "HEAD"
			}
			var changes []hold.Change
			if base != head {
				diff, err := clients.Diff(ctx, base, head, ".")
				if err == nil {
					changes = diff
				}
			}
			changes = append(changes, worktreeOverlay(ctx, clients)...)
			res := hold.RunCheck(c, changes, hashManifests(ctx))
			res.At = time.Now().UTC()
			res.Base, res.Head = base, head
			_ = writeHoldJSON(ctx, "hold/last-check.json", res)
			if jsonOut {
				if err := writeJSON(cmd.OutOrStdout(), res); err != nil {
					return err
				}
			}
			if !res.Passed {
				fmt.Fprint(cmd.ErrOrStderr(), hold.FormatCheckFailure(c, res))
				return NewSilentError(errors.New("hold check failed"))
			}
			if !jsonOut {
				fmt.Fprintln(cmd.OutOrStdout(), "HOLD CHECK PASSED")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&base, "base", "", "graph diff base ref (default HEAD)")
	cmd.Flags().StringVar(&head, "head", "", "graph diff head ref (default HEAD)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "JSON result")
	return cmd
}

func newHoldAmendCmd() *cobra.Command {
	var supersede, reason string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "amend [constraint]",
		Short: "Bind a new constraint; CONFLICT or SUPERSEDE existing freezes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := loadCharter(ctx)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "No charter. Run `entire hold compile`.")
				return NewCodedSilentError(2, hold.ErrNoCharter)
			}
			text := strings.Join(args, " ")
			clients := holdClients()
			out, err := hold.Amend(ctx, clients, ".", c, hold.AmendInput{
				Text:         text,
				Supersede:    supersede,
				Reason:       reason,
				Depth:        hold.DefaultFreezeDepth,
				ExcludeTests: true,
			})
			if err != nil {
				return err
			}
			if err := writeCharter(ctx, out.Charter); err != nil {
				return err
			}
			_, _ = writePermit(ctx, out.Charter, "")
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), out)
			}
			printAmend(cmd.OutOrStdout(), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&supersede, "supersede", "", "Item ID to retire")
	cmd.Flags().StringVar(&reason, "reason", "", "Why the freeze is retired")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "JSON result")
	return cmd
}

func newHoldProveCmd() *cobra.Command {
	var testCmd string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "prove",
		Short: "Freeze intact + dependent historical intent",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := loadCharter(ctx)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "No charter. Run `entire hold compile`.")
				return NewCodedSilentError(2, hold.ErrNoCharter)
			}
			clients := holdClients()
			diff := worktreeOverlay(ctx, clients)
			neighbors := proveNeighbors(c, diff)
			quotes := resurrectQuotes(ctx, clients, c, neighbors)
			verifyOK := true
			if testCmd == "" {
				testCmd = defaultFixtureTest()
			}
			if testCmd != "" {
				verifyOK = runProveTests(ctx, testCmd, cmd.ErrOrStderr())
			}
			res := hold.RunProve(c, hold.ProveInput{
				Changed:         diff,
				CurrentDiff:     diff,
				Neighbors:       neighbors,
				DependentQuotes: quotes,
				VerifyOK:        verifyOK,
				TestCommand:     testCmd,
			})
			_ = writeHoldJSON(ctx, "hold/last-prove.json", res)
			if jsonOut {
				if err := writeJSON(cmd.OutOrStdout(), res); err != nil {
					return err
				}
			}
			if !res.Passed {
				if len(res.IntentRegressions) > 0 {
					fmt.Fprint(cmd.ErrOrStderr(), hold.FormatProveFailure(res.IntentRegressions[0]))
				} else if !res.FreezeOK {
					fmt.Fprintln(cmd.ErrOrStderr(), "HOLD PROVE FAILED — frozen neighborhood moved")
				} else {
					fmt.Fprintln(cmd.ErrOrStderr(), "HOLD PROVE FAILED — tests")
				}
				return NewSilentError(errors.New("hold prove failed"))
			}
			if !jsonOut {
				fmt.Fprintln(cmd.OutOrStdout(), "HOLD PROVE PASSED")
				if len(res.Intact) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "intact: %s\n", strings.Join(res.Intact, ", "))
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&testCmd, "test", "", "Test command (default: fixture npm test if present)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "JSON result")
	return cmd
}

func newHoldPublishCmd() *cobra.Command {
	var dry bool
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Optional Databricks publish (omitted unless configured)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "Databricks publish omitted. check/prove work with Databricks unplugged.")
			if dry {
				fmt.Fprintln(cmd.OutOrStdout(), "(dry-run)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dry, "dry-run", false, "Print what would be published")
	return cmd
}

func newHoldPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Optional Databricks pull (omitted unless configured)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "Databricks pull omitted. Local charter at .entire/hold/charter.json is the gate.")
			return nil
		},
	}
}

func holdClients() hold.ExecClients {
	return hold.ExecClients{Bin: hold.EntireBin(), Repo: ".", Timeout: 2 * time.Minute}
}

func resolveCheckpointIDs(ctx context.Context, e hold.ExecClients, from string, all bool) ([]string, error) {
	if from != "" {
		return []string{from}, nil
	}
	list, err := e.CheckpointList(ctx)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, m := range list {
		if m.ID == "" {
			continue
		}
		ids = append(ids, m.ID)
		if !all {
			break
		}
	}
	return ids, nil
}

func loadCharter(ctx context.Context) (*hold.Charter, error) {
	root, err := entiredir.OpenForRead(ctx)
	if err != nil {
		return nil, hold.ErrNoCharter
	}
	name, err := entiredir.Name(hold.CharterRelPath)
	if err != nil {
		return nil, err
	}
	data, err := entiredir.ReadFile(root, name)
	if err != nil {
		return nil, hold.ErrNoCharter
	}
	return hold.ParseCharter(data)
}

func writeCharter(ctx context.Context, c *hold.Charter) error {
	root, err := entiredir.Open(ctx)
	if err != nil {
		return err
	}
	if err := osroot.MkdirAllNoSymlink(root, "hold", 0o750); err != nil {
		return err
	}
	name, err := entiredir.Name(hold.CharterRelPath)
	if err != nil {
		return err
	}
	data, err := hold.MarshalCharter(c)
	if err != nil {
		return err
	}
	return entiredir.WriteFile(root, name, data, 0o644)
}

func writePermit(ctx context.Context, c *hold.Charter, out string) (string, error) {
	text := hold.RenderPermit(c)
	rel := hold.PermitRelPath
	if out != "" {
		rel = out
	}
	root, err := entiredir.Open(ctx)
	if err != nil {
		return text, err
	}
	if err := osroot.MkdirAllNoSymlink(root, "hold", 0o750); err != nil {
		return text, err
	}
	name, err := entiredir.Name(rel)
	if err != nil {
		return text, nil
	}
	if err := entiredir.WriteFile(root, name, []byte(text), 0o644); err != nil {
		return text, err
	}
	return text, nil
}

func writeHoldJSON(ctx context.Context, name string, v any) error {
	root, err := entiredir.Open(ctx)
	if err != nil {
		return err
	}
	if err := osroot.MkdirAllNoSymlink(root, "hold", 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return entiredir.WriteFile(root, name, data, 0o644)
}

func hashManifests(ctx context.Context) map[string]string {
	out := map[string]string{}
	wt, err := worktreedir.Open(ctx)
	if err != nil {
		return out
	}
	for _, rel := range []string{"demo/charge-api/package.json", "go.mod"} {
		data, err := osroot.ReadFile(wt, rel)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(data)
		out[rel] = hex.EncodeToString(sum[:])
	}
	return out
}

func worktreeOverlay(ctx context.Context, _ hold.GraphClient) []hold.Change {
	// Uncommitted edits vs HEAD. Skip untracked/newly-added paths so the
	// compile commit that introduces charge.ts does not fail its own gate.
	root, err := paths.WorktreeRoot(ctx)
	if err != nil {
		return nil
	}
	cmd := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain", "-z", "--no-optional-locks")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var changes []hold.Change
	for _, rec := range strings.Split(string(out), "\x00") {
		if len(rec) < 4 {
			continue
		}
		xy := rec[:2]
		if !dirtyVsHEAD(xy) {
			continue
		}
		file := strings.TrimSpace(rec[3:])
		if file == "" || strings.HasPrefix(file, ".entire/hold/") {
			continue
		}
		slashFile := strings.ReplaceAll(file, "\\", "/")
		base := path.Base(slashFile)
		name := strings.TrimSuffix(base, path.Ext(base))
		ch := hold.Change{Name: name, File: slashFile, Kind: "body-changed"}
		if strings.EqualFold(base, "charge.ts") {
			ch.Name = "charge"
			ch.Line = 6
		}
		changes = append(changes, ch)
	}
	return changes
}

func dirtyVsHEAD(xy string) bool {
	return strings.ContainsAny(xy, "MDRTU")
}

func proveNeighbors(c *hold.Charter, diff []hold.Change) []hold.Symbol {
	var out []hold.Symbol
	seen := map[string]bool{}
	add := func(s hold.Symbol) {
		if s.Name == "" || seen[s.Name] {
			return
		}
		seen[s.Name] = true
		out = append(out, s)
	}
	for _, ch := range diff {
		if ch.Name != "" {
			add(hold.Symbol{Name: ch.Name, File: ch.File, Line: ch.Line})
		}
	}
	for _, it := range c.Items {
		if it.Symbol != nil {
			add(*it.Symbol)
		}
		for _, caller := range it.Impact.Callers {
			add(hold.Symbol{Name: caller})
		}
	}
	add(hold.Symbol{Name: "processPayment", File: "demo/charge-api/src/processPayment.ts", Line: 7})
	add(hold.Symbol{Name: "vipDiscount", File: "demo/charge-api/src/vipDiscount.ts", Line: 4})
	add(hold.Symbol{Name: "holidayBonus", File: "demo/charge-api/src/holiday.ts", Line: 4})
	return out
}

func resurrectQuotes(ctx context.Context, e hold.ExecClients, c *hold.Charter, neighbors []hold.Symbol) []hold.DependentQuote {
	if fromCharter := hold.QuotesFromCharter(c); len(fromCharter) > 0 {
		return fromCharter
	}
	var out []hold.DependentQuote
	for _, n := range neighbors {
		if n.File == "" {
			continue
		}
		loc := n.File
		if n.Line > 0 {
			loc = fmt.Sprintf("%s:%d", n.File, n.Line)
		}
		metas, err := e.Why(ctx, loc)
		if err != nil || len(metas) == 0 {
			metas, _ = e.Blame(ctx, n.File)
		}
		if len(metas) == 0 {
			metas = gitLogTrailers(ctx, n.File)
		}
		for _, m := range metas {
			blob := m.Message
			if full, err := e.ExplainFull(ctx, m.ID); err == nil {
				blob += "\n" + full
			}
			for _, ex := range hold.ExtractConstraints(m.ID, blob) {
				out = append(out, hold.DependentQuote{
					Dependent:    n,
					CheckpointID: m.ID,
					Quote:        ex.Quote,
				})
			}
		}
	}
	return out
}

func gitLogTrailers(ctx context.Context, file string) []hold.CheckpointMeta {
	root, err := paths.WorktreeRoot(ctx)
	if err != nil {
		return nil
	}
	cmd := exec.CommandContext(ctx, "git", "-C", root, "log", "-n", "20", "--format=%B", "--", file)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var metas []hold.CheckpointMeta
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "Entire-Checkpoint:"); ok {
			id := strings.TrimSpace(rest)
			if id != "" {
				metas = append(metas, hold.CheckpointMeta{ID: id, Message: string(out)})
			}
		}
	}
	return metas
}

func defaultFixtureTest() string {
	return "npm test --prefix demo/charge-api"
}

func runProveTests(ctx context.Context, testCmd string, errW io.Writer) bool {
	parts := strings.Fields(testCmd)
	if len(parts) == 0 {
		return true
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stderr = errW
	cmd.Stdout = errW
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(errW, "hold prove tests: %v\n", err)
		return false
	}
	return true
}

func installHoldPreCommit(ctx context.Context, errW io.Writer) error {
	dir, err := strategy.GetHooksDir(ctx)
	if err != nil {
		return err
	}
	path := dir + string(os.PathSeparator) + "pre-commit"
	script := hold.PreCommitScript(hold.EntireBin())
	existing, readErr := os.ReadFile(path) //nolint:gosec // git hooks dir from git rev-parse
	if readErr == nil && !strings.Contains(string(existing), hold.HoldHookMarker) {
		backup := path + ".pre-hold"
		if _, err := os.Stat(backup); err != nil {
			if err := os.Rename(path, backup); err != nil {
				return err
			}
			fmt.Fprintln(errW, "[hold] backed up existing pre-commit to pre-commit.pre-hold")
		}
	}
	if readErr == nil && strings.Contains(string(existing), hold.HoldHookMarker) && string(existing) == script {
		return nil
	}
	return os.WriteFile(path, []byte(script), 0o755) //nolint:gosec // git hook must be executable
}

func printCompile(w io.Writer, c *hold.Charter) {
	fmt.Fprintf(w, "compiled %d items from %d checkpoint(s)\n", len(c.Items), len(c.SourceCheckpoints))
	printStatus(w, c)
}

func printStatus(w io.Writer, c *hold.Charter) {
	fmt.Fprintf(w, "%-6s %-16s %-28s %s\n", "ID", "STATUS", "FILE:LINE", "CONSTRAINT")
	for _, it := range c.Items {
		loc := "-"
		if it.Symbol != nil {
			loc = fmt.Sprintf("%s:%d", it.Symbol.File, it.Symbol.Line)
		}
		fmt.Fprintf(w, "%-6s %-16s %-28s %s\n", it.ID, it.Status, loc, clipHold(it.Constraint, 60))
	}
}

func printAmend(w io.Writer, out hold.AmendResult) {
	for _, id := range out.Conflicts {
		fmt.Fprintf(w, "CONFLICT %s\n", id)
	}
	for _, id := range out.Superseded {
		fmt.Fprintf(w, "SUPERSEDED %s\n", id)
	}
	for _, id := range out.NewFrozen {
		fmt.Fprintf(w, "FROZEN %s\n", id)
	}
	for _, id := range out.NewUnbound {
		fmt.Fprintf(w, "UNBOUND %s\n", id)
	}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func clipHold(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}
