package shell

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// background_security_workdir_test.go — standing regression guards for HXC-209
// (CRITICAL) and HXC-210 (High).
//
// THE SHARED ROOT CAUSE
//
// ExecuteWithProgress (background.go) was written as a parallel implementation
// rather than as another entry point onto DefaultExecutor. It built its own
// exec.Cmd from the raw params map and never consulted the policy the other
// three entry points share, so every guarantee that lives in that policy was
// simply absent on this path — not weaker, absent.
//
// HXC-209 — command-security validation was entirely missing.
//
//	DefaultExecutor.Execute       (executor.go:199) → e.security.ValidateCommand
//	DefaultExecutor.ExecuteAsync  (executor.go:329) → e.security.ValidateCommand
//	DefaultExecutor.ExecuteStream (executor.go:357) → e.security.ValidateCommand
//	ShellExecutor.ExecuteWithProgress                → (nothing)
//
// The route is selected by an ordinary caller-supplied boolean —
// `run_in_background: true` on the same tool, dispatched through
// ToolRegistry.adaptToolForBackground (registry.go:1222). So anyone able to ask
// for a shell command could also ask for it in the background and step around
// the blocklist, the allowlist, the dangerous-pattern screen and the workdir
// check together. Identical interface, divergent enforcement: the hallmark of
// this defect class, and the reason the guards below are written as DIFFERENTIAL
// tests. Each first proves the foreground path refuses the command, then asserts
// the background path does the same. That framing is deliberate — it keeps the
// guard honest if the policy itself is ever retuned, because the premise is
// re-established at run time rather than hardcoded.
//
// HXC-210 — the working directory was silently dropped.
//
// background.go:56 read params["cwd"]. The published schema emits "workdir"
// (ShellTool.Schema, shell_tools.go:39) and the synchronous path reads "workdir"
// (shell_tools.go:70), so the key that was actually sent was never the key that
// was read. Nothing failed: no error, no warning, no fallback notice. The
// command simply ran wherever the server process happened to be. The dangerous
// outcome is not a failure — it is a command SUCCEEDING against the wrong files,
// which is why the guard below asserts the OBSERVED directory rather than that
// the parameter was accepted.
//
// §11.4.115 RED-polarity switch: ONE test source, two roles.
//
//	RED_MODE=1 → reproduce each defect on the pre-fix artifact, proving it with
//	             POSITIVE evidence — a blocklisted command's side effect actually
//	             landing, a pwd that actually reports the wrong directory. An
//	             absent error would only show the guard did not fire; a deleted
//	             canary shows the command ran.
//	RED_MODE=0 → standing GREEN guard (default): the command is refused BEFORE it
//	             executes, and the working directory is honoured.
//
// §11.4.135: these are ordinary `go test` cases in the package the defect lives
// in, exactly as the HXC-198 guard next door is registered, so they run on every
// build with no separate wiring to fall out of sync.

// redMode reports whether the suite is running in §11.4.115 reproduce-the-defect
// polarity.
func redMode() bool { return os.Getenv("RED_MODE") == "1" }

// TestExecuteWithProgressEnforcesBlocklist is the HXC-209 guard on the blocklist
// dimension.
//
// `rm` is in DefaultBlocklist(). The command is confined to this test's own
// TempDir, so the only observable effect anywhere on the host is a deleted
// canary file this test created (§12 host safety).
func TestExecuteWithProgressEnforcesBlocklist(t *testing.T) {
	skipIfWindows(t)

	dir := t.TempDir()
	canary := filepath.Join(dir, "canary.txt")
	require.NoError(t, os.WriteFile(canary, []byte("present"), 0o600))

	command := "rm -f " + canary
	executor := NewShellExecutor(DefaultConfig())

	// Establish the premise at run time: this exact command IS refused on the
	// foreground path. Without this the guard could pass against a policy that
	// no longer blocks anything.
	_, fgErr := executor.Execute(context.Background(), &Command{
		ID:      "hxc209-blocklist-foreground",
		Command: command,
	})
	var secErr *SecurityError
	require.ErrorAs(t, fgErr, &secErr,
		"premise: the foreground path must refuse a blocklisted command")
	require.FileExists(t, canary,
		"premise: a command refused on the foreground path must not have run")

	_, bgErr := executor.ExecuteWithProgress(context.Background(), map[string]interface{}{
		"command": command,
	}, func(string) {})

	if redMode() {
		require.NoError(t, bgErr,
			"RED_MODE: expected the BROKEN behaviour — the background path runs the "+
				"command without consulting the security policy at all")
		require.NoFileExists(t, canary,
			"RED_MODE: the blocklisted command must actually have EXECUTED. The deleted "+
				"canary is the positive evidence that the guard was BYPASSED; a merely "+
				"absent error would not distinguish that from a guard that passed it")
		t.Log("RED_MODE: defect reproduced — a command the foreground path refuses ran " +
			"to completion on the background path and deleted its target")
		return
	}

	require.ErrorAs(t, bgErr, &secErr,
		"the background path must refuse exactly what the foreground path refuses — "+
			"the route is chosen by a caller-supplied boolean, so divergent enforcement "+
			"is an authorization bypass")
	assert.FileExists(t, canary,
		"the refusal must happen BEFORE execution — a surviving canary is what separates "+
			"a real guard from an error returned after the damage was already done")
}

