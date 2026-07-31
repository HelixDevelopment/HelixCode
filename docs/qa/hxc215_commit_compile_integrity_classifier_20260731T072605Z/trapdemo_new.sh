#!/usr/bin/env bash
set -uo pipefail
cleanup(){ echo "CLEANUP RAN"; }
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
echo "phase-1"
( sleep 2 ) &
kill -TERM $$ 2>/dev/null
wait
echo "PHASE-2-STILL-RUNNING  <-- script resumed after SIGTERM"
