# HXC-187 — the module-identity collision is GONE: runtime-resolution proof
Captured 2026-08-08T09:30:09Z

## Why this artifact exists
HXC-187 was reopened 2026-08-05 (item_history id=354, by=AI,
reason=captured-evidence-contradicts) because its 2026-07-28 closure cited
COMMITS ONLY -- zero tracked paths, zero on-disk evidence dirs matched hxc187.
Commit-reference-only is not captured evidence (§11.4.5/§11.4.123). This file
is the missing artifact: the SEMANTIC proof, not a narrative.

## 1. The two declared module identities
root  go.mod : module dev.helix.code/meta
inner go.mod : module dev.helix.code

## 2. The collision was on package path dev.helix.code/internal/theme
   root internal/theme  = 204 bytes (thin placeholder)
   inner internal/theme = 84K (the real subsystem)

## 3. Runtime resolution probe -- the semantic proof the ambiguity is gone
### from repo ROOT: dev.helix.code/internal/theme must NOT resolve
no required module provides package dev.helix.code/internal/theme; to add it:
	go get dev.helix.code/internal/theme
ROOT_OLD_PATH_EXIT=1   <-- non-zero = no longer claimed by the root module
### from repo ROOT: dev.helix.code/meta/internal/theme must resolve
dev.helix.code/meta/internal/theme
ROOT_NEW_PATH_EXIT=0   <-- 0 = root now owns its own namespace
### from helix_code/: dev.helix.code/internal/theme must resolve (the real one)
dev.helix.code/internal/theme
INNER_PATH_EXIT=0   <-- 0 = inner keeps the original identity

## 4. No inner package sits under the root's dev.helix.code/meta prefix
inner packages under dev.helix.code/meta : 0
(0 = disjoint namespaces; the collision cannot recur for existing packages)

## 5. Standing guard (G28) is GREEN -- see docs/qa/hxc220_module_identity_guard_verify_20260808T143000Z/

## 6. HONEST RESIDUAL RISK (§11.4.6) -- NOT covered by G28
dev.helix.code/meta is a strict SUBPATH of dev.helix.code. Creating a directory
helix_code/meta/ would re-create an ambiguous dev.helix.code/meta/... import while
BOTH module lines stayed distinct -- and G28, which compares only the two module
paths, would stay GREEN. helix_code/meta does not exist today:
ls: cannot access 'helix_code/meta': No such file or directory
HELIX_CODE_META_EXIT=2   <-- non-zero = absent
This prefix-nesting shadow is recorded here rather than silently inherited by the
closure. It is a guard-coverage gap, not a live defect.