// TestExecuteWithProgressEnforcesAllowlist is the HXC-209 guard on the allowlist
// dimension — a second, independent enforcement path through ValidateCommand.
//
// A fix that only consulted the blocklist would satisfy the test above while
// leaving allowlist-mode deployments (StrictConfig) exposed.
func TestExecuteWithProgressEnforcesAllowlist(t *testing.T) {
	skipIfWindows(t)

	dir := t.TempDir()
	artifact := filepath.Join(dir, "created-by-touch.txt")

	// StrictConfig's allowlist is {ls, cat, echo, pwd}. `touch` is absent from it
	// and absent from the blocklist, so only the allowlist can reject this.
	command := "touch " + artifact
	executor := NewShellExecutor(StrictConfig())

	_, fgErr := executor.Execute(context.Background(), &Command{
		ID:      "hxc209-allowlist-foreground",
		Command: command,
	})
	var secErr *SecurityError
	require.ErrorAs(t, fgErr, &secErr,
		"premise: the foreground path must enforce the allowlist under StrictConfig")
	require.NoFileExists(t, artifact,
		"premise: a command refused on the foreground path must not have run")

	_, bgErr := executor.ExecuteWithProgress(context.Background(), map[string]interface{}{
		"command": command,
	}, func(string) {})

	if redMode() {
		require.NoError(t, bgErr,
			"RED_MODE: expected the BROKEN behaviour — the background path ignores the "+
				"allowlist as completely as it ignores the blocklist")
		require.FileExists(t, artifact,
			"RED_MODE: the non-allowlisted command must actually have EXECUTED — the "+
				"created file is the positive evidence of the bypass")
		t.Log("RED_MODE: defect reproduced — a command outside the strict allowlist ran " +
			"on the background path and created its target")
		return
	}

	require.ErrorAs(t, bgErr, &secErr,
		"the background path must enforce the allowlist, not only the blocklist")
	assert.NoFileExists(t, artifact,
		"the refusal must precede execution")
}

