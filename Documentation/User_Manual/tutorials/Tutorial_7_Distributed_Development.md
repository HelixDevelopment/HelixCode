# Tutorial 7: Distributed Development with Workers

**Duration**: 30 minutes
**Level**: Advanced

## Overview

Scale development with distributed SSH workers:
- Add remote workers
- Auto-installation
- Task distribution
- Health monitoring

## Step 1: Prepare Worker Machines

```bash
# On each worker machine
# Ensure SSH access and Go installed

# From control machine
ssh worker1.example.com
ssh worker2.example.com
```

## Step 2: Add Workers to HelixCode

```bash
# Add first worker
helixcode worker add \
  --host worker1.example.com \
  --user helix \
  --key ~/.ssh/id_rsa

# HelixCode automatically:
# 1. Connects via SSH
# 2. Detects hardware (CPU, RAM, GPU)
# 3. Installs HelixCode CLI
# 4. Starts worker agent

# Output:
# ✓ Connected to worker1.example.com
# ✓ Hardware detected: 16 CPU, 64GB RAM, NVIDIA RTX 4090
# ✓ Installing HelixCode CLI...
# ✓ Worker registered: worker-001

# Add more workers
helixcode worker add --host worker2.example.com --user helix --key ~/.ssh/id_rsa
helixcode worker add --host worker3.example.com --user helix --key ~/.ssh/id_rsa
```

## Step 3: View Worker Pool

```bash
helixcode worker list

# ID          Host                 Status   CPU   RAM    GPU            Tasks
# worker-001  worker1.example.com  healthy  16    64GB   RTX 4090       2/10
# worker-002  worker2.example.com  healthy  32    128GB  RTX 4090 x2    5/20
# worker-003  worker3.example.com  healthy  8     32GB   None           1/5
```

## Step 4: Distribute Tasks

```bash
# Create parallel tasks
helixcode task create "Build Linux binary" --worker worker-001
helixcode task create "Build macOS binary" --worker worker-002
helixcode task create "Build Windows binary" --worker worker-003

# Or let HelixCode auto-assign
helixcode task create "Run test suite" --auto-assign

# HelixCode considers:
# - Worker capabilities (OS, arch, GPU)
# - Current load
# - Task requirements
```

## Step 5: Monitor Execution

```bash
# Watch all workers
helixcode worker monitor

# Live dashboard:
# ┌─ worker-001 ────────────────────┐
# │ CPU: ████████░░ 80%             │
# │ RAM: ██████░░░░ 60%             │
# │ Task: Building Linux binary...  │
# │ Progress: 75%                   │
# └─────────────────────────────────┘
```

## Step 6: Collect Results

```bash
# Tasks complete automatically
# Results synced back to control machine

helixcode task results task-001

# Output:
# Task: Build Linux binary
# Status: Completed
# Duration: 2m 34s
# Artifacts:
#   - bin/app-linux-amd64
#   - bin/app-linux-arm64
```

## Step 7: Complex Workflow

```bash
# Distributed test execution
helixcode test --distributed \
  --workers worker-001,worker-002,worker-003

# HelixCode:
# 1. Splits test suite
# 2. Distributes to workers
# 3. Runs in parallel
# 4. Aggregates results

# Time: 45 minutes → 15 minutes (3x speedup)
```

## Step 8: GPU Acceleration

```bash
# Use GPU worker for AI tasks
helixcode generate --worker worker-002 \
  --model ollama/llama3:70b \
  --prompt "Generate complete user authentication system"

# Worker-002 uses local GPU for inference
# No API costs!
```

## Configuration

```yaml
workers:
  ssh:
    timeout: 30s
    max_retries: 3

  auto_install: true
  health_check_interval: 30s

  # Task assignment strategy
  assignment:
    strategy: "capability"  # capability, load, round_robin
    prefer_gpu: true
    prefer_local: false
```

## Results

- **Parallelization**: 3x+ faster builds and tests
- **Scalability**: Add workers as needed
- **Cost Efficiency**: Use existing infrastructure

---

Continue to [Tutorial 8: Using Plan Mode](Tutorial_8_Using_Plan_Mode.md)
