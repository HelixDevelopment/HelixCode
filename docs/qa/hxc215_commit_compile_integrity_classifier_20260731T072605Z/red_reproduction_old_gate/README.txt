RED reproduction of HXC-215 against the OLD gate logic (§11.4.115).

Produced by my predecessor, verified by me before use:
  * loop.sh drove /tmp/hxc215/gate_instrumented.sh — a copy of the gate carrying
    the OLD verdict logic (grep confirms ZERO occurrences of classify_build_failure)
    plus log preservation via HXC215_KEEP.
  * pinned_range.txt is the fixed SHA range, so every run had an identical input.

loop_8runs_old_gate.out — 8 runs, unchanged tree, pinned range:
    RUN 1 exit=0 PASS      RUN 5 exit=1
    RUN 2 exit=0 PASS      RUN 6 exit=1 NON-COMPILING COMMITS: 11861996
    RUN 3 exit=1           RUN 7 exit=0 PASS
    RUN 4 exit=0 PASS      RUN 8 exit=1 NON-COMPILING COMMITS: d99ce58c
  4 PASS / 4 FAIL on identical input, naming two MORE innocent commits.

The two kept logs are the causes, and neither is a source defect:

  run6_11861996_EAGAIN.log
    fork/exec .../compile: resource temporarily unavailable
    = EAGAIN on fork(2): RLIMIT_NPROC exhausted (§12.12). No commit was read.

  run8_d99ce58c_WORKDIR_VANISHED.log
    chdir .../cci-gate.7GVMhX/d99ce58c/helix_code: no such file or directory
    = the gate's own worktree deleted mid-build by `trap cleanup EXIT INT TERM`.
      The trap defect MANUFACTURED the evidence the classifier defect misread.

Both logs are fed to the FINAL classifier in ../classifier_vs_real_flake_logs.txt,
where both classify INFRA — blocking, but naming nobody.