// TestExecuteWithProgressHonoursWorkdir is the HXC-210 guard.
//
// It asserts the directory the command OBSERVES, via pwd, rather than that the
// parameter was accepted — accepting a parameter and then discarding it is
// precisely the defect.
func TestExecuteWithProgressHonoursWorkdir(t *testing.T) {
	skipIfWindows(t)

	// Resolve both sides through EvalSymlinks: temp roots are symlinked on some
	// platforms (/var → /private/var on macOS), and comparing an unresolved
	// request against a resolved observation would fail for the wrong reason.
	workdir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)

	processCwd, err := os.Getwd()
	require.NoError(t, err)
	processCwd, err = filepath.EvalSymlinks(processCwd)
	require.NoError(t, err)

	require.NotEqual(t, workdir, processCwd,
		"premise: the requested directory must differ from where the process already "+
			"is, or a passing assertion would prove nothing")

	executor := newShellToolInstance()
	sink, delivered := collectingSink()

	// "workdir" is the key the published schema emits and the key the
	// synchronous path reads. It is the key a caller actually sends.
	out, err := executor.ExecuteWithProgress(context.Background(), map[string]interface{}{
		"command": "pwd",
		"workdir": workdir,
	}, sink)
	require.NoError(t, err)

	lines := delivered()
	require.Len(t, lines, 1, "pwd emits exactly one line")
	observed, err := filepath.EvalSymlinks(strings.TrimSpace(lines[0]))
	require.NoError(t, err)

	if redMode() {
		require.Equal(t, processCwd, observed,
			"RED_MODE: expected the BROKEN behaviour — the schema's \"workdir\" key is "+
				"never read (background.go reads \"cwd\"), so the command runs wherever "+
				"the server process happens to be")
		require.NotEqual(t, workdir, observed,
			"RED_MODE: the requested directory must NOT have taken effect")
		t.Logf("RED_MODE: defect reproduced — requested %s, command actually ran in %s, "+
			"and nothing reported the discrepancy", workdir, observed)
		return
	}

	assert.Equal(t, workdir, observed,
		"the command must execute IN the requested directory — the danger of this defect "+
			"is not a failure but a command SUCCEEDING against the wrong files")

	payload, ok := out.(map[string]interface{})
	require.True(t, ok, "ExecuteWithProgress returns a map payload")
	assert.Equal(t, 0, payload["exit_code"], "pwd succeeds")
	aggregated, err := filepath.EvalSymlinks(strings.TrimSpace(payload["output"].(string)))
	require.NoError(t, err)
	assert.Equal(t, workdir, aggregated,
		"the aggregated return value must agree with what the sink observed")
}

// TestExecuteWithProgressValidatesWorkdir guards the intersection of the two
// defects.
//
// Reading the working directory is only half of honouring it: the foreground
// path also VALIDATES it (ValidateCommand → isValidPath, executor.go:210), which
// rejects traversal and command-substitution attempts. A fix that read "workdir"
// but skipped validation would have quietly turned a dropped parameter into an
// unchecked one — trading a silent no-op for a live traversal vector.
func TestExecuteWithProgressValidatesWorkdir(t *testing.T) {
	skipIfWindows(t)

	// Clean() keeps the leading "..", so isValidPath rejects it.
	const traversal = "../../etc"
	executor := newShellToolInstance()

	_, fgErr := executor.Execute(context.Background(), &Command{
		ID:      "hxc210-traversal-foreground",
		Command: "pwd",
		WorkDir: traversal,
	})
	var secErr *SecurityError
	require.ErrorAs(t, fgErr, &secErr,
		"premise: the foreground path rejects a traversal working directory")

	_, bgErr := executor.ExecuteWithProgress(context.Background(), map[string]interface{}{
		"command": "pwd",
		"workdir": traversal,
	}, func(string) {})

	if redMode() {
		require.NoError(t, bgErr,
			"RED_MODE: expected the BROKEN behaviour — \"workdir\" is not read at all, so "+
				"there is nothing to validate and the command runs in the process cwd")
		t.Log("RED_MODE: defect reproduced — a traversal workdir was neither honoured " +
			"nor rejected; it was silently ignored")
		return
	}

	require.ErrorAs(t, bgErr, &secErr,
		"a working directory that reaches the background path must be validated there "+
			"too — honouring it without checking it would be a worse defect than "+
			"dropping it")
}

// TestExecuteWithProgressAllowsPermittedCommand is the preserve-property guard.
//
// Asserted unconditionally in BOTH polarities. A "fix" that refused everything
// would satisfy every guard above while destroying the feature, and §11.4.122
// forbids securing a capability by removing it. This is the assertion that makes
// the others mean "enforces policy" rather than "rejects input".
func TestExecuteWithProgressAllowsPermittedCommand(t *testing.T) {
	skipIfWindows(t)

	executor := NewShellExecutor(DefaultConfig())
	sink, delivered := collectingSink()

	out, err := executor.ExecuteWithProgress(context.Background(), map[string]interface{}{
		"command": "echo hello-from-background",
	}, sink)
	require.NoError(t, err,
		"a permitted command must still run on the background path under the "+
			"production default config")

	assert.Equal(t, []string{"hello-from-background"}, delivered(),
		"streaming must still deliver the command's output line by line")

	payload, ok := out.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 0, payload["exit_code"])
	assert.Equal(t, "hello-from-background", payload["output"])
	assert.Equal(t, false, payload["output_incomplete"],
		"a normally-terminating command loses nothing")
}
