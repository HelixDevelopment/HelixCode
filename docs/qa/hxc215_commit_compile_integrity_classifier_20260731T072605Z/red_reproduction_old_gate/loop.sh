#!/usr/bin/env bash
# HXC-215 reproduction harness: run the (log-preserving) gate N times on a PINNED range.
set -u
ROOT=/home/milos/Factory/projects/tools_and_research/helix_code
RANGE="$(cat /tmp/hxc215/range.txt)"
N="${1:-6}"
for i in $(seq 1 "$N"); do
  ts=$(date +%Y%m%d-%H%M%S)
  out=/tmp/hxc215/runs/run${i}_${ts}
  mkdir -p "$out"
  { echo "df_home_avail_bytes=$(df -B1 /home | awk 'NR==2{print $4}')";
    echo "mem_available_kb=$(awk '/MemAvailable/{print $2}' /proc/meminfo)";
    echo "load=$(cut -d' ' -f1-3 /proc/loadavg)"; } > "$out/pre.env"
  HXC215_ROOT="$ROOT" HXC215_KEEP="$out/logs" /tmp/hxc215/gate_instrumented.sh --range "$RANGE" > "$out/stdout.txt" 2>&1
  echo "$?" > "$out/exit_code"
  { echo "df_home_avail_bytes=$(df -B1 /home | awk 'NR==2{print $4}')";
    echo "mem_available_kb=$(awk '/MemAvailable/{print $2}' /proc/meminfo)";
    echo "load=$(cut -d' ' -f1-3 /proc/loadavg)"; } > "$out/post.env"
  echo "RUN $i exit=$(cat "$out/exit_code") $(grep -E 'NON-COMPILING|^PASS:' "$out/stdout.txt" | tr '\n' ' ')"
done
