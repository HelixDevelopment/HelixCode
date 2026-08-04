#!/bin/bash
# HXC-204: uncapped duration distribution of the ConcurrentReadWrite workload
# under controlled load. Uses the ownership-tagged loadgen (§11.4.174).
set -u
REPO=/home/milos/Factory/projects/tools_and_research/helix_code
EV="$1"          # evidence dir, repo-relative
WORKERS="${2:-160}"
REPEATS="${3:-8}"

cd "$REPO" || exit 90
LOADGEN="$REPO/$EV/loadgen.sh"

cleanup() { "$LOADGEN" stop >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM

echo "=== HXC-204 uncapped duration distribution ==="
echo "date_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "nproc=$(nproc) loadavg_before=$(cut -d' ' -f1-3 /proc/loadavg)"
"$LOADGEN" start "$WORKERS"
sleep 20
echo "loadavg_after_spinup=$(cut -d' ' -f1-3 /proc/loadavg)"
"$LOADGEN" status

for ITERS in 120 60 40; do
  echo ""
  echo "=== matrix 16x${ITERS}, ${REPEATS} repeats, UNCAPPED, -race ==="
  cd "$REPO/helix_code" || exit 91
  HXC204_SCALING=1 HXC204_ITERS="$ITERS" HXC204_REPEATS="$REPEATS" \
    go test -tags=nogui -race -count=1 -timeout 3600s \
    -run TestHXC204_ScalingCurve ./internal/memory/ -v 2>&1 |
    grep -E "HXC204SCALE|--- (PASS|FAIL)|^(ok|FAIL)"
  echo "matrix_${ITERS}_pipeline_status=${PIPESTATUS[0]}"
  cd "$REPO" || exit 92
  echo "loadavg_now=$(cut -d' ' -f1-3 /proc/loadavg)"
done

echo ""
echo "=== done $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
