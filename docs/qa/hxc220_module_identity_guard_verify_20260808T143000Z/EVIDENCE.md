# HXC-220 — first-hand verification that the module-name guard exists, is wired, and is falsifiable
Captured 2026-08-08T09:29:49Z

## 1. GREEN polarity (standing guard)
CM-MODULE-IDENTITY-EXACT-MATCH  RED_MODE=0
  root  go.mod module path : dev.helix.code/meta   (/home/milos/Factory/projects/tools_and_research/helix_code/go.mod)
  inner go.mod module path : dev.helix.code   (/home/milos/Factory/projects/tools_and_research/helix_code/helix_code/go.mod)
  S1a genuine recurrence caught (identical paths)         : yes   (want yes)
  S1b prefix-lookalike NOT falsely caught, BOTH orders    : yes   (want yes)
  S1c OLD substring predicate DOES misfire on the lookalike: yes   (want yes — reproduces the historical bug's mechanism)
  S1d historical R-26 pair NOT falsely caught, BOTH orders: yes   (want yes)
GREEN PASS — root go.mod (dev.helix.code/meta) and inner go.mod (dev.helix.code) are exact-match distinct,
             verified via module_paths_identical() (scripts/lib/module_identity.sh) — never a
             substring/prefix test. The reconstructed OLD substring predicate would have
             wrongly flagged this pair as a collision (see S1c above); this guard would not.
GREEN_EXIT=0

## 2. RED polarity (reproduces the loose-substring false alarm)
CM-MODULE-IDENTITY-EXACT-MATCH  RED_MODE=1
  root  go.mod module path : dev.helix.code/meta   (/home/milos/Factory/projects/tools_and_research/helix_code/go.mod)
  inner go.mod module path : dev.helix.code   (/home/milos/Factory/projects/tools_and_research/helix_code/helix_code/go.mod)
  S1a genuine recurrence caught (identical paths)         : yes   (want yes)
  S1b prefix-lookalike NOT falsely caught, BOTH orders    : yes   (want yes)
  S1c OLD substring predicate DOES misfire on the lookalike: yes   (want yes — reproduces the historical bug's mechanism)
  S1d historical R-26 pair NOT falsely caught, BOTH orders: yes   (want yes)
  old-substring verdict on real root go.mod line : collision(WRONG)
  exact-match   verdict on real module paths     : distinct
RED PASS — reproduced the false positive: the OLD substring predicate reports a collision
           on the CURRENT tree (root='dev.helix.code/meta' inner='dev.helix.code') even though the
           two module paths are exact-match distinct. This is precisely the defect
           HXC-199 exists to guard against — a reader trusting the old predicate would be
           misled into believing the R-26 collision is still unresolved.
RED_EXIT=0

## 3. Registered in the standing suite (explicit list, not a glob)
1049:if want_gate G28; then

## 4. Guard + predicate are tracked
scripts/gates/hxc199_module_identity_exact_match_gate.sh
scripts/lib/module_identity.sh
TRACKED_EXIT=0
