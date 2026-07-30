package tools

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shell_tools_background_security_test.go — end-to-end guards for HXC-209
// (CRITICAL) and HXC-210, asserted on the ACTUAL dispatch route.
//
// The package-level guards in internal/tools/shell/background_security_workdir_
// test.go prove ShellExecutor.ExecuteWithProgress enforces the policy. These
// prove the property the end user actually depends on: that the enforcement
// holds through the route a caller reaches, with the configuration production
// really runs.
//
// That distinction is the §11.4.108 layer-3 seam, and it is not ceremony here.
// The whole defect was that two entry points onto the same executor disagreed,
// so a guard proving only "the function I just edited validates" would be
// checking the layer that was never in doubt. What matters is that
// `run_in_background: true` — an ordinary boolean on an ordinary tool call —
// can no longer buy a caller anything the synchronous call would have refused.
//
// The route under test:
//
//	ToolRegistry.adaptToolForBackground (registry.go:1222)
//	  → BackgroundAware.ExecuteWithProgress
//	  → ShellTool.ExecuteWithProgress      (shell_tools.go:88)
//	  → ShellExecutor.ExecuteWithProgress  (shell/background.go)
//
// The tool is obtained from a real ToolRegistry via DefaultRegistryConfig, which
// wires shell.DefaultConfig() (registry.go:170) — the production posture, where
// the allowlist is disabled and the blocklist is live. So `rm` here is blocked
// by the same policy a deployed server enforces, not by a test-only tightening.

func skipIfNotPOSIX(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("SKIP-OK: shell test relies on POSIX bash; CI runs on Linux only for now")
	}
}

// backgroundAwareShellTool returns the registry-registered shell tool through
// the BackgroundAware interface the background dispatcher uses.
func backgroundAwareShellTool(t *testing.T) BackgroundAware {
	t.Helper()
	tool, err := newShellToolForBackgroundTest(t)
	require.NoError(t, err)
	ba, ok := tool.(BackgroundAware)
	require.True(t, ok, "*ShellTool must satisfy BackgroundAware")
	return ba
}

// TestShellTool_BackgroundDoesNotBypassSecurity is the HXC-209 end-to-end guard.
//
// It is written as a DIFFERENTIAL test between the two dispatch modes of one
// tool, because that is exactly what the defect was: same tool, same params,
// same executor — and, until this fix, wildly different enforcement depending on
// a boolean the caller controls.
func TestShellTool_BackgroundDoesNotBypassSecurity(t *testing.T) {
	skipIfNotPOSIX(t)

	dir := t.TempDir()
	canary := filepath.Join(dir, "canary.txt")
	require.NoError(t, os.WriteFile(canary, []byte("present"), 0o600))

	// `rm` is in shell.DefaultBlocklist(), which DefaultRegistryConfig activates.
	// The blast radius is this test's own TempDir and nothing else (§12).
	params := map[string]interface{}{"command": "rm -f " + canary}

	tool, err := newShellToolForBackgroundTest(t)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Synchronous dispatch: refused. This is the baseline expectation a caller
	// forms about the tool, and the promise the background flag used to break.
	_, syncErr := tool.Execute(ctx, params)
	require.Error(t, syncErr,
		"premise: the synchronous dispatch of this tool refuses a blocklisted command")
	require.FileExists(t, canary, "premise: a refused command must not have run")

	// Background dispatch: the SAME tool, the SAME params, reached the way
	// ToolRegistry reaches it when run_in_background is set.
	ba, ok := tool.(BackgroundAware)
	require.True(t, ok)
	_, bgErr := ba.ExecuteWithProgress(ctx, params, LineSink(func(string) {}))

	require.Error(t, bgErr,
		"asking for the same command in the background must not make it permissible — "+
			"the background flag is caller-controlled, so any enforcement gap between "+
			"the two dispatch modes is an authorization bypass")
	assert.FileExists(t, canary,
		"the command must be refused BEFORE it executes; a surviving canary is the "+
			"positive evidence, where a mere error return would not distinguish "+
			"'refused' from 'ran, then complained'")
}

// TestShellTool_BackgroundHonoursSchemaWorkdir is the HXC-210 end-to-end guard.
//
// It sends the parameter under the name the tool's own published Schema()
// advertises, which is the contract a caller codes against. Asserting the
// OBSERVED directory is the point: the defect was never a rejected parameter, it
// was an accepted-and-discarded one, so only what the command actually saw can
// tell the two apart.
func TestShellTool_BackgroundHonoursSchemaWorkdir(t *testing.T) {
	skipIfNotPOSIX(t)

	workdir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)

	processCwd, err := os.Getwd()
	require.NoError(t, err)
	processCwd, err = filepath.EvalSymlinks(processCwd)
	require.NoError(t, err)
	require.NotEqual(t, workdir, processCwd,
		"premise: the requested directory must differ from the process cwd, or a "+
			"passing assertion would prove nothing")

	// The key is taken from the tool's own schema rather than hardcoded, so this
	// guard tracks the published contract if it is ever renamed — a rename that
	// silently desynchronised schema from implementation is the whole defect.
	tool, err := newShellToolForBackgroundTest(t)
	require.NoError(t, err)
	_, declared := tool.Schema().Properties["workdir"]
	require.True(t, declared,
		"the shell tool must publish a \"workdir\" property; if this name changes, the "+
			"implementation must change with it")

	ba := backgroundAwareShellTool(t)

	var (
		mu    sync.Mutex
		lines []string
	)
	sink := LineSink(func(line string) {
		mu.Lock()
		lines = append(lines, line)
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = ba.ExecuteWithProgress(ctx, map[string]interface{}{
		"command": "pwd",
		"workdir": workdir,
	}, sink)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, lines, 1, "pwd emits exactly one line")
	observed, err := filepath.EvalSymlinks(strings.TrimSpace(lines[0]))
	require.NoError(t, err)

	assert.Equal(t, workdir, observed,
		"a background command must run in the directory the caller asked for; the "+
			"hazard of this defect was not failure but a command SUCCEEDING against "+
			"the wrong files")
	assert.NotEqual(t, processCwd, observed,
		"and specifically not in whatever directory the server process happens to "+
			"occupy, which is where the dropped parameter used to leave it")
}
