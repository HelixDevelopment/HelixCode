# Agent stall root cause — my GOMAXPROCS attribution was WRONG
Captured 2026-08-07T10:10:12Z

## What I claimed
In commit 40c8d5ce and in ~8 agent briefs I stated the cgroup/GOMAXPROCS
oversubscription was why agents overran the 600s stream watchdog, and told
agents 'this should materially reduce your chance of being killed'.

## Evidence 1 — transcript size INVERSELY predicts death
  STALLED : 270, 470, 502, 552 KB   mean ~449
  COMPLETED: 499, 583, 685, 707, 806 KB   mean ~656
Starvation predicts heavier work dies MORE. The opposite holds: the agents
that did the most work completed. Size does not predict death.

## Evidence 2 — deaths do not cluster in time
  stalls    2026-08-05 16:18, 17:17, 19:43, 08-06 16:58, 08-07 02:59
  completes 2026-08-06 18:50, 21:03, 23:43, 08-07 05:57, 11:06
Interleaved, no clean separation, no common host condition at those moments.

## Evidence 3 — the decisive one: WHERE they died
All four stalled agents' final text is an INCOMPLETE SENTENCE, composed just
before issuing a tool call:
  HXC-229 'I'll start by investigating the current state of the systemd unit'
  HXC-218 'url.Parse is lenient here ... let me check that recorded output'
  HXC-217 'Now I have the full picture. Let me parallelize: dispatch'
  HXC-221 'Fixing the fixture to be genuinely stateful:'

The watchdog measures STREAM output. A CPU-starved agent still emits tokens,
only slower, and would die DURING a long command with that command as its
last action. These died while COMPOSING TEXT — the token stream stopped
mid-generation and never resumed. That is an API/stream-layer condition, not
a host-resource one.

## Corrected conclusion
The cgroup finding is REAL and stands on its own: cpu.max 860000/100000 = 8.6
CPUs vs nproc 64, 50.3% of periods throttled, 75.5h frozen, GOMAXPROCS=8 beat
64 on a trivial build. The Makefile fix is correct and worth keeping.

But it does NOT explain the stalls, and I asserted that it did without
testing it. Two separate facts got welded into one causal story because they
appeared in the same investigation.

## What actually mitigates stalls
Respawn-on-stall with preserved partial work — which is what SS11.4.147
already prescribes and what has in fact recovered every stalled agent today.
Not a tuning knob. The mitigation was already correct; my explanation was not.

## Honest boundary
I cannot fix an upstream token-stream interruption from here. What I can do,
and have done, is stop telling agents a CPU setting will protect them.
