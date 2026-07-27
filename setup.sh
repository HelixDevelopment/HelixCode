#!/usr/bin/env bash
#
# HelixCode Setup — installs the ENTIRE Helix platform.
#
# This is the single entry point. It initialises submodules, installs system
# dependencies, builds every sub-system (HelixCode, HelixAgent, HelixLLM), and
# installs the systemd *user* units so the whole platform boots with the host
# and survives restarts.
#
# Usage:
#   ./setup.sh                 # install + build + install systemd units (no start)
#   ./setup.sh --start         # ... and start the platform immediately
#   ./setup.sh --no-systemd    # build only; skip systemd installation
#   ./setup.sh --skip-build    # wire systemd only; assume binaries already built
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${REPO_ROOT}"

DO_SYSTEMD=1
DO_BUILD=1
START_FLAG=""
for arg in "$@"; do
  case "$arg" in
    --start)      START_FLAG="--start" ;;
    --no-systemd) DO_SYSTEMD=0 ;;
    --skip-build) DO_BUILD=0 ;;
    -h|--help)    sed -n '2,16p' "$0"; exit 0 ;;
    *) echo "unknown option: $arg (try --help)" >&2; exit 2 ;;
  esac
done

section() { printf '\n\033[1m==> %s\033[0m\n' "$*"; }
ok()      { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn()    { printf '  \033[33m!\033[0m %s\n' "$*"; }
die()     { printf '  \033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

echo "=================================="
echo "  Helix Platform Setup"
echo "=================================="
echo "  root: ${REPO_ROOT}"

command -v git >/dev/null 2>&1 || die "git is not installed."
command -v go  >/dev/null 2>&1 || die "go is not installed (required to build all sub-systems)."

# --- 1. submodules -----------------------------------------------------------
section "Initialising git submodules"
./scripts/init-submodules.sh
ok "submodules ready"

# --- 2. git hooks ------------------------------------------------------------
section "Installing local git hooks (CONST-042 enforcement)"
./scripts/install-git-hooks.sh
ok "git hooks installed"

# --- 3. system dependencies --------------------------------------------------
section "Installing system dependencies"
case "$OSTYPE" in
  linux-gnu*)
    if [ -x ./install_missing_libs.sh ]; then
      ./install_missing_libs.sh
      ok "system libraries installed"
    else
      warn "install_missing_libs.sh not found or not executable — skipping"
    fi
    ;;
  darwin*)
    warn "macOS: ensure Xcode Command Line Tools are present (xcode-select --install)"
    ;;
  *)
    warn "unrecognised OS '$OSTYPE' — install dependencies manually"
    ;;
esac

# --- 4. build every sub-system ----------------------------------------------
# Each sub-system owns its own Makefile; setup.sh calls into them rather than
# duplicating build logic (§11.4.74 extend-don't-reimplement).
if [ "${DO_BUILD}" -eq 1 ]; then
  section "Building HelixCode (main application)"
  make -C helix_code build
  ok "helix_code/bin/helixcode"

  section "Building HelixAgent"
  if [ -f submodules/helix_agent/Makefile ]; then
    make -C submodules/helix_agent build
    ok "submodules/helix_agent/bin/helixagent"
  else
    warn "submodules/helix_agent not initialised — skipping (run scripts/init-submodules.sh)"
  fi

  section "Building LLMsVerifier"
  # The real Go module lives one level down at llms_verifier/llm-verifier/
  # (module digital.vasic.llmsverifier); the submodule root is a thin wrapper.
  if [ -f submodules/llms_verifier/llm-verifier/go.mod ]; then
    ( cd submodules/llms_verifier/llm-verifier && go build -o bin/llm-verifier ./cmd )
    ok "submodules/llms_verifier/llm-verifier/bin/llm-verifier"
  else
    warn "submodules/llms_verifier not initialised — skipping"
  fi

  section "Building HelixLLM"
  if [ -f submodules/helix_llm/Makefile ]; then
    make -C submodules/helix_llm build
    # The gateway unit resolves `helixllm` from PATH, so publish it to
    # ~/.local/bin rather than leaving it only in the submodule's bin/.
    mkdir -p "${HOME}/.local/bin"
    install -m 0755 submodules/helix_llm/bin/helixllm "${HOME}/.local/bin/helixllm"
    ok "~/.local/bin/helixllm"
    case ":${PATH}:" in
      *":${HOME}/.local/bin:"*) : ;;
      *) warn "~/.local/bin is not on PATH — add it to your shell profile" ;;
    esac
  else
    warn "submodules/helix_llm not initialised — skipping"
  fi
else
  section "Skipping builds (--skip-build)"
fi

# --- 5. secrets --------------------------------------------------------------
# Never generated with real values, never committed (CONST-042 / §12.1).
section "Checking environment file"
if [ ! -f .env ] && [ -f .env.example ]; then
  cp .env.example .env
  chmod 0600 .env
  warn "created .env from .env.example — fill in secrets (DB_PASSWORD, API keys) before starting"
elif [ -f .env ]; then
  chmod 0600 .env
  ok ".env present (mode 0600)"
else
  warn "no .env and no .env.example — services needing secrets may fail dependency verification"
fi

# --- 6. systemd --------------------------------------------------------------
if [ "${DO_SYSTEMD}" -eq 1 ]; then
  section "Installing systemd user units (boot-persistent)"
  ./scripts/install_systemd_units.sh ${START_FLAG}
else
  section "Skipping systemd installation (--no-systemd)"
fi

# --- done --------------------------------------------------------------------
cat <<EOF

==================================
  ✅ Setup complete
==================================

The platform is installed as systemd USER units and will start on boot.

  Start everything   : systemctl --user start helix.target
  Stop everything    : systemctl --user stop helix.target
  Status (one)       : systemctl --user status helixagent
  Status (all)       : systemctl --user list-units 'helix*'
  Logs               : journalctl --user -u helixagent -f

Services and ports:
  helixcode-server    :8081    HelixCode API
  helixllm-gateway    :8443    HelixLLM multi-provider router (TLS)
  helixagent          :7061    HelixAgent runtime
  llmsverifier        :8100    LLMsVerifier model/provider scoring API
  helixllm-coder      :18434   Local Qwen3-Coder model
  helixcode-infra     :5433 postgres  :6380 redis  :8083 weaviate
                      :8082 chromadb  :8000 cognee :6333 qdrant
                      :11434 ollama   :11211 memcached

EOF
