package hold

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Runner execs the Entire CLI. Tests replace it.
type Runner func(ctx context.Context, bin string, args []string) ([]byte, error)

func defaultRunner(ctx context.Context, bin string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return stdout.Bytes(), fmt.Errorf("entire %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}

// ExecClients talks to `entire` / `entire graph` via subprocess.
type ExecClients struct {
	Bin     string
	Repo    string
	Run     Runner
	Timeout time.Duration
}

func (e ExecClients) bin() string {
	if e.Bin != "" {
		return e.Bin
	}
	return "entire"
}

func (e ExecClients) repo() string {
	if e.Repo != "" {
		return e.Repo
	}
	return "."
}

func (e ExecClients) run(ctx context.Context, args []string) ([]byte, error) {
	r := e.Run
	if r == nil {
		r = defaultRunner
	}
	to := e.Timeout
	if to == 0 {
		to = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	return r(ctx, e.bin(), args)
}

// Doctor implements GraphClient.
func (e ExecClients) Doctor(ctx context.Context) (GraphPluginInfo, error) {
	out, err := e.run(ctx, []string{"graph", "doctor", "--json"})
	if err != nil {
		return GraphPluginInfo{Version: "unknown", DoctorOK: false}, err
	}
	ver, ok := ParseGraphDoctorOK(out)
	if ver == "" {
		ver = "graph-plugin"
	}
	return GraphPluginInfo{Version: ver, DoctorOK: ok}, nil
}

func (e ExecClients) Def(ctx context.Context, symbol, repo string) ([]Symbol, error) {
	if repo == "" {
		repo = e.repo()
	}
	out, err := e.run(ctx, []string{"graph", "def", symbol, "--repo", repo})
	if err != nil {
		return nil, err
	}
	return ParseGraphDefSymbols(out), nil
}

func (e ExecClients) Search(ctx context.Context, query, repo string) ([]Symbol, error) {
	if repo == "" {
		repo = e.repo()
	}
	out, err := e.run(ctx, []string{"graph", "search", "--repo", repo, "--profile", "full", "--format", "json", "--query", query})
	if err != nil {
		return nil, err
	}
	return ParseGraphSearchSymbols(out), nil
}

func (e ExecClients) Impact(ctx context.Context, symbol, repo string, depth int, excludeTests bool) (ImpactResult, error) {
	if repo == "" {
		repo = e.repo()
	}
	if depth <= 0 {
		depth = DefaultFreezeDepth
	}
	args := []string{"graph", "impact", "--repo", repo, "--symbol", symbol, "--depth", itoa(depth), "--format", "json"}
	if excludeTests {
		args = append(args, "--exclude-tests")
	}
	out, err := e.run(ctx, args)
	if err != nil {
		return ImpactResult{}, err
	}
	return ParseGraphImpact(out), nil
}

func (e ExecClients) Diff(ctx context.Context, base, head, repo string) ([]Change, error) {
	if repo == "" {
		repo = e.repo()
	}
	if base == "" {
		base = "HEAD~1"
	}
	if head == "" {
		head = "HEAD"
	}
	out, err := e.run(ctx, []string{"graph", "diff", "--base", base, "--head", head, "--json", "--repo", repo})
	if err != nil {
		return nil, err
	}
	return ParseGraphDiffChanges(out), nil
}

func (e ExecClients) Neighbors(ctx context.Context, symbol, repo, relation, direction string) ([]Symbol, error) {
	if repo == "" {
		repo = e.repo()
	}
	if relation == "" {
		relation = "CALLS"
	}
	if direction == "" {
		direction = "in"
	}
	out, err := e.run(ctx, []string{"graph", "neighbors", "--symbol", symbol, "--relation", relation, "--direction", direction, "--repo", repo, "--format", "json"})
	if err != nil {
		return nil, err
	}
	return ParseGraphSearchSymbols(out), nil
}

func (e ExecClients) CheckpointList(ctx context.Context) ([]CheckpointMeta, error) {
	out, err := e.run(ctx, []string{"checkpoint", "list", "--json"})
	if err != nil {
		return nil, err
	}
	return parseCheckpointList(out)
}

// ExplainFull is a local `entire checkpoint explain` subprocess — not HTTP.
func (e ExecClients) ExplainFull(ctx context.Context, id string) (string, error) {
	out, err := e.run(ctx, []string{"checkpoint", "explain", id, "--full", "--no-pager"})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (e ExecClients) ExplainRawTranscript(ctx context.Context, id string) (string, error) {
	out, err := e.run(ctx, []string{"checkpoint", "explain", id, "--raw-transcript", "--no-pager"})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (e ExecClients) Why(ctx context.Context, fileLine string) ([]CheckpointMeta, error) {
	out, err := e.run(ctx, []string{"why", fileLine, "--json"})
	if err != nil {
		return nil, err
	}
	return parseWhyBlame(out)
}

func (e ExecClients) Blame(ctx context.Context, file string) ([]CheckpointMeta, error) {
	out, err := e.run(ctx, []string{"blame", file, "--json"})
	if err != nil {
		return nil, err
	}
	return parseWhyBlame(out)
}

func parseCheckpointList(raw []byte) ([]CheckpointMeta, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, nil
	}
	var list []CheckpointMeta
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}
	var wrap struct {
		Checkpoints []CheckpointMeta `json:"checkpoints"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Checkpoints) > 0 {
		return wrap.Checkpoints, nil
	}
	return collectCheckpointIDs(raw), nil
}

func parseWhyBlame(raw []byte) ([]CheckpointMeta, error) {
	if ids := collectCheckpointIDs(raw); len(ids) > 0 {
		return ids, nil
	}
	return parseCheckpointList(raw)
}

func collectCheckpointIDs(raw []byte) []CheckpointMeta {
	var out []CheckpointMeta
	seen := map[string]bool{}
	walk(decodeJSON(raw), func(m map[string]any) {
		id, _ := firstString(m, "checkpoint_id", "id", "Entire-Checkpoint")
		if id == "" {
			return
		}
		if seen[id] {
			return
		}
		seen[id] = true
		sess, _ := firstString(m, "session_id", "session")
		msg, _ := firstString(m, "message", "quote")
		out = append(out, CheckpointMeta{ID: id, Session: sess, Message: msg})
	})
	return out
}

// EntireBin resolves the CLI to exec for graph/checkpoint (PATH, else this process).
func EntireBin() string {
	if p, err := exec.LookPath("entire"); err == nil {
		return p
	}
	if e, err := os.Executable(); err == nil {
		return e
	}
	return "entire"
}
