#!/bin/bash
# HXC-204 load generator.
#
# §11.4.174: every process this spawns carries the unique marker HXC204LOADMARK in
# its argv, and stop() kills ONLY processes carrying that marker. It never uses a
# bare pattern like `pkill -f sh` that could match another project's work on this
# shared host.
#
# §12.12 / cgroup pids: the binding ceiling here is pids.max=4096 on the project
# session slice (NOT ulimit -u). start() refuses to spawn if it would take the
# slice's pids.current above a safety fraction of that ceiling.

set -u
MARK=HXC204LOADMARK

# min pids.max across every level of our cgroup path, and current usage at the
# level that imposes it.
cgroup_headroom() {
  local rel path mx cur min_mx=999999999 min_cur=0 p
  rel=$(cut -d: -f3 /proc/self/cgroup)
  path=/sys/fs/cgroup
  IFS='/' read -ra parts <<< "$rel"
  for p in "${parts[@]}"; do
    [ -z "$p" ] && continue
    path="$path/$p"
    mx=$(cat "$path/pids.max" 2>/dev/null || echo max)
    cur=$(cat "$path/pids.current" 2>/dev/null || echo 0)
    if [ "$mx" != "max" ] && [ "$mx" -lt "$min_mx" ] 2>/dev/null; then
      min_mx=$mx; min_cur=$cur
    fi
  done
  echo "$min_cur $min_mx"
}

start() {
  local n="${1:-192}"
  read -r cur mx <<< "$(cgroup_headroom)"
  local budget=$(( mx * 70 / 100 - cur ))
  if [ "$n" -gt "$budget" ]; then
    echo "loadgen: requested $n workers but cgroup budget is $budget (cur=$cur max=$mx) -- clamping" >&2
    n=$budget
  fi
  if [ "$n" -lt 1 ]; then
    echo "loadgen: NO headroom (cur=$cur max=$mx) -- refusing to spawn" >&2
    return 1
  fi
  local i
  for i in $(seq 1 "$n"); do
    setsid bash -c "$MARK=1; while :; do :; done" >/dev/null 2>&1 &
  done
  disown -a 2>/dev/null || true
  echo "loadgen: started $n busy workers (cgroup cur=$cur max=$mx)"
}

stop() {
  # Kill ONLY processes whose argv carries our marker, and never this shell.
  local pids
  pids=$(pgrep -f "$MARK" 2>/dev/null | grep -v "^$$\$" | grep -v "^$PPID\$")
  if [ -n "$pids" ]; then
    echo "$pids" | xargs -r kill -9 2>/dev/null
  fi
  sleep 0.3
  echo "loadgen: stopped; remaining marked=$(pgrep -cf "$MARK" 2>/dev/null || echo 0)"
}

status() {
  read -r cur mx <<< "$(cgroup_headroom)"
  echo "loadgen: marked=$(pgrep -cf "$MARK" 2>/dev/null || echo 0) loadavg=$(cut -d' ' -f1-3 /proc/loadavg) runnable=$(cut -d/ -f1 /proc/loadavg >/dev/null; awk '{print $4}' /proc/loadavg) cgroup_pids=$cur/$mx"
}

case "${1:-}" in
  start) shift; start "$@" ;;
  stop) stop ;;
  status) status ;;
  *) echo "usage: $0 {start [n]|stop|status}" >&2; exit 2 ;;
esac
