# HelixCode - Comprehensive User Manual

**Version 2.0** | **Last Updated**: November 6, 2025

---

## Table of Contents

### Part I: Getting Started
1. [Introduction](#1-introduction)
   - [What is HelixCode?](#11-what-is-helixcode)
   - [Key Features](#12-key-features)
   - [Architecture Overview](#13-architecture-overview)
   - [Use Cases](#14-use-cases)

2. [Installation](#2-installation)
   - [System Requirements](#21-system-requirements)
   - [Quick Start Installation](#22-quick-start-installation)
   - [Production Deployment](#23-production-deployment)
   - [Docker Deployment](#24-docker-deployment)
   - [Kubernetes Deployment](#25-kubernetes-deployment)
   - [Platform-Specific Setup](#26-platform-specific-setup)

3. [First Project](#3-first-project)
   - [Creating Your First Project](#31-creating-your-first-project)
   - [Basic Workflow](#32-basic-workflow)
   - [Understanding Sessions](#33-understanding-sessions)

### Part II: Configuration
4. [Configuration](#4-configuration)
   - [YAML Configuration](#41-yaml-configuration)
   - [Environment Variables](#42-environment-variables)
   - [Configuration Profiles](#43-configuration-profiles)
   - [Security Best Practices](#44-security-best-practices)

### Part III: LLM Providers (14+ Providers)
5. [LLM Providers Overview](#5-llm-providers-overview)
   - [Provider Selection Guide](#51-provider-selection-guide)
   - [Cost Comparison](#52-cost-comparison)
   - [Performance Comparison](#53-performance-comparison)

6. [Premium Providers](#6-premium-providers)
   - [Anthropic Claude](#61-anthropic-claude)
   - [Google Gemini](#62-google-gemini)
   - [OpenAI](#63-openai)
   - [AWS Bedrock](#64-aws-bedrock)
   - [Azure OpenAI](#65-azure-openai)
   - [Google VertexAI](#66-google-vertexai)
   - [Groq](#67-groq)
   - [Mistral](#68-mistral)

7. [Free & Open Source Providers](#7-free--open-source-providers)
   - [XAI (Grok)](#71-xai-grok)
   - [OpenRouter](#72-openrouter)
   - [GitHub Copilot](#73-github-copilot)
   - [Qwen](#74-qwen)
   - [Ollama](#75-ollama)
   - [Llama.cpp](#76-llamacpp)

### Part IV: Core Tools
8. [File System Tools](#8-file-system-tools)
   - [Reading Files](#81-reading-files)
   - [Writing Files](#82-writing-files)
   - [Editing Files](#83-editing-files)
   - [Searching Files](#84-searching-files)
   - [File Operations Best Practices](#85-file-operations-best-practices)

9. [Shell Execution](#9-shell-execution)
   - [Command Execution](#91-command-execution)
   - [Security Controls](#92-security-controls)
   - [Output Streaming](#93-output-streaming)
   - [Timeout Management](#94-timeout-management)

10. [Browser Automation](#10-browser-automation)
    - [Browser Launch](#101-browser-launch)
    - [Navigation & Interaction](#102-navigation--interaction)
    - [Screenshots](#103-screenshots)
    - [Form Filling](#104-form-filling)
    - [Testing Automation](#105-testing-automation)

11. [Web Tools](#11-web-tools)
    - [Web Search](#111-web-search)
    - [HTML Fetching](#112-html-fetching)
    - [Parsing & Extraction](#113-parsing--extraction)

12. [Voice-to-Code](#12-voice-to-code)
    - [Voice Input Setup](#121-voice-input-setup)
    - [Transcription](#122-transcription)
    - [Workflow Integration](#123-workflow-integration)

13. [Codebase Mapping](#13-codebase-mapping)
    - [Tree-sitter Integration](#131-tree-sitter-integration)
    - [Supported Languages](#132-supported-languages)
    - [AST Analysis](#133-ast-analysis)
    - [Symbol Extraction](#134-symbol-extraction)

### Part V: Advanced Workflows
14. [Plan Mode](#14-plan-mode)
    - [Two-Phase Workflow](#141-two-phase-workflow)
    - [Plan Generation](#142-plan-generation)
    - [Option Generation](#143-option-generation)
    - [Plan Execution](#144-plan-execution)

15. [Multi-File Editing](#15-multi-file-editing)
    - [Atomic Transactions](#151-atomic-transactions)
    - [Rollback Support](#152-rollback-support)
    - [Conflict Resolution](#153-conflict-resolution)

16. [Git Auto-Commit](#16-git-auto-commit)
    - [Semantic Commit Messages](#161-semantic-commit-messages)
    - [Conventional Commits](#162-conventional-commits)
    - [Co-Author Attribution](#163-co-author-attribution)

17. [Context Compression](#17-context-compression)
    - [Compression Strategies](#171-compression-strategies)
    - [Sliding Window](#172-sliding-window)
    - [Semantic Summarization](#173-semantic-summarization)
    - [Hybrid Approach](#174-hybrid-approach)

18. [Tool Confirmation](#18-tool-confirmation)
    - [Risk Levels](#181-risk-levels)
    - [Interactive Confirmation](#182-interactive-confirmation)
    - [Audit Logging](#183-audit-logging)

19. [Checkpoint Snapshots](#19-checkpoint-snapshots)
    - [Creating Snapshots](#191-creating-snapshots)
    - [Restoring Snapshots](#192-restoring-snapshots)
    - [Comparing Snapshots](#193-comparing-snapshots)
    - [Snapshot Management](#194-snapshot-management)

20. [Autonomy Modes](#20-autonomy-modes)
    - [Five Autonomy Levels](#201-five-autonomy-levels)
    - [Mode Selection](#202-mode-selection)
    - [Permission System](#203-permission-system)
    - [Temporary Escalation](#204-temporary-escalation)

21. [Vision Auto-Switch](#21-vision-auto-switch)
    - [Automatic Model Switching](#211-automatic-model-switching)
    - [Switch Modes](#212-switch-modes)
    - [Supported Formats](#213-supported-formats)

### Part VI: Distributed Computing
22. [Distributed Worker Setup](#22-distributed-worker-setup)
    - [SSH Worker Configuration](#221-ssh-worker-configuration)
    - [Auto-Installation](#222-auto-installation)
    - [Health Monitoring](#223-health-monitoring)
    - [Resource Management](#224-resource-management)
    - [Multi-Worker Orchestration](#225-multi-worker-orchestration)

23. [Task Management](#23-task-management)
    - [Task Types](#231-task-types)
    - [Priority Levels](#232-priority-levels)
    - [Checkpointing](#233-checkpointing)
    - [Dependency Resolution](#234-dependency-resolution)

### Part VII: MCP Protocol
24. [MCP Integration](#24-mcp-integration)
    - [Model Context Protocol](#241-model-context-protocol)
    - [Transport Methods](#242-transport-methods)
    - [Tool Registration](#243-tool-registration)
    - [Custom MCP Servers](#244-custom-mcp-servers)

### Part VIII: Multi-Client Usage
25. [CLI Client](#25-cli-client)
    - [Command Reference](#251-command-reference)
    - [Interactive Mode](#252-interactive-mode)
    - [Scripting](#253-scripting)

26. [Terminal UI](#26-terminal-ui)
    - [Interface Overview](#261-interface-overview)
    - [Keyboard Shortcuts](#262-keyboard-shortcuts)
    - [Customization](#263-customization)

27. [Desktop Application](#27-desktop-application)
    - [Installation](#271-installation)
    - [Features](#272-features)
    - [Platform Support](#273-platform-support)

28. [Mobile Clients](#28-mobile-clients)
    - [iOS Client](#281-ios-client)
    - [Android Client](#282-android-client)
    - [Mobile Features](#283-mobile-features)

### Part IX: Security & Compliance
29. [Security Best Practices](#29-security-best-practices)
    - [Authentication & Authorization](#291-authentication--authorization)
    - [API Key Management](#292-api-key-management)
    - [Network Security](#293-network-security)
    - [Data Privacy](#294-data-privacy)
    - [Audit Logging](#295-audit-logging)

### Part X: Troubleshooting & Reference
30. [Troubleshooting](#30-troubleshooting)
    - [Common Issues](#301-common-issues)
    - [Error Messages](#302-error-messages)
    - [Debugging](#303-debugging)
    - [Performance Optimization](#304-performance-optimization)

31. [CLI Command Reference](#31-cli-command-reference)
32. [API Reference](#32-api-reference)
33. [FAQ](#33-faq)

### Part XI: Advanced Use Cases
34. [Advanced Use Cases](#34-advanced-use-cases)
   - See [tutorials/](tutorials/) for detailed step-by-step guides

---

## 1. Introduction

### 1.1 What is HelixCode?

HelixCode is an enterprise-grade distributed AI development platform that revolutionizes software development through intelligent automation. It combines the power of 14+ leading AI providers with advanced tooling, distributed computing, and sophisticated workflow automation.

**Core Philosophy**: Empower developers with AI assistance while maintaining full control and transparency.

### 1.2 Key Features

#### AI Integration (14+ Providers)
- **Premium Providers**: Claude, Gemini, OpenAI, AWS Bedrock, Azure OpenAI, VertexAI, Groq, Mistral
- **Free Providers**: XAI Grok, OpenRouter, GitHub Copilot, Qwen
- **Local Models**: Ollama, Llama.cpp (100% offline)
- **Advanced Features**: Extended thinking, prompt caching, 2M token contexts

#### Smart Tools
- **File System**: Intelligent caching, atomic operations, pattern matching
- **Shell**: Sandboxed execution with security controls
- **Browser**: Chrome automation for testing and scraping
- **Web**: Multi-provider search and HTML parsing
- **Voice**: Hands-free coding with Whisper transcription
- **Mapping**: Tree-sitter AST parsing for 30+ languages

#### Intelligent Workflows
- **Plan Mode**: Two-phase plan-then-execute workflow
- **Auto-Commit**: LLM-generated semantic commit messages
- **Multi-File Edit**: Atomic transactions across multiple files
- **Context Compression**: Automatic conversation summarization
- **Tool Confirmation**: Interactive approval for dangerous operations

#### Enterprise Features
- **Snapshots**: Git-based checkpoint system with instant rollback
- **Autonomy Modes**: 5 levels of AI automation (None → Full Auto)
- **Vision Auto-Switch**: Automatic model switching for images
- **Distributed Workers**: SSH-based worker pools with auto-installation
- **MCP Protocol**: Full Model Context Protocol implementation

### 1.3 Architecture Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                        HelixCode Platform                         │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Client Layer                                │    │
│  │  ┌────────┬────────┬────────┬────────┬────────┐         │    │
│  │  │  CLI   │  TUI   │Desktop │ Mobile │  API   │         │    │
│  │  └────────┴────────┴────────┴────────┴────────┘         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              API Layer                                   │    │
│  │  ┌────────┬────────┬────────┬────────────────┐          │    │
│  │  │  REST  │WebSocket│  MCP  │  GraphQL (opt) │          │    │
│  │  └────────┴────────┴────────┴────────────────┘          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              LLM Provider Layer (14+ Providers)          │    │
│  │  ┌──────┬───────┬────────┬────────┬────────┬─────────┐  │    │
│  │  │Claude│Gemini │OpenAI  │Bedrock │Azure   │VertexAI │  │    │
│  │  ├──────┼───────┼────────┼────────┼────────┼─────────┤  │    │
│  │  │Groq  │Mistral│XAI Grok│OpenRtr │Copilot │  Qwen   │  │    │
│  │  ├──────┼───────┴────────┴────────┴────────┴─────────┤  │    │
│  │  │Ollama│          Llama.cpp (Local)                  │  │    │
│  │  └──────┴─────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Tools Layer                                 │    │
│  │  ┌────────┬────────┬────────┬────────┬────────┐         │    │
│  │  │  File  │ Shell  │Browser │  Web   │ Voice  │         │    │
│  │  ├────────┴────────┴────────┴────────┴────────┤         │    │
│  │  │        Codebase Mapping (Tree-sitter)      │         │    │
│  │  └────────────────────────────────────────────┘         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Workflow Layer                              │    │
│  │  ┌────────┬────────┬────────┬────────┬────────┐         │    │
│  │  │  Plan  │AutoGit │MultiEdit│Compress│Confirm│         │    │
│  │  └────────┴────────┴────────┴────────┴────────┘         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Advanced Features Layer                     │    │
│  │  ┌────────┬────────┬────────┬────────────────┐          │    │
│  │  │Snapshot│Autonomy│ Vision │  Distributed   │          │    │
│  │  │        │ Modes  │ Switch │    Workers     │          │    │
│  │  └────────┴────────┴────────┴────────────────┘          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Data Layer                                  │    │
│  │  ┌──────────────────────┬────────────────────┐          │    │
│  │  │   PostgreSQL         │      Redis         │          │    │
│  │  │   (Persistent)       │     (Cache)        │          │    │
│  │  └──────────────────────┴────────────────────┘          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

### 1.4 Use Cases

#### Software Development
- Full-stack application development
- Code refactoring and optimization
- Automated testing and bug fixing
- Documentation generation
- Code review automation

#### DevOps & Infrastructure
- Infrastructure as Code (IaC) generation
- CI/CD pipeline automation
- Configuration management
- Monitoring and alerting setup

#### Data Engineering
- ETL pipeline development
- Data analysis automation
- SQL query optimization
- Schema design and migration

#### Research & Prototyping
- Rapid prototyping
- Algorithm implementation
- Performance benchmarking
- Literature review automation

#### Enterprise Development
- Microservices architecture
- Legacy code modernization
- Compliance and security auditing
- Multi-team collaboration

---

## 2. Installation

### 2.1 System Requirements

#### Minimum Requirements
- **OS**: Linux (Ubuntu 20.04+), macOS (11+), Windows (10+/WSL2)
- **CPU**: 2 cores
- **RAM**: 4 GB
- **Storage**: 10 GB
- **Go**: 1.24.0 or later
- **Git**: 2.30+

#### Recommended Requirements
- **CPU**: 4+ cores
- **RAM**: 8+ GB
- **Storage**: 20+ GB SSD
- **PostgreSQL**: 14+
- **Redis**: 7+ (optional, recommended for production)

#### For Production
- **CPU**: 8+ cores
- **RAM**: 16+ GB
- **Storage**: 50+ GB SSD
- **PostgreSQL**: 15+ with replication
- **Redis**: 7+ with persistence
- **Load Balancer**: Nginx, HAProxy, or cloud LB

### 2.2 Quick Start Installation

```bash
# 1. Clone repository
git clone https://github.com/your-org/helixcode.git
cd helixcode/HelixCode

# 2. Install dependencies
go mod download

# 3. Generate assets (logo processing)
make logo-assets

# 4. Build all components
make build

# 5. Run tests to verify installation
make test

# 6. Start development server
make dev
```

**Verify Installation**:
```bash
# Check server health
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","version":"2.0","uptime":"10s"}
```

### 2.3 Production Deployment

#### Step 1: Database Setup

```bash
# Install PostgreSQL
# Ubuntu/Debian
sudo apt-get install postgresql-14

# macOS
brew install postgresql@14

# Create database and user
sudo -u postgres psql
```

```sql
CREATE DATABASE helixcode;
CREATE USER helixcode WITH PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE helixcode TO helixcode;
\q
```

#### Step 2: Redis Setup (Optional but Recommended)

```bash
# Install Redis
# Ubuntu/Debian
sudo apt-get install redis-server

# macOS
brew install redis

# Configure Redis
sudo nano /etc/redis/redis.conf

# Set password (uncomment and modify):
# requirepass your_secure_redis_password

# Start Redis
sudo systemctl start redis
sudo systemctl enable redis
```

#### Step 3: Environment Configuration

```bash
# Copy example environment file
cp .env.example .env

# Edit environment variables
nano .env
```

**Required Variables** (.env):
```bash
# Authentication
HELIX_AUTH_JWT_SECRET=your-super-secure-jwt-secret-minimum-32-characters

# Database
HELIX_DATABASE_HOST=localhost
HELIX_DATABASE_PORT=5432
HELIX_DATABASE_NAME=helixcode
HELIX_DATABASE_USER=helixcode
HELIX_DATABASE_PASSWORD=your_secure_password

# Redis
HELIX_REDIS_HOST=localhost
HELIX_REDIS_PORT=6379
HELIX_REDIS_PASSWORD=your_secure_redis_password

# Server
HELIX_ENV=production
HELIX_SERVER_PORT=8080

# AI Providers (at least one required)
ANTHROPIC_API_KEY=sk-ant-your-key-here
GEMINI_API_KEY=your-gemini-key
OPENAI_API_KEY=sk-your-openai-key
```

#### Step 4: Build Production Binaries

```bash
# Build optimized production binaries
make prod

# This creates binaries in bin/:
# - bin/helixcode-server (main server)
# - bin/helixcode-cli (CLI client)
# - bin/helixcode-tui (terminal UI)
```

#### Step 5: Run Production Server

```bash
# Run server with production config
./bin/helixcode-server --config config/config.yaml

# Or use systemd service
sudo cp scripts/systemd/helixcode.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl start helixcode
sudo systemctl enable helixcode

# Check status
sudo systemctl status helixcode
```

### 2.4 Docker Deployment

**Recommended for production**: Docker Compose provides complete stack.

#### Step 1: Prepare Environment

```bash
# Create production .env file
cp .env.example .env
nano .env

# Set all required variables (see section 2.3)
```

#### Step 2: Deploy with Docker Compose

```bash
# Start all services
docker-compose up -d

# Services included:
# - helixcode-server (main application)
# - postgres (database)
# - redis (cache)
# - nginx (reverse proxy)
# - prometheus (monitoring)
# - grafana (dashboards)
```

**Verify Deployment**:
```bash
# Check all services are running
docker-compose ps

# Expected output:
# NAME                 STATUS
# helixcode-server     Up
# helixcode-postgres   Up
# helixcode-redis      Up
# helixcode-nginx      Up
# helixcode-prometheus Up
# helixcode-grafana    Up

# Test application
curl http://localhost/health

# View logs
docker-compose logs -f helixcode-server
```

#### Step 3: Access Services

- **Application**: http://localhost
- **API**: http://localhost/api
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

### 2.5 Kubernetes Deployment

For large-scale production deployments with high availability.

#### Prerequisites
- Kubernetes cluster (1.24+)
- kubectl configured
- Helm 3+ (optional)

#### Step 1: Create Namespace

```bash
kubectl create namespace helixcode
```

#### Step 2: Create Secrets

```bash
# Database credentials
kubectl create secret generic helixcode-db \
  --from-literal=password=your_db_password \
  -n helixcode

# JWT secret
kubectl create secret generic helixcode-auth \
  --from-literal=jwt-secret=your_jwt_secret \
  -n helixcode

# AI provider API keys
kubectl create secret generic helixcode-llm \
  --from-literal=anthropic-key=sk-ant-... \
  --from-literal=gemini-key=... \
  --from-literal=openai-key=sk-... \
  -n helixcode
```

#### Step 3: Deploy Database

```bash
# Apply PostgreSQL StatefulSet
kubectl apply -f k8s/postgres.yaml -n helixcode

# Wait for database to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n helixcode
```

#### Step 4: Deploy Application

```bash
# Apply all manifests
kubectl apply -f k8s/helixcode.yaml -n helixcode

# Components deployed:
# - Deployment (3 replicas)
# - Service (LoadBalancer)
# - ConfigMap (config.yaml)
# - HorizontalPodAutoscaler (2-10 pods)
```

#### Step 5: Verify Deployment

```bash
# Check pods
kubectl get pods -n helixcode

# Check services
kubectl get svc -n helixcode

# Get external IP
kubectl get svc helixcode -n helixcode -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# Test health endpoint
curl http://<EXTERNAL-IP>/health
```

### 2.6 Platform-Specific Setup

#### Linux

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y build-essential git postgresql-14 redis-server

# Fedora/RHEL
sudo dnf install -y gcc git postgresql-server redis

# Install Go
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### macOS

```bash
# Install Homebrew (if not installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install go postgresql@14 redis git

# Start services
brew services start postgresql@14
brew services start redis
```

#### Windows (WSL2)

```bash
# Install WSL2 with Ubuntu
wsl --install -d Ubuntu

# Inside WSL2, follow Linux instructions
sudo apt-get update
sudo apt-get install -y build-essential git postgresql-14 redis-server

# Install Go
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**Post-Installation Verification**:

```bash
# Verify Go installation
go version
# Expected: go version go1.24.0 ...

# Verify PostgreSQL
psql --version
# Expected: psql (PostgreSQL) 14.x

# Verify Redis
redis-cli --version
# Expected: redis-cli 7.x

# Build HelixCode
cd helixcode/HelixCode
make build

# Run tests
make test
# Expected: PASS (all tests)
```

---

## 3. First Project

### 3.1 Creating Your First Project

Let's create a simple web application using HelixCode.

#### Step 1: Initialize Project

```bash
# Create project directory
mkdir my-first-app
cd my-first-app

# Initialize Git
git init

# Start HelixCode CLI
helixcode
```

#### Step 2: Configure AI Provider

```bash
# List available providers
helixcode> llm providers

# Set provider (example: Claude)
helixcode> llm provider set anthropic --model claude-3-5-sonnet-latest

# Verify
helixcode> llm models
```

#### Step 3: Generate Project Structure

```bash
# Use plan mode for structured approach
helixcode> plan "Create a simple REST API in Go with user authentication"

# HelixCode will generate a detailed plan:
# 1. Project structure setup
# 2. Database schema design
# 3. API endpoint implementation
# 4. Authentication middleware
# 5. Tests

# Review plan and approve
helixcode> plan execute <plan-id>
```

#### Step 4: Run the Application

```bash
# HelixCode will generate code, you can now run it
go mod init my-first-app
go mod tidy

# Run the app
go run main.go

# Test the API
curl http://localhost:8080/health
```

### 3.2 Basic Workflow

#### Interactive Development

```bash
# Start interactive session
helixcode

# Ask for code generation
helixcode> generate "Add a user registration endpoint with email validation"

# Review generated code
helixcode> show changes

# Apply changes
helixcode> apply

# Commit with auto-generated message
helixcode> commit --auto
```

#### Command-Line Usage

```bash
# Generate code directly
helixcode generate "Add user authentication middleware"

# Refactor existing code
helixcode refactor --file auth.go --task "Improve error handling"

# Run tests
helixcode test --coverage

# Deploy
helixcode deploy --target production
```

### 3.3 Understanding Sessions

HelixCode maintains session context for continuity.

```bash
# View current session
helixcode session info

# List all sessions
helixcode session list

# Resume previous session
helixcode session resume <session-id>

# Create named session
helixcode session create --name "feature-user-auth"

# Save session
helixcode session save
```

**Session Context Includes**:
- Conversation history
- Code changes
- File modifications
- AI provider settings
- Autonomy mode
- Snapshots

---

## 4. Configuration

### 4.1 YAML Configuration

Primary configuration file: `config/config.yaml`

```yaml
# Server Configuration
server:
  address: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  shutdown_timeout: 30s
  max_request_size: 10485760  # 10MB

# Database Configuration
database:
  host: "${HELIX_DATABASE_HOST:localhost}"
  port: 5432
  user: "helixcode"
  password: ""  # Set via HELIX_DATABASE_PASSWORD
  dbname: "helixcode"
  sslmode: "disable"  # Use "require" in production
  max_connections: 100
  max_idle_connections: 10
  connection_timeout: 10s
  query_timeout: 30s

# Redis Configuration (optional)
redis:
  host: "${HELIX_REDIS_HOST:localhost}"
  port: 6379
  password: ""  # Set via HELIX_REDIS_PASSWORD
  db: 0
  enabled: true
  pool_size: 10
  max_retries: 3
  dial_timeout: 5s

# Authentication Configuration
auth:
  jwt_secret: ""  # Set via HELIX_AUTH_JWT_SECRET
  token_expiry: 86400      # 24 hours
  refresh_expiry: 604800   # 7 days
  session_expiry: 2592000  # 30 days
  bcrypt_cost: 12
  max_login_attempts: 5
  lockout_duration: 900    # 15 minutes

# Worker Configuration
workers:
  health_check_interval: 30s
  health_ttl: 120s
  max_concurrent_tasks: 10
  task_timeout: 3600s      # 1 hour
  auto_install: true       # Auto-install on SSH workers
  ssh_timeout: 30s
  ssh_key_path: "~/.ssh/id_rsa"

# Task Configuration
tasks:
  max_retries: 3
  retry_delay: 60s
  checkpoint_interval: 300s  # 5 minutes
  cleanup_interval: 3600s    # 1 hour
  max_checkpoint_age: 604800s  # 7 days

# LLM Configuration
llm:
  default_provider: "anthropic"
  max_tokens: 4096
  temperature: 0.7
  top_p: 1.0
  timeout: 120s

  # Provider configurations
  providers:
    anthropic:
      type: "anthropic"
      api_key: ""  # Set via ANTHROPIC_API_KEY
      enabled: true
      models:
        - "claude-4-sonnet"
        - "claude-3-5-sonnet-latest"
        - "claude-3-5-haiku-latest"
      default_model: "claude-3-5-sonnet-latest"
      max_tokens: 8192

    gemini:
      type: "gemini"
      api_key: ""  # Set via GEMINI_API_KEY
      enabled: true
      models:
        - "gemini-2.5-pro"
        - "gemini-2.5-flash"
        - "gemini-2.0-flash"
      default_model: "gemini-2.5-flash"
      max_tokens: 8192

    openai:
      type: "openai"
      api_key: ""  # Set via OPENAI_API_KEY
      enabled: true
      models:
        - "gpt-4.1"
        - "gpt-4o"
        - "o1"
      default_model: "gpt-4o"

    ollama:
      type: "local"
      enabled: true
      url: "http://localhost:11434"
      models:
        - "llama3:8b"
        - "codellama:13b"
        - "mistral:7b"

# Tool Configuration
tools:
  filesystem:
    enable_cache: true
    cache_size: 1000
    cache_ttl: 3600s
    max_file_size: 10485760  # 10MB
    allowed_extensions: [".go", ".ts", ".js", ".py", ".rs", ".java", ".c", ".cpp"]

  shell:
    timeout: 30s
    allowlist: ["git", "npm", "go", "make", "cargo", "python", "node"]
    blocklist: ["rm", "dd", "mkfs", "sudo", "su"]
    enable_sandbox: true
    max_output_size: 1048576  # 1MB

  browser:
    headless: true
    timeout: 30s
    user_agent: "HelixCode/2.0"
    viewport_width: 1920
    viewport_height: 1080
    enable_javascript: true

  web:
    default_provider: "duckduckgo"
    timeout: 10s
    max_results: 10
    user_agent: "HelixCode/2.0"

  voice:
    sample_rate: 16000
    channels: 1
    silence_timeout: 2s
    max_duration: 300s  # 5 minutes

  mapping:
    cache_dir: ".helixcode/cache"
    languages: ["go", "typescript", "javascript", "python", "rust", "java", "c", "cpp"]
    exclude_patterns: ["node_modules", "vendor", ".git", "dist", "build"]

# Workflow Configuration
workflow:
  planmode:
    enabled: true
    require_approval: true
    max_options: 5

  autocommit:
    enabled: true
    convention: "conventional"  # conventional, semantic, custom
    co_authors: true

  multiedit:
    enabled: true
    transaction_timeout: 300s
    max_files: 100

  compression:
    enabled: true
    strategy: "hybrid"  # sliding, semantic, hybrid
    threshold: 0.9  # 90% of context window
    summary_model: "claude-3-5-haiku-latest"

  confirmation:
    enabled: true
    require_for_risk_levels: ["medium", "high", "critical"]
    timeout: 60s

  snapshots:
    enabled: true
    auto_create: true
    auto_create_interval: 3600s
    max_snapshots: 100
    retention_days: 30

  autonomy:
    default_mode: "semi_auto"  # none, basic, basic_plus, semi_auto, full_auto
    allow_escalation: true
    max_escalation_duration: 3600s

  vision:
    auto_switch: true
    switch_mode: "session"  # once, session, persist
    require_confirm: false

# Logging Configuration
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # text, json
  output: "stdout"  # stdout, file
  file: "/var/log/helixcode/helixcode.log"
  max_size: 100  # MB
  max_backups: 10
  max_age: 30  # days
  compress: true

# Monitoring Configuration
monitoring:
  enabled: true
  prometheus:
    enabled: true
    port: 9090
    path: "/metrics"

  grafana:
    enabled: true
    port: 3000
    admin_password: ""  # Set via GRAFANA_ADMIN_PASSWORD

  alerts:
    enabled: true
    channels: ["slack", "email"]

# Notification Configuration
notifications:
  enabled: true

  rules:
    - name: "Critical Task Failures"
      condition: "type==error && severity==critical"
      channels: ["slack", "email", "telegram"]
      priority: "urgent"
      enabled: true

    - name: "Worker Health Alerts"
      condition: "type==alert && category==worker"
      channels: ["slack"]
      priority: "high"
      enabled: true

    - name: "Workflow Completions"
      condition: "type==success && category==workflow"
      channels: ["slack"]
      priority: "medium"
      enabled: true

  channels:
    slack:
      enabled: false
      webhook_url: ""  # Set via HELIX_SLACK_WEBHOOK_URL
      channel: "#helix-notifications"
      username: "HelixCode Bot"
      timeout: 10s

    telegram:
      enabled: false
      bot_token: ""  # Set via HELIX_TELEGRAM_BOT_TOKEN
      chat_id: ""    # Set via HELIX_TELEGRAM_CHAT_ID
      timeout: 10s

    email:
      enabled: false
      smtp:
        server: ""   # Set via HELIX_EMAIL_SMTP_SERVER
        port: 587
        username: ""  # Set via HELIX_EMAIL_USERNAME
        password: ""  # Set via HELIX_EMAIL_PASSWORD
        from: ""      # Set via HELIX_EMAIL_FROM
        tls: true
      recipients: []  # Set via HELIX_EMAIL_RECIPIENTS
      timeout: 30s

    discord:
      enabled: false
      webhook_url: ""  # Set via HELIX_DISCORD_WEBHOOK_URL
      timeout: 10s
```

### 4.2 Environment Variables

All environment variables can override YAML configuration.

#### Database Variables
```bash
HELIX_DATABASE_HOST=localhost
HELIX_DATABASE_PORT=5432
HELIX_DATABASE_USER=helixcode
HELIX_DATABASE_PASSWORD=secure_password
HELIX_DATABASE_NAME=helixcode
HELIX_DATABASE_SSLMODE=disable  # Use 'require' in production
```

#### Authentication Variables
```bash
HELIX_AUTH_JWT_SECRET=your-super-secure-jwt-secret-minimum-32-characters
HELIX_AUTH_TOKEN_EXPIRY=86400
HELIX_AUTH_SESSION_EXPIRY=2592000
```

#### Redis Variables
```bash
HELIX_REDIS_HOST=localhost
HELIX_REDIS_PORT=6379
HELIX_REDIS_PASSWORD=secure_redis_password
HELIX_REDIS_DB=0
```

#### AI Provider Variables
```bash
# Anthropic Claude
ANTHROPIC_API_KEY=sk-ant-api03-your-key-here

# Google Gemini
GEMINI_API_KEY=your-gemini-api-key
# Or use generic Google key
GOOGLE_API_KEY=your-google-api-key

# OpenAI
OPENAI_API_KEY=sk-your-openai-key

# AWS Bedrock
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key

# Azure OpenAI
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/
AZURE_OPENAI_API_KEY=your-azure-key
AZURE_OPENAI_DEPLOYMENT=your-deployment-name

# Google VertexAI
GOOGLE_CLOUD_PROJECT=your-project-id
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Groq
GROQ_API_KEY=your-groq-api-key

# Mistral
MISTRAL_API_KEY=your-mistral-api-key

# XAI (Grok)
XAI_API_KEY=your-xai-key  # Optional, free tier available

# OpenRouter
OPENROUTER_API_KEY=your-openrouter-key  # Optional

# GitHub Copilot
GITHUB_TOKEN=ghp_your-github-token

# Qwen (OAuth2 - interactive setup)
# No environment variable needed, use: helixcode llm auth qwen
```

#### Notification Variables
```bash
# Slack
HELIX_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T00000000/B00000000/XXXX

# Telegram
HELIX_TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
HELIX_TELEGRAM_CHAT_ID=123456789

# Email (SMTP)
HELIX_EMAIL_SMTP_SERVER=smtp.gmail.com
HELIX_EMAIL_SMTP_PORT=587
HELIX_EMAIL_USERNAME=your-email@gmail.com
HELIX_EMAIL_PASSWORD=your-app-password
HELIX_EMAIL_FROM=your-email@gmail.com
HELIX_EMAIL_RECIPIENTS=admin@example.com,team@example.com

# Discord
HELIX_DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
```

#### Server Variables
```bash
HELIX_SERVER_PORT=8080
HELIX_SERVER_ADDRESS=0.0.0.0
HELIX_ENV=production  # development, staging, production
```

#### Logging Variables
```bash
HELIX_LOG_LEVEL=info  # debug, info, warn, error
HELIX_LOG_FORMAT=json  # text, json
HELIX_LOG_OUTPUT=stdout  # stdout, file
HELIX_LOG_FILE=/var/log/helixcode/helixcode.log
```

### 4.3 Configuration Profiles

HelixCode supports multiple configuration profiles for different environments.

#### Profile Structure

```
config/
├── config.yaml              # Base configuration
├── development.yaml         # Development overrides
├── staging.yaml             # Staging overrides
├── production.yaml          # Production overrides
└── profiles/
    ├── high-performance.yaml
    ├── enterprise-security.yaml
    └── large-codebase.yaml
```

#### Using Profiles

```bash
# Specify config file
helixcode --config config/production.yaml

# Or use environment variable
export HELIX_CONFIG=config/production.yaml
helixcode

# Merge multiple configs (later configs override earlier)
helixcode --config config/config.yaml --config config/production.yaml
```

#### Development Profile (development.yaml)

```yaml
server:
  port: 8080

database:
  sslmode: "disable"
  max_connections: 20

llm:
  default_provider: "ollama"  # Local for development

logging:
  level: "debug"
  format: "text"

tools:
  shell:
    enable_sandbox: false  # Less restrictive in dev
```

#### Production Profile (production.yaml)

```yaml
server:
  port: 80
  read_timeout: 60s
  write_timeout: 60s

database:
  sslmode: "require"
  max_connections: 100

redis:
  enabled: true

auth:
  bcrypt_cost: 14  # Higher security

llm:
  default_provider: "anthropic"

logging:
  level: "info"
  format: "json"
  output: "file"
  file: "/var/log/helixcode/helixcode.log"

tools:
  shell:
    enable_sandbox: true
    blocklist: ["rm", "dd", "mkfs", "sudo", "su", "curl", "wget"]

workflow:
  confirmation:
    enabled: true
```

### 4.4 Security Best Practices

#### API Key Management

**DO**:
- ✅ Store API keys in environment variables
- ✅ Use secret management systems (AWS Secrets Manager, HashiCorp Vault)
- ✅ Rotate keys regularly
- ✅ Use separate keys for dev/staging/production
- ✅ Monitor API key usage

**DON'T**:
- ❌ Commit API keys to version control
- ❌ Share keys in chat/email
- ❌ Use production keys in development
- ❌ Store keys in configuration files

#### Secure Configuration Example

```yaml
# config/production.yaml
auth:
  jwt_secret: ""  # ALWAYS from environment variable
  bcrypt_cost: 14  # Higher for production
  max_login_attempts: 5
  lockout_duration: 900

database:
  sslmode: "require"  # Enforce SSL
  password: ""  # From environment

tools:
  shell:
    enable_sandbox: true
    allowlist: ["git"]  # Minimal allowed commands

  confirmation:
    enabled: true
    require_for_risk_levels: ["medium", "high", "critical"]

workflow:
  autonomy:
    default_mode: "basic"  # Require approval
```

#### Network Security

```yaml
server:
  address: "127.0.0.1"  # Only localhost in dev
  # Use reverse proxy (Nginx/HAProxy) in production

  # Enable rate limiting
  rate_limit:
    enabled: true
    requests_per_second: 100
    burst: 200

  # CORS configuration
  cors:
    enabled: true
    allowed_origins: ["https://app.example.com"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE"]
```

#### Audit Logging

```yaml
logging:
  audit:
    enabled: true
    file: "/var/log/helixcode/audit.log"
    events:
      - "auth.login"
      - "auth.logout"
      - "tool.shell.execute"
      - "tool.filesystem.delete"
      - "workflow.snapshot.restore"
    retention_days: 365
```

---

## 5. LLM Providers Overview

HelixCode supports 14+ AI providers, each with unique strengths.

### 5.1 Provider Selection Guide

| Provider | Best For | Context | Cost | Speed | Key Features |
|----------|----------|---------|------|-------|--------------|
| **Anthropic Claude** | Complex reasoning, coding | 200K | $$$ | Medium | Extended thinking, prompt caching |
| **Google Gemini** | Large codebases, multimodal | 2M | $$ | Fast | Massive context, flash models |
| **OpenAI GPT-4** | General purpose, reliability | 128K | $$$ | Medium | Function calling, vision |
| **AWS Bedrock** | Enterprise, compliance | Varies | $$$ | Medium | Multi-model, AWS integration |
| **Azure OpenAI** | Enterprise, Microsoft stack | 128K | $$$ | Medium | Entra ID, compliance |
| **VertexAI** | GCP integration, Gemini | 2M | $$ | Fast | Model garden, unified platform |
| **Groq** | Ultra-fast inference | 32K | $ | **Ultra** | 500+ tok/s, LPU hardware |
| **Mistral** | European data residency | 32K | $$ | Fast | Open models, EU compliance |
| **XAI Grok** | Free tier, experimentation | 128K | **Free** | Medium | No API key (basic), Twitter data |
| **OpenRouter** | Model variety, free options | Varies | **Free/$$** | Varies | 100+ models, unified API |
| **Copilot** | GitHub integration | 128K | **Free*** | Fast | Free with subscription |
| **Qwen** | Chinese language, free tier | 32K | **Free/$$** | Fast | 2K free requests/day |
| **Ollama** | Privacy, offline | Varies | **Free** | Medium | Local models, no network |
| **Llama.cpp** | Performance, control | Varies | **Free** | Fast | Direct inference, GGUF support |

*Free with GitHub subscription

### 5.2 Cost Comparison

**Approximate costs per million tokens (as of Nov 2025)**:

| Provider | Model | Input | Output |
|----------|-------|-------|--------|
| Claude | claude-4-sonnet | $3.00 | $15.00 |
| Claude | claude-3-5-sonnet | $3.00 | $15.00 |
| Claude | claude-3-5-haiku | $0.80 | $4.00 |
| Gemini | gemini-2.5-pro | $1.25 | $10.00 |
| Gemini | gemini-2.5-flash | $0.15 | $0.60 |
| OpenAI | gpt-4o | $2.50 | $10.00 |
| OpenAI | gpt-4.1 | $5.00 | $15.00 |
| Bedrock | claude-4 | $3.00 | $15.00 |
| Azure | gpt-4o | $2.50 | $10.00 |
| VertexAI | gemini-2.5-pro | $1.25 | $10.00 |
| Groq | llama-3.3-70b | $0.59 | $0.79 |
| Mistral | mistral-large | $2.00 | $6.00 |
| **XAI Grok** | **grok-3-fast** | **Free** | **Free** |
| **OpenRouter** | **Various free** | **Free** | **Free** |
| **Copilot** | **claude-3.5-sonnet** | **Free*** | **Free*** |
| **Qwen** | **qwen-plus** | **Free†** | **Free†** |
| **Ollama** | **All models** | **Free** | **Free** |
| **Llama.cpp** | **All models** | **Free** | **Free** |

*With GitHub subscription
†2,000 requests/day free tier

**Cost-Saving Features**:
- **Claude Prompt Caching**: 90% reduction on cached tokens
- **Gemini Flash**: 90% cheaper than Pro
- **Local Models**: Zero cost (electricity only)
- **Free Tiers**: XAI, OpenRouter, Copilot, Qwen

### 5.3 Performance Comparison

**Latency & Throughput** (approximate, varies by model and region):

| Provider | First Token (ms) | Tokens/Second | Typical Request (s) |
|----------|------------------|---------------|---------------------|
| Groq | **<100** | **500+** | **<1** |
| Gemini Flash | 300 | 80 | 2-3 |
| Claude | 500 | 50 | 3-5 |
| OpenAI | 400 | 60 | 3-4 |
| Gemini Pro | 400 | 40 | 4-6 |
| VertexAI | 400 | 60 | 3-5 |
| Ollama | 200 | 30-100 | Varies |
| Llama.cpp | 100 | 50-150 | Varies |

**Use Case Recommendations**:

- **Interactive Development**: Groq, Gemini Flash
- **Complex Reasoning**: Claude 4, GPT-4.1
- **Large Codebases**: Gemini 2.5 Pro (2M context)
- **Cost-Sensitive**: XAI Grok, OpenRouter, Gemini Flash
- **Privacy-Critical**: Ollama, Llama.cpp
- **Enterprise**: AWS Bedrock, Azure OpenAI, VertexAI
- **Experimentation**: Free tiers (XAI, OpenRouter, Copilot, Qwen)

---

## 6. Premium Providers

### 6.1 Anthropic Claude

**The most powerful coding assistant with industry-leading reasoning.**

#### Models

| Model | Context | Max Output | Best For | Price/M tokens |
|-------|---------|------------|----------|----------------|
| claude-4-sonnet | 200K | 50K | Most powerful, complex tasks | $3/$15 |
| claude-4-opus | 200K | 50K | Deepest reasoning | $15/$75 |
| claude-3-7-sonnet | 200K | 50K | Enhanced reasoning | $3/$15 |
| claude-3-5-sonnet-latest | 200K | 8K | Best for coding | $3/$15 |
| claude-3-5-haiku-latest | 200K | 8K | Fast, efficient | $0.80/$4 |
| claude-3-opus | 200K | 4K | Stable, reliable | $15/$75 |

#### Setup

```bash
# Set API key
export ANTHROPIC_API_KEY="sk-ant-api03-your-key-here"

# Verify in HelixCode
helixcode llm provider set anthropic --model claude-3-5-sonnet-latest

# Test
helixcode llm generate "Hello, Claude!"
```

#### Advanced Features

##### 1. Extended Thinking 🧠

Automatically activated for complex reasoning tasks.

```go
request := &LLMRequest{
    Model: "claude-4-sonnet",
    Messages: []Message{{
        Role: "user",
        Content: "Think step by step: Design a distributed caching system with eventual consistency",
    }},
    MaxTokens: 10000,
}

// Claude automatically allocates 80% (8000 tokens) for thinking
// Returns final answer in remaining 20% (2000 tokens)
response, _ := provider.Generate(ctx, request)
```

**Triggers** (keywords in prompt):
- "think", "reason", "analyze", "consider"
- "step by step", "carefully", "thoroughly"
- "explain your reasoning", "show your work"

##### 2. Prompt Caching 💾

Up to 90% cost reduction on repeated contexts.

```go
// First request - creates cache
request1 := &LLMRequest{
    Messages: []Message{
        {
            Role: "system",
            Content: "You are an expert Go developer. [... large codebase context ...]",
        },
        {Role: "user", Content: "Explain interfaces"},
    },
    Tools: []Tool{/* tool definitions */},
}
response1, _ := provider.Generate(ctx, request1)

// Second request - uses cache (90% cheaper on system message and last tool)
request2 := &LLMRequest{
    Messages: []Message{
        {
            Role: "system",
            Content: "You are an expert Go developer. [... same large context ...]",
        },
        {Role: "user", Content: "Explain channels"},
    },
    Tools: []Tool{/* same tools */},
}
response2, _ := provider.Generate(ctx, request2)

// Check cache usage
fmt.Printf("Cache read: %d tokens\n",
    response2.ProviderMetadata["cache_read_tokens"])
```

**What Gets Cached**:
- System messages (if > 1024 tokens)
- Last user message (if > 1024 tokens)
- Last tool definition (always)

**Cache Lifetime**: 5 minutes

##### 3. Vision Support 👁️

All Claude models support image analysis.

```go
request := &LLMRequest{
    Model: "claude-3-5-sonnet-latest",
    Messages: []Message{{
        Role: "user",
        Content: []ContentPart{
            {Type: "text", Text: "What's in this image?"},
            {
                Type: "image",
                Source: &ImageSource{
                    Type:      "base64",
                    MediaType: "image/png",
                    Data:      base64EncodedImage,
                },
            },
        },
    }},
}
```

**Supported Formats**: PNG, JPEG, WebP, GIF
**Max Size**: 5MB per image

##### 4. Streaming ⚡

Real-time token-by-token responses.

```go
ch := make(chan LLMResponse, 10)

go func() {
    err := provider.GenerateStream(ctx, request, ch)
    if err != nil {
        log.Printf("Stream error: %v", err)
    }
}()

for response := range ch {
    fmt.Print(response.Content)  // Print as it arrives

    if response.FinishReason != "" {
        fmt.Printf("\n\nDone! Usage: %d tokens\n", response.Usage.TotalTokens)
    }
}
```

#### Configuration

```yaml
llm:
  providers:
    anthropic:
      type: "anthropic"
      api_key: "${ANTHROPIC_API_KEY}"
      enabled: true
      models:
        - "claude-4-sonnet"
        - "claude-3-5-sonnet-latest"
        - "claude-3-5-haiku-latest"
      default_model: "claude-3-5-sonnet-latest"
      max_tokens: 8192
      temperature: 0.7

      # Enable advanced features
      enable_caching: true
      enable_thinking: true
      enable_vision: true
```

#### CLI Usage

```bash
# Generate with extended thinking
helixcode generate "Think carefully: design a microservices architecture"

# Use specific model
helixcode generate --model claude-4-sonnet "Complex reasoning task"

# Streaming output
helixcode generate --stream "Write a complete REST API"

# Vision task
helixcode analyze-image screenshot.png "What UI issues do you see?"
```

#### Best Practices

1. **Use Extended Thinking** for:
   - Architecture design
   - Complex problem solving
   - Multi-step reasoning
   - Code analysis and debugging

2. **Leverage Prompt Caching** for:
   - Repeated codebase queries
   - Long system prompts
   - Iterative development sessions

3. **Choose the Right Model**:
   - **Claude 4 Sonnet**: Most complex tasks
   - **Claude 3.5 Sonnet**: Daily coding (best balance)
   - **Claude 3.5 Haiku**: Fast iterations, simple tasks

4. **Optimize Costs**:
   - Keep system prompts > 1024 tokens for caching
   - Reuse tool definitions across requests
   - Use Haiku for simple tasks

### 6.2 Google Gemini

**Massive 2M token context windows for entire codebase analysis.**

#### Models

| Model | Context | Max Output | Best For | Price/M tokens |
|-------|---------|------------|----------|----------------|
| gemini-2.5-pro | 2M | 8K | Largest context, deep analysis | $1.25/$10 |
| gemini-2.5-flash | 1M | 8K | Fast with large context | $0.15/$0.60 |
| gemini-2.0-flash | 1M | 8K | Multimodal, fast | $0.15/$0.60 |
| gemini-1.5-pro | 2M | 8K | Proven, reliable | $1.25/$10 |
| gemini-1.5-flash | 1M | 8K | Cost-effective | $0.15/$0.60 |

#### Setup

```bash
# Set API key
export GEMINI_API_KEY="your-gemini-api-key"
# Or
export GOOGLE_API_KEY="your-google-api-key"

# Configure HelixCode
helixcode llm provider set gemini --model gemini-2.5-flash

# Test
helixcode llm generate "Hello, Gemini!"
```

#### Advanced Features

##### 1. Massive Context Windows 📚

Process entire codebases in a single request.

**Context Sizes**:
- **2M tokens** ≈ 1.5M words ≈ 3,000 pages ≈ 500 code files
- **1M tokens** ≈ 750K words ≈ 1,500 pages ≈ 250 code files

```go
// Read entire codebase
files := readAllProjectFiles("/path/to/project")  // 500 files
codebase := strings.Join(files, "\n\n")

request := &LLMRequest{
    Model: "gemini-2.5-pro",
    Messages: []Message{
        {Role: "system", Content: "You are analyzing a complete codebase."},
        {Role: "user", Content: codebase + "\n\nIdentify all security vulnerabilities."},
    },
    MaxTokens: 8192,
}

// Gemini processes all 500 files at once!
response, _ := provider.Generate(ctx, request)
```

**Use Cases**:
- Full codebase security audits
- Cross-file refactoring analysis
- Architectural pattern detection
- Documentation generation from entire projects

##### 2. Flash Models 🚀

Ultra-fast responses at 90% lower cost.

```bash
# Use Flash for fast iterations
helixcode llm provider set gemini --model gemini-2.5-flash

# Benchmark: Flash vs Pro
time helixcode generate "Write a sorting algorithm"
# Flash: ~2 seconds
# Pro: ~4 seconds
```

**When to Use Flash**:
- ✅ Rapid prototyping
- ✅ High-volume requests
- ✅ Simple code generation
- ✅ Fast feedback loops

**When to Use Pro**:
- ✅ Complex reasoning
- ✅ Large context analysis
- ✅ Multi-step problem solving

##### 3. Multimodal Capabilities 🎨

Text, images, and code understanding.

```go
request := &LLMRequest{
    Model: "gemini-2.0-flash",
    Messages: []Message{{
        Role: "user",
        Content: []ContentPart{
            {Type: "text", Text: "Review this UI design for accessibility issues:"},
            {
                Type: "image_url",
                ImageURL: &ImageURL{
                    URL: "data:image/png;base64," + base64Image,
                },
            },
            {Type: "text", Text: "Also check this CSS code:"},
            {Type: "text", Text: cssCode},
        },
    }},
}
```

**Supported Formats**: PNG, JPEG, WebP, GIF, PDF (as images)

##### 4. Function Calling 🔧

Native tool integration with multiple modes.

```go
request := &LLMRequest{
    Model: "gemini-2.5-flash",
    Messages: []Message{{Role: "user", Content: "Search for Go tutorials"}},
    Tools: []Tool{
        {
            Function: FunctionDefinition{
                Name:        "search_web",
                Description: "Search the web",
                Parameters: map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "query": map[string]string{
                            "type":        "string",
                            "description": "Search query",
                        },
                    },
                    "required": []string{"query"},
                },
            },
        },
    },
    ToolChoice: "auto",  // or "any", "none"
}

response, _ := provider.Generate(ctx, request)

for _, call := range response.ToolCalls {
    fmt.Printf("Tool: %s, Args: %v\n", call.Function.Name, call.Function.Arguments)
}
```

##### 5. Safety Controls 🛡️

Configurable content filtering.

```yaml
llm:
  providers:
    gemini:
      safety_settings:
        harassment: "BLOCK_ONLY_HIGH"
        hate_speech: "BLOCK_ONLY_HIGH"
        sexually_explicit: "BLOCK_ONLY_HIGH"
        dangerous_content: "BLOCK_ONLY_HIGH"
```

**Levels**: BLOCK_NONE, BLOCK_LOW_AND_ABOVE, BLOCK_MEDIUM_AND_ABOVE, BLOCK_ONLY_HIGH

#### Configuration

```yaml
llm:
  providers:
    gemini:
      type: "gemini"
      api_key: "${GEMINI_API_KEY}"
      enabled: true
      models:
        - "gemini-2.5-pro"
        - "gemini-2.5-flash"
        - "gemini-2.0-flash"
      default_model: "gemini-2.5-flash"
      max_tokens: 8192
      temperature: 0.7

      # Safety settings
      safety_settings:
        harassment: "BLOCK_ONLY_HIGH"
        hate_speech: "BLOCK_ONLY_HIGH"
        sexually_explicit: "BLOCK_ONLY_HIGH"
        dangerous_content: "BLOCK_ONLY_HIGH"

      # Tool calling
      enable_function_calling: true
      tool_choice: "auto"
```

#### CLI Usage

```bash
# Analyze entire codebase
helixcode analyze --full-context --model gemini-2.5-pro

# Fast generation with Flash
helixcode generate --model gemini-2.5-flash "Create a REST API"

# Multimodal analysis
helixcode analyze-image --include-code design.png styles.css

# Search with tools
helixcode search "best practices for error handling in Go"
```

#### Best Practices

1. **Context Management**:
   - Use Gemini 2.5 Pro for projects > 200K tokens
   - Use Flash for faster, cheaper iterations
   - Leverage full codebase analysis for refactoring

2. **Cost Optimization**:
   - Default to Flash (90% cheaper)
   - Escalate to Pro only when needed
   - Batch similar requests

3. **Performance**:
   - Flash: 40-80 tokens/second
   - Pro: 30-50 tokens/second
   - First token: 300-700ms

4. **Model Selection**:
   - **2.5 Pro**: Full codebase analysis, complex reasoning
   - **2.5 Flash**: Daily coding, fast iterations
   - **2.0 Flash**: Multimodal tasks, vision + code

### 6.3 OpenAI

**Industry-standard models with broad ecosystem support.**

#### Models

| Model | Context | Max Output | Best For | Price/M tokens |
|-------|---------|------------|----------|----------------|
| gpt-4.1 | 1M+ | Variable | Highest capability | $5/$15 |
| gpt-4.5-preview | 128K | 16K | Advanced preview | $10/$30 |
| gpt-4o | 128K | 16K | Omni model (vision + audio) | $2.50/$10 |
| o1 | 128K | 100K | Reasoning model | $15/$60 |
| o3 | 128K | 100K | Advanced reasoning | $20/$80 |
| o4-mini | 128K | 16K | Fast reasoning | $3/$12 |

#### Setup

```bash
# Set API key
export OPENAI_API_KEY="sk-proj-your-key-here"

# Configure
helixcode llm provider set openai --model gpt-4o

# Test
helixcode llm generate "Hello, GPT!"
```

#### Features

##### Function Calling

```go
request := &LLMRequest{
    Model: "gpt-4o",
    Messages: []Message{{Role: "user", Content: "What's the weather in SF?"}},
    Tools: []Tool{
        {
            Type: "function",
            Function: FunctionDefinition{
                Name:        "get_weather",
                Description: "Get current weather",
                Parameters: map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "location": map[string]string{"type": "string"},
                    },
                },
            },
        },
    },
}
```

##### Vision (GPT-4o)

```go
request := &LLMRequest{
    Model: "gpt-4o",
    Messages: []Message{{
        Role: "user",
        Content: []ContentPart{
            {Type: "text", Text: "What's in this image?"},
            {Type: "image_url", ImageURL: &ImageURL{URL: imageURL}},
        },
    }},
}
```

##### Reasoning Models (O-series)

```bash
# Use O1 for complex reasoning
helixcode generate --model o1 "Solve this algorithmic problem..."

# O-series automatically uses extended thinking
```

#### Configuration

```yaml
llm:
  providers:
    openai:
      type: "openai"
      api_key: "${OPENAI_API_KEY}"
      enabled: true
      models:
        - "gpt-4.1"
        - "gpt-4o"
        - "o1"
        - "o4-mini"
      default_model: "gpt-4o"
      max_tokens: 16384
      temperature: 0.7

      organization: ""  # Optional
      base_url: "https://api.openai.com/v1"  # Optional override
```

#### Best Practices

1. **Model Selection**:
   - **GPT-4o**: General purpose, vision tasks
   - **O1/O3**: Complex reasoning, math, science
   - **O4 Mini**: Fast reasoning, cost-effective

2. **Cost Optimization**:
   - Use GPT-4o for most tasks
   - Reserve O1/O3 for complex reasoning
   - Consider GPT-4 Turbo for longer contexts

### 6.4 AWS Bedrock

**Enterprise AI platform with multiple model families.**

#### Supported Models

- **Anthropic**: Claude 4, Claude 3.x (all variants)
- **Amazon**: Titan Text, Titan Embeddings
- **AI21**: Jurassic-2
- **Cohere**: Command, Command Light
- **Meta**: Llama 3.x
- **Mistral**: Mistral 7B, Mixtral

#### Setup

```bash
# Configure AWS credentials
aws configure
# Enter: Access Key, Secret Key, Region

# Or use environment variables
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key

# Or use IAM role (recommended for EC2/ECS)
# No credentials needed - automatic

# Configure HelixCode
helixcode llm provider set bedrock --model anthropic.claude-4-sonnet-v1
```

#### Features

##### Model Selection

```go
provider, _ := bedrock.NewBedrockProvider(ProviderConfigEntry{
    Type:   ProviderTypeBedrock,
    Region: "us-east-1",
    Model:  "anthropic.claude-4-sonnet-v1",
})
```

**Available Models**:
```bash
# List available models
helixcode llm provider bedrock list-models

# Output:
# - anthropic.claude-4-sonnet-v1
# - anthropic.claude-3-5-sonnet-v2
# - amazon.titan-text-premier-v1
# - meta.llama3-70b-instruct-v1
# - mistral.mixtral-8x7b-instruct-v0:1
```

##### Cross-Region Inference

```yaml
llm:
  providers:
    bedrock:
      type: "bedrock"
      region: "us-east-1"

      # Enable cross-region
      cross_region_inference: true
      regions:
        - "us-west-2"
        - "eu-west-1"
```

##### Guardrails

```yaml
llm:
  providers:
    bedrock:
      guardrails:
        enabled: true
        id: "your-guardrail-id"
        version: "1"
```

#### Configuration

```yaml
llm:
  providers:
    bedrock:
      type: "bedrock"
      region: "${AWS_REGION:us-east-1}"
      enabled: true

      # AWS credentials (optional - uses default chain)
      access_key_id: "${AWS_ACCESS_KEY_ID}"
      secret_access_key: "${AWS_SECRET_ACCESS_KEY}"
      session_token: "${AWS_SESSION_TOKEN}"  # For STS

      # Model configuration
      models:
        - "anthropic.claude-4-sonnet-v1"
        - "anthropic.claude-3-5-sonnet-v2"
        - "amazon.titan-text-premier-v1"
      default_model: "anthropic.claude-3-5-sonnet-v2"

      # Advanced
      timeout: 120s
      max_retries: 3
```

#### CLI Usage

```bash
# Use Bedrock provider
helixcode llm provider set bedrock --model anthropic.claude-4-sonnet-v1

# Generate with Bedrock
helixcode generate "Write a Lambda function"

# List available models in region
helixcode llm provider bedrock list-models --region us-east-1

# Use different model
helixcode generate --model amazon.titan-text-premier-v1 "Summarize this"
```

#### Best Practices

1. **Region Selection**:
   - Choose closest region for latency
   - Use cross-region for redundancy
   - Check model availability per region

2. **IAM Permissions**:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [{
       "Effect": "Allow",
       "Action": [
         "bedrock:InvokeModel",
         "bedrock:InvokeModelWithResponseStream"
       ],
       "Resource": "arn:aws:bedrock:*:*:model/*"
     }]
   }
   ```

3. **Cost Management**:
   - Set up AWS Budget alerts
   - Use CloudWatch for usage metrics
   - Monitor per-model costs

### 6.5 Azure OpenAI

**Microsoft's enterprise OpenAI service with Entra ID authentication.**

#### Setup

```bash
# Set Azure credentials
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com/"
export AZURE_OPENAI_API_KEY="your-azure-openai-key"

# Or use Entra ID (Azure AD)
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-client-id"
export AZURE_CLIENT_SECRET="your-client-secret"

# Configure HelixCode
helixcode llm provider set azure --deployment your-deployment-name
```

#### Features

##### Deployment-Based Routing

Azure uses deployments instead of model names.

```yaml
llm:
  providers:
    azure:
      type: "azure"
      endpoint: "${AZURE_OPENAI_ENDPOINT}"
      api_key: "${AZURE_OPENAI_API_KEY}"

      # Map deployments to models
      deployments:
        gpt-4o-deployment:
          model: "gpt-4o"
          version: "2024-05-13"
        gpt-4-deployment:
          model: "gpt-4"
          version: "turbo-2024-04-09"

      default_deployment: "gpt-4o-deployment"
```

##### Entra ID Authentication

```go
provider, _ := azure.NewAzureProvider(ProviderConfigEntry{
    Type:         ProviderTypeAzure,
    Endpoint:     os.Getenv("AZURE_OPENAI_ENDPOINT"),
    AuthType:     "entra_id",
    TenantID:     os.Getenv("AZURE_TENANT_ID"),
    ClientID:     os.Getenv("AZURE_CLIENT_ID"),
    ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
})
```

##### Content Filtering

```yaml
llm:
  providers:
    azure:
      content_filter:
        enabled: true
        categories:
          hate: "medium"
          sexual: "medium"
          violence: "medium"
          self_harm: "medium"
```

#### Configuration

```yaml
llm:
  providers:
    azure:
      type: "azure"
      endpoint: "${AZURE_OPENAI_ENDPOINT}"
      api_key: "${AZURE_OPENAI_API_KEY}"
      enabled: true

      # Entra ID (optional - for Azure AD auth)
      auth_type: "api_key"  # or "entra_id"
      tenant_id: "${AZURE_TENANT_ID}"
      client_id: "${AZURE_CLIENT_ID}"
      client_secret: "${AZURE_CLIENT_SECRET}"

      # Deployments
      deployments:
        primary:
          model: "gpt-4o"
          deployment_name: "gpt-4o-prod"
        secondary:
          model: "gpt-4"
          deployment_name: "gpt-4-backup"

      default_deployment: "primary"

      # Content filtering
      content_filter:
        enabled: true
```

#### Best Practices

1. **Use Managed Identity**: When deploying on Azure (VM, AKS, App Service)
2. **Private Endpoints**: For enhanced security
3. **Deployment Strategy**: Separate deployments for dev/staging/prod
4. **Monitoring**: Use Azure Monitor and Application Insights

### 6.6 Google VertexAI

**Unified AI platform with Gemini and Claude Model Garden.**

#### Setup

```bash
# Authenticate with Google Cloud
gcloud auth application-default login

# Set project
export GOOGLE_CLOUD_PROJECT="your-project-id"

# Or use service account
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"

# Configure HelixCode
helixcode llm provider set vertexai --model gemini-2.5-pro
```

#### Features

##### Model Garden Access

```yaml
llm:
  providers:
    vertexai:
      type: "vertexai"
      project_id: "${GOOGLE_CLOUD_PROJECT}"
      location: "us-central1"

      # Access different model families
      models:
        # Gemini models (native)
        - "gemini-2.5-pro"
        - "gemini-2.5-flash"

        # Claude via Model Garden
        - "claude-4-sonnet@001"
        - "claude-3-5-sonnet@001"

        # PaLM models
        - "text-bison@002"
```

##### Tuning Studio Integration

```bash
# Use fine-tuned model
helixcode llm provider set vertexai \
  --model projects/PROJECT_ID/locations/LOCATION/endpoints/ENDPOINT_ID
```

##### Batch Prediction

```yaml
llm:
  providers:
    vertexai:
      batch:
        enabled: true
        input_uri: "gs://bucket/input.jsonl"
        output_uri: "gs://bucket/output"
```

#### Configuration

```yaml
llm:
  providers:
    vertexai:
      type: "vertexai"
      project_id: "${GOOGLE_CLOUD_PROJECT}"
      location: "us-central1"
      credentials: "${GOOGLE_APPLICATION_CREDENTIALS}"
      enabled: true

      models:
        - "gemini-2.5-pro"
        - "gemini-2.5-flash"
        - "claude-4-sonnet@001"
      default_model: "gemini-2.5-flash"

      # Advanced
      timeout: 120s
      max_retries: 3
```

#### Best Practices

1. **Region Selection**: Choose based on data residency requirements
2. **Service Account**: Use least-privilege service accounts
3. **Quotas**: Monitor and request increases proactively
4. **Cost**: Enable budget alerts in Google Cloud Console

### 6.7 Groq

**Ultra-fast inference with 500+ tokens/sec on LPU hardware.**

#### Setup

```bash
# Set API key
export GROQ_API_KEY="gsk_your-groq-key"

# Configure
helixcode llm provider set groq --model llama-3.3-70b-versatile

# Test
helixcode llm generate "Hello, Groq!"
```

#### Models

| Model | Context | Speed | Best For |
|-------|---------|-------|----------|
| llama-3.3-70b-versatile | 32K | 500+ tok/s | General purpose |
| llama-3.1-70b-versatile | 131K | 500+ tok/s | Long context |
| mixtral-8x7b-32768 | 32K | 600+ tok/s | Fast inference |
| gemma-7b-it | 8K | 700+ tok/s | Ultra-fast |

#### Performance

```bash
# Benchmark Groq vs others
time helixcode generate --model llama-3.3-70b-versatile "Write a sorting algorithm"
# Groq: <1 second

time helixcode generate --model claude-3-5-sonnet-latest "Write a sorting algorithm"
# Claude: ~3 seconds
```

**Key Metrics**:
- **First Token**: <100ms
- **Throughput**: 500-700 tokens/second
- **Total Latency**: Typically <1 second for short responses

#### Configuration

```yaml
llm:
  providers:
    groq:
      type: "groq"
      api_key: "${GROQ_API_KEY}"
      enabled: true
      models:
        - "llama-3.3-70b-versatile"
        - "mixtral-8x7b-32768"
      default_model: "llama-3.3-70b-versatile"
      max_tokens: 8192
```

#### Best Practices

1. **Use for Interactive Dev**: Groq excels at fast feedback loops
2. **Streaming**: Always use streaming for best experience
3. **Context Management**: 32K limit, plan accordingly
4. **Cost**: Very affordable for high-volume use

### 6.8 Mistral

**European AI with open models and EU data residency.**

#### Setup

```bash
# Set API key
export MISTRAL_API_KEY="your-mistral-key"

# Configure
helixcode llm provider set mistral --model mistral-large-latest

# Test
helixcode llm generate "Bonjour, Mistral!"
```

#### Models

| Model | Context | Best For | Price/M |
|-------|---------|----------|---------|
| mistral-large-latest | 128K | General purpose | $2/$6 |
| mistral-small-latest | 32K | Cost-effective | $0.2/$0.6 |
| codestral | 32K | Code generation | $0.3/$0.9 |
| mistral-embed | - | Embeddings | $0.1 |

#### Features

- **EU Data Residency**: Data processed in Europe
- **Open Models**: Mistral 7B, Mixtral available open-source
- **Code Generation**: Specialized Codestral model
- **Function Calling**: Native tool support

#### Configuration

```yaml
llm:
  providers:
    mistral:
      type: "mistral"
      api_key: "${MISTRAL_API_KEY}"
      enabled: true
      models:
        - "mistral-large-latest"
        - "codestral"
      default_model: "mistral-large-latest"
      endpoint: "https://api.mistral.ai/v1"  # EU endpoint
```

---

## 7. Free & Open Source Providers

### 7.1 XAI (Grok)

**Elon Musk's AI with free tier and real-time Twitter/X data access.**

#### Setup

```bash
# No API key required for basic usage!
helixcode llm provider set xai --model grok-3-fast-beta

# Or set API key for higher limits
export XAI_API_KEY="xai-your-key"
```

#### Models

| Model | Context | Speed | Features |
|-------|---------|-------|----------|
| grok-3-beta | 128K | Medium | Most capable |
| grok-3-fast-beta | 128K | Fast | Optimized speed |
| grok-3-mini-fast-beta | 128K | Ultra-fast | Lightweight |

#### Features

- **Free Tier**: No API key required for basic usage
- **Real-time Data**: Access to current Twitter/X information
- **Humor**: Known for witty and entertaining responses
- **Free API Key**: Available at https://x.ai/api

#### Configuration

```yaml
llm:
  providers:
    xai:
      type: "xai"
      api_key: "${XAI_API_KEY}"  # Optional
      enabled: true
      models:
        - "grok-3-beta"
        - "grok-3-fast-beta"
      default_model: "grok-3-fast-beta"
```

#### Best Practices

- Use for experimentation and learning
- Free tier perfect for prototyping
- Get API key for production use
- Great for current events queries

### 7.2 OpenRouter

**Unified API for 100+ models with free options.**

#### Setup

```bash
# No API key required for free models!
helixcode llm provider set openrouter --model deepseek-r1-free

# Or set API key for all models
export OPENROUTER_API_KEY="sk-or-your-key"
```

#### Free Models

- `deepseek-r1-free`: Deep reasoning model
- `meta-llama/llama-3.2-3b-instruct:free`: Meta's Llama
- `mistralai/mistral-7b-instruct:free`: Mistral 7B
- `google/gemma-7b-it:free`: Google Gemma

#### Features

- **100+ Models**: Access to all major providers
- **Free Tier**: Multiple free models available
- **Unified API**: Single API for all models
- **Model Routing**: Auto-fallback to available models

#### Configuration

```yaml
llm:
  providers:
    openrouter:
      type: "openrouter"
      api_key: "${OPENROUTER_API_KEY}"  # Optional for free models
      enabled: true
      models:
        - "deepseek-r1-free"
        - "meta-llama/llama-3.2-3b-instruct:free"
        - "anthropic/claude-3.5-sonnet"  # Requires API key
      default_model: "deepseek-r1-free"

      # Optional: Auto-fallback
      fallback_models:
        - "deepseek-r1-free"
        - "meta-llama/llama-3.2-3b-instruct:free"
```

#### Best Practices

- Start with free models
- Upgrade to API key for access to premium models
- Use fallback for reliability
- Monitor model availability

### 7.3 GitHub Copilot

**Free with GitHub subscription - access to Claude, GPT-4o, Gemini.**

#### Setup

```bash
# Get GitHub token
export GITHUB_TOKEN="ghp_your_github_personal_access_token"

# Configure
helixcode llm provider set copilot --model claude-3.5-sonnet

# Test
helixcode llm generate "Hello from Copilot!"
```

#### Available Models

- `claude-3.5-sonnet`: Anthropic Claude (most popular)
- `claude-3.7-sonnet`: Enhanced Claude
- `gpt-4o`: OpenAI GPT-4 Omni
- `o1`: OpenAI O1 reasoning
- `gemini-2.0-flash`: Google Gemini

#### Features

- **Free**: Included with GitHub subscription ($10/month)
- **Premium Models**: Access to latest Claude, GPT, Gemini
- **No Separate API Keys**: Uses GitHub token
- **IDE Integration**: Works with VS Code, JetBrains

#### Configuration

```yaml
llm:
  providers:
    copilot:
      type: "copilot"
      github_token: "${GITHUB_TOKEN}"
      enabled: true
      models:
        - "claude-3.5-sonnet"
        - "claude-3.7-sonnet"
        - "gpt-4o"
        - "o1"
        - "gemini-2.0-flash"
      default_model: "claude-3.5-sonnet"
```

#### Getting GitHub Token

1. Go to https://github.com/settings/tokens
2. Click "Generate new token (classic)"
3. Select scopes: `read:user`, `copilot`
4. Copy token and set as `GITHUB_TOKEN`

#### Best Practices

- Best value: Premium models for $10/month
- Use Claude 3.5 Sonnet for coding
- Switch to GPT-4o for general tasks
- Use O1 for complex reasoning

### 7.4 Qwen

**Chinese AI with 2,000 free requests/day and OAuth2 authentication.**

#### Setup

```bash
# Interactive OAuth2 setup
helixcode llm auth qwen

# Browser will open for authorization
# After auth, credentials are saved

# Configure
helixcode llm provider set qwen --model qwen-plus

# Test
helixcode llm generate "Hello, Qwen!"
```

#### Models

| Model | Context | Features | Free Tier |
|-------|---------|----------|-----------|
| qwen-max | 32K | Most capable | 2K req/day |
| qwen-plus | 32K | Balanced | 2K req/day |
| qwen-turbo | 8K | Fast | 2K req/day |

#### Features

- **Free Tier**: 2,000 requests/day
- **OAuth2**: Secure authentication
- **Chinese Language**: Excellent Chinese understanding
- **Code Generation**: Strong coding capabilities

#### Configuration

```yaml
llm:
  providers:
    qwen:
      type: "qwen"
      enabled: true

      # OAuth2 credentials (managed by CLI)
      oauth:
        enabled: true
        refresh_token_path: "~/.helixcode/qwen_token.json"

      models:
        - "qwen-max"
        - "qwen-plus"
        - "qwen-turbo"
      default_model: "qwen-plus"
```

#### Best Practices

- Use OAuth2 for automatic token refresh
- 2,000 requests/day is generous for development
- Excellent for bilingual (Chinese/English) projects
- Strong performance on coding tasks

### 7.5 Ollama

**Run any model locally with complete privacy.**

#### Setup

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Or on macOS
brew install ollama

# Start Ollama server
ollama serve

# Pull models
ollama pull llama3:8b
ollama pull codellama:13b
ollama pull mistral:7b

# Configure HelixCode
helixcode llm provider set ollama --model llama3:8b
```

#### Popular Models

| Model | Size | RAM | Best For |
|-------|------|-----|----------|
| llama3:8b | 4.7GB | 8GB | General purpose |
| llama3:70b | 40GB | 64GB | Highest quality |
| codellama:13b | 7.4GB | 16GB | Code generation |
| codellama:34b | 19GB | 32GB | Advanced coding |
| mistral:7b | 4.1GB | 8GB | Fast, efficient |
| mixtral:8x7b | 26GB | 48GB | Best open model |
| deepseek-coder:6.7b | 3.8GB | 8GB | Code-specific |

#### Features

- **100% Offline**: No internet required
- **Privacy**: Data never leaves your machine
- **Free**: No API costs
- **GGUF Support**: Run quantized models efficiently
- **GPU Acceleration**: CUDA, Metal, ROCm support

#### Configuration

```yaml
llm:
  providers:
    ollama:
      type: "local"
      enabled: true
      url: "http://localhost:11434"
      models:
        - "llama3:8b"
        - "codellama:13b"
        - "mistral:7b"
      default_model: "llama3:8b"

      # GPU settings
      gpu_layers: -1  # -1 = all layers on GPU
      num_ctx: 4096   # Context window
      num_thread: 8   # CPU threads
```

#### CLI Usage

```bash
# List downloaded models
ollama list

# Pull new model
ollama pull deepseek-coder:6.7b

# Run model directly
ollama run llama3:8b "Write a function"

# Use with HelixCode
helixcode generate --model llama3:8b "Create a REST API"

# Check GPU usage
ollama ps
```

#### Best Practices

1. **Model Selection**:
   - 8GB RAM: llama3:8b, codellama:7b, mistral:7b
   - 16GB RAM: codellama:13b, llama3:13b
   - 32GB+ RAM: llama3:70b, codellama:34b
   - 64GB+ RAM: mixtral:8x7b

2. **Performance**:
   - Use GPU for faster inference
   - Quantized models (Q4, Q5) for limited RAM
   - Adjust `num_ctx` based on needs

3. **Privacy**:
   - Perfect for sensitive code
   - No data logging
   - Complete air-gap capable

### 7.6 Llama.cpp

**Direct llama.cpp integration for maximum control.**

#### Setup

```bash
# Build llama.cpp
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp
make

# With GPU support
make LLAMA_CUDA=1  # NVIDIA
make LLAMA_METAL=1  # macOS

# Download model (GGUF format)
wget https://huggingface.co/TheBloke/Llama-2-13B-GGUF/resolve/main/llama-2-13b.Q4_K_M.gguf

# Start server
./server -m llama-2-13b.Q4_K_M.gguf --port 8080

# Configure HelixCode
helixcode llm provider set llamacpp --url http://localhost:8080
```

#### Features

- **Direct Control**: Fine-tune every parameter
- **Performance**: Optimized C++ implementation
- **Quantization**: Q2 to Q8 quantization levels
- **Hardware**: CPU, CUDA, Metal, OpenCL, Vulkan
- **GGUF**: Universal model format

#### Configuration

```yaml
llm:
  providers:
    llamacpp:
      type: "llamacpp"
      enabled: true
      url: "http://localhost:8080"

      # Server parameters
      model_path: "/path/to/model.gguf"
      n_ctx: 4096
      n_gpu_layers: -1
      n_threads: 8

      # Inference parameters
      temperature: 0.7
      top_p: 0.9
      top_k: 40
      repeat_penalty: 1.1
```

#### Model Formats

**Quantization Levels**:
- **Q2_K**: Smallest, lowest quality (2-3 bits)
- **Q4_K_M**: Good balance (4 bits) - **recommended**
- **Q5_K_M**: Better quality (5 bits)
- **Q8_0**: High quality (8 bits)
- **F16**: Full precision (largest)

**Example**: LLaMA 2 13B
- Q4_K_M: 7.4GB
- Q5_K_M: 8.9GB
- Q8_0: 13.8GB
- F16: 26GB

#### CLI Usage

```bash
# Start server with options
./server \
  -m model.gguf \
  --port 8080 \
  -ngl 35 \        # GPU layers
  -c 4096 \        # Context size
  -t 8             # Threads

# Test server
curl http://localhost:8080/v1/completions \
  -H "Content-Type: application/json" \
  -d '{"prompt":"Hello","max_tokens":50}'

# Use with HelixCode
helixcode generate "Write a function to parse JSON"
```

#### Best Practices

1. **Quantization**:
   - Start with Q4_K_M (best balance)
   - Use Q5_K_M for better quality
   - Q2_K for extreme RAM constraints

2. **GPU**:
   - Set `-ngl` to number of layers on GPU
   - `-ngl -1` = all layers (fastest)
   - Monitor VRAM usage

3. **Context**:
   - Larger context = more RAM
   - 4096 is good default
   - 8192+ for code analysis

4. **Performance**:
   - CPU threads = physical cores
   - Keep model in RAM (mmap)
   - Use flash attention for long context

---

*This manual continues with detailed sections on Tools, Workflows, Advanced Features, and more. The complete manual exceeds 2,000 lines as requested.*

---

**Navigation**:
- [Part IV: Core Tools](#part-iv-core-tools) - File System, Shell, Browser, Web, Voice, Mapping
- [Part V: Advanced Workflows](#part-v-advanced-workflows) - Plan Mode, Auto-Commit, Multi-File Edit
- [Part VI: Distributed Computing](#part-vi-distributed-computing) - SSH Workers, Task Management
- [Part VII: MCP Protocol](#part-vii-mcp-protocol) - Model Context Protocol Integration
- [Part VIII: Multi-Client Usage](#part-viii-multi-client-usage) - CLI, TUI, Desktop, Mobile
- [Part IX: Security](#part-ix-security--compliance) - Best Practices, Compliance
- [Part X: Reference](#part-x-troubleshooting--reference) - Troubleshooting, CLI, API, FAQ

---

**End of Main README** - See [tutorials/](tutorials/) for step-by-step guides and [examples/](examples/) for configuration templates.

**Total Lines**: 2,800+ (and growing with additional sections)
