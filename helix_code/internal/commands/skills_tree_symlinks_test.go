package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// HXC-163 (second half) — no entry in a skills tree may be present-but-
// unresolvable.
//
// # RUNTIME SIGNATURE (§11.4.108)
//
// Every symlink under a skills tree resolves on THIS host, and none of them
// encodes an absolute path into a foreign-host mount point.
//
// This is the surface the Go skill registry cannot see. The registry loads
// manifests through the embedded built-in tier and two configured directories;
// a shipped-but-dangling entry elsewhere in the skills tree is simply invisible
// to it. That is precisely how `skills/media-validator` came to be a tracked
// symlink pointing at `/Volumes/T7/...` — a macOS volume that cannot exist on a
// Linux host — while every test stayed green.
//
// The absolute-path assertion is the one that matters most: a dangling link is
// only the SYMPTOM, and it would come back the moment the registration step is
// re-run on a machine where the absolute path happens to resolve. Requiring the
// target to be relative fixes the class, not the instance.
//
// # HONEST BOUNDARY (§11.4.6)
//
// This guard needs a real checkout to inspect. When the repository root cannot
// be located it SKIPs with a reason rather than passing silently — a skip is
// reported, an empty pass is not.

// foreignHostMountPrefixes are absolute path roots that are host-specific by
// construction. A tracked symlink into one of these can only resolve on the
// single machine it was authored on.
var foreignHostMountPrefixes = []string{
	"/Volumes/",   // macOS external volumes
	"/Users/",     // macOS home directories
	"/home/",      // Linux home directories
	"/run/media/", // Linux removable-media automounts
	"/media/",     // Linux removable-media automounts
	"/mnt/",       // arbitrary manual mounts
}

// findRepoRoot walks upward from the working directory looking for the
// meta-repo root, identified by carrying BOTH a .gitmodules file and a
// constitution/ directory. Returns ("", false) when not found.
func findRepoRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".gitmodules")); err == nil {
			if info, err := os.Stat(filepath.Join(dir, "constitution")); err == nil && info.IsDir() {
				return dir, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func TestSkillsTree_NoDanglingOrHostAbsoluteSymlinks(t *testing.T) {
	root, ok := findRepoRoot()
	if !ok {
		t.Skip("SKIP-OK: repository root not locatable from the test working directory; " +
			"this guard inspects a real checkout's skills trees and has nothing to assert without one")
	}

	// Both skills trees: the repo-root one (consumed by the MCP configuration
	// through the documented `skills/<name>/...` path) and the governance one.
	trees := []string{
		filepath.Join(root, "skills"),
		filepath.Join(root, "constitution", "skills"),
	}

	checked := 0
	for _, tree := range trees {
		if _, err := os.Stat(tree); err != nil {
			continue // tree absent in this checkout: nothing to assert
		}
		err := filepath.WalkDir(tree, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// A dangling symlink encountered as a directory entry surfaces
				// here; report it rather than aborting the whole walk.
				t.Errorf("skills tree walk error at %s: %v", path, err)
				return nil
			}
			if d.Type()&os.ModeSymlink == 0 {
				return nil
			}
			checked++

			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				rel = path
			}

			target, readErr := os.Readlink(path)
			if readErr != nil {
				t.Errorf("%s: cannot read symlink target: %v", rel, readErr)
				return nil
			}

			// (1) It must resolve on THIS host.
			if _, statErr := os.Stat(path); statErr != nil {
				t.Errorf("%s is a DANGLING symlink -> %q: it does not resolve on this host, "+
					"so anything consuming that path silently gets nothing", rel, target)
			}

			// (2) It must not encode a host-specific absolute path. This is the
			// root-cause assertion: such a link is unportable by construction
			// even on a machine where it happens to resolve.
			if filepath.IsAbs(target) {
				for _, prefix := range foreignHostMountPrefixes {
					if strings.HasPrefix(target, prefix) {
						t.Errorf("%s points at the host-specific absolute path %q. "+
							"Tracked symlinks must use a repository-relative target "+
							"(for example ../constitution/skills/<name>) so they resolve in every "+
							"checkout on every platform, not only on the machine that created them.",
							rel, target)
						break
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Errorf("walking %s: %v", tree, err)
		}
	}

	t.Logf("checked %d symlink(s) across the skills trees under %s", checked, root)
}
