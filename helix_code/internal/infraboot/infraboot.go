// Package infraboot gives the helixcode server the ability to boot its
// required infrastructure containers (Postgres + Redis) out-of-the-box on
// startup, then rewrite the in-process *config.Config so the rest of the
// process connects to the just-booted endpoints.
//
// Authority: §11.4.76 (Containers-submodule on-demand-infra invariant) —
// "operators are never required to start podman machine / docker compose up
// manually; the boot is part of the entry point." Before this package existed,
// `helixcode` (cmd/server) did NOT boot any containers: it loaded a config that
// pointed at the compose-network DNS names (postgres/redis), failed to reach
// them on a developer host, and crashed fatally on the Redis dial
// (`lookup redis: no such host`). See cmd/server/main.go wiring.
//
// Decoupling: this package reaches the container runtime ONLY through the
// existing internal/adapters/containers.Adapter (which itself wraps the
// digital.vasic.containers submodule). It introduces no new host dependency
// and reimplements nothing the Containers module already provides (§11.4.74).
//
// Host safety (§11.4.133, §11.4.101): the boot uses DEDICATED host ports
// (default 55432 for Postgres, 56379 for Redis) so it NEVER collides with — or
// clobbers — a host-native Postgres already bound to 5432. Nothing the package
// does is irreversible: it only `compose up`s a dedicated project.
package infraboot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	containersadapter "dev.helix.code/internal/adapters/containers"
	"dev.helix.code/internal/config"
	"dev.helix.code/internal/database"
	"dev.helix.code/internal/redis"
)

// Default dedicated host ports. BOTH deliberately AVOID the conventional ports
// (5432 / 6379) so a host-native Postgres or Redis is never disturbed AND the
// idempotent "already healthy" TCP pre-check can never latch onto a stranger's
// service on the conventional port (host-safety + correctness, §11.4.133).
// Override via HELIX_AUTOBOOT_PG_PORT / HELIX_AUTOBOOT_REDIS_PORT.
const (
	defaultPgPort       = 55432
	defaultRedisPort    = 56379
	defaultPgUser       = "helix"
	defaultPgDBName     = "helixcode_prod"
	defaultHealthBudget = 120 * time.Second
	bootHost            = "127.0.0.1"

	// envAutobootPgPassword is the variable the autoboot compose file reads
	// (`POSTGRES_PASSWORD: ${HELIX_AUTOBOOT_PG_PASSWORD:?...}`). resolvePgPassword
	// exports it before `compose up` so the container and the in-process config
	// are guaranteed to agree on one value.
	envAutobootPgPassword = "HELIX_AUTOBOOT_PG_PASSWORD"
	// envDatabasePassword is the project-wide credential variable, populated
	// from the private, gitignored .env by internal/secrets at server startup.
	envDatabasePassword = "HELIX_DATABASE_PASSWORD"
)

// errNoPgPassword is returned when neither credential variable is set.
// CONST-042 / §12.1: there is deliberately NO built-in fallback value — a
// literal default compiled into a published binary is a published credential.
// Failing fast with an actionable message is the only honest behaviour.
var errNoPgPassword = fmt.Errorf(
	"infraboot: no Postgres password configured — set %s (or %s) in the gitignored .env "+
		"(run ./setup.sh, or `cp .env.example .env` and fill it in), "+
		"or set HELIX_AUTOBOOT_INFRA=false to use externally-provisioned infra",
	envDatabasePassword, envAutobootPgPassword)

// resolvePgPassword reads the auto-boot Postgres credential from the process
// environment — which internal/secrets has already gap-filled from .env before
// EnsureInfra runs (see cmd/server/main.go ordering). The resolved value is
// exported under envAutobootPgPassword so the child `compose` process
// interpolates the SAME secret into the container. The value is never logged.
func resolvePgPassword() (string, error) {
	pw := strings.TrimSpace(os.Getenv(envAutobootPgPassword))
	if pw == "" {
		pw = strings.TrimSpace(os.Getenv(envDatabasePassword))
	}
	if pw == "" {
		return "", errNoPgPassword
	}
	// Export for the compose child process (no-op when already identical).
	if err := os.Setenv(envAutobootPgPassword, pw); err != nil {
		return "", fmt.Errorf("infraboot: could not export %s: %w", envAutobootPgPassword, err)
	}
	return pw, nil
}

// composeFileCandidates are searched (relative to cwd, then the executable's
// directory) when Options.ComposeFile / $HELIX_AUTOBOOT_COMPOSE_FILE are unset.
var composeFileCandidates = []string{
	"docker/autoboot/docker-compose.autoboot.yml",
	"helix_code/docker/autoboot/docker-compose.autoboot.yml",
}

// Booter is the minimal container-runtime surface infraboot needs. It is
// satisfied as-is by *internal/adapters/containers.Adapter and is an interface
// purely so unit tests can inject a fake without a live runtime (CONST-050:
// fakes live in tests only; production wiring uses the real Adapter).
type Booter interface {
	RuntimeAvailable(ctx context.Context) bool
	RuntimeName() string
	ComposeUp(ctx context.Context, composeFile string, services []string) error
	HealthCheckTCP(host, port string) error
}

// Options tunes EnsureInfra. The zero value is valid; nil is also accepted.
type Options struct {
	// Booter overrides the default real containers.Adapter. Tests inject a fake.
	Booter Booter
	// ComposeFile overrides the auto-resolved compose file path.
	ComposeFile string
	// PgPort / RedisPort override the dedicated host ports.
	PgPort    int
	RedisPort int
	// HealthBudget caps the post-up health wait. Zero → defaultHealthBudget.
	HealthBudget time.Duration
	// Enabled overrides the $HELIX_AUTOBOOT_INFRA env gate when non-nil.
	Enabled *bool
	// Readiness, when set, is polled (after cfg is rewritten) until it returns
	// nil — a PROTOCOL-level readiness probe that TCP-up cannot guarantee
	// (Postgres opens its port mid-initdb, before it accepts auth). When nil
	// AND Booter is nil (production path) it defaults to defaultReadiness
	// (real pgx ping + real go-redis PING). When a fake Booter is injected
	// (unit tests) it stays nil so tests need no live DB.
	Readiness func(ctx context.Context, cfg *config.Config) error
	// pollInterval is the health re-check cadence (test seam; 0 → 1s).
	pollInterval time.Duration
}

// Result reports what EnsureInfra did. It is always non-nil on success.
type Result struct {
	// Booted is true when infra is ensured up AND cfg has been rewritten to
	// point at it (whether this call started the containers or found them
	// already healthy).
	Booted bool
	// AlreadyRunning is true when both endpoints were already healthy, so no
	// `compose up` was issued.
	AlreadyRunning bool
	// Skipped is true when autoboot was disabled; cfg is left untouched.
	Skipped     bool
	Reason      string
	RuntimeName string
	PgPort      int
	RedisPort   int
	ComposeFile string
}

// EnsureInfra boots the required infrastructure when enabled and rewrites cfg's
// Database + Redis sections to target the booted endpoints. It is idempotent:
// if both endpoints are already healthy it skips `compose up` but still rewrites
// cfg. When disabled (HELIX_AUTOBOOT_INFRA=false) it returns immediately and
// leaves cfg untouched so externally-provisioned infra is honoured verbatim.
func EnsureInfra(ctx context.Context, cfg *config.Config, opts *Options) (*Result, error) {
	if opts == nil {
		opts = &Options{}
	}
	if !enabled(opts.Enabled) {
		return &Result{Skipped: true, Reason: "disabled via HELIX_AUTOBOOT_INFRA"}, nil
	}
	if cfg == nil {
		return nil, fmt.Errorf("infraboot: nil config")
	}

	pgPort := resolvePort(opts.PgPort, "HELIX_AUTOBOOT_PG_PORT", defaultPgPort)
	redisPort := resolvePort(opts.RedisPort, "HELIX_AUTOBOOT_REDIS_PORT", defaultRedisPort)

	// Resolve the credential BEFORE either branch: the already-healthy path
	// still needs it for rewriteConfig, and failing here gives the operator one
	// actionable message instead of a downstream auth error (CONST-042).
	pgPassword, err := resolvePgPassword()
	if err != nil {
		return nil, err
	}

	b := opts.Booter
	if b == nil {
		b = containersadapter.NewAdapter()
	}

	res := &Result{PgPort: pgPort, RedisPort: redisPort}

	healthy := func() bool {
		return b.HealthCheckTCP(bootHost, strconv.Itoa(pgPort)) == nil &&
			b.HealthCheckTCP(bootHost, strconv.Itoa(redisPort)) == nil
	}

	if healthy() {
		res.AlreadyRunning = true
		res.Reason = "endpoints already healthy"
	} else {
		// Host-safety (§11.4.133): never issue an irreversible `compose up` once
		// the caller has already cancelled — that would boot containers the
		// caller no longer wants and leave them orphaned (the call returns an
		// error, so the caller never gets a Result to tear them down). Check
		// BEFORE the boot, not only inside the post-up health wait.
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("infraboot: boot aborted before compose up: %w", err)
		}
		if !b.RuntimeAvailable(ctx) {
			return nil, fmt.Errorf("infraboot: no container runtime available (docker/podman required to auto-boot infra; set HELIX_AUTOBOOT_INFRA=false to use external infra)")
		}
		res.RuntimeName = b.RuntimeName()

		composeFile, err := resolveComposeFile(opts.ComposeFile)
		if err != nil {
			return nil, err
		}
		res.ComposeFile = composeFile

		if err := b.ComposeUp(ctx, composeFile, []string{"postgres", "redis"}); err != nil {
			return nil, fmt.Errorf("infraboot: compose up failed: %w", err)
		}

		if err := waitHealthy(ctx, b, pgPort, redisPort, budget(opts.HealthBudget), poll(opts.pollInterval)); err != nil {
			return nil, err
		}
		res.Reason = "booted via " + res.RuntimeName
	}

	if res.RuntimeName == "" {
		res.RuntimeName = b.RuntimeName()
	}
	rewriteConfig(cfg, pgPort, redisPort, pgPassword)

	// Protocol-level readiness: TCP-up does NOT mean Postgres accepts auth
	// (it opens the port mid-initdb). Poll a real readiness probe until the
	// booted services actually answer, so a PASS here means the next
	// database.New / redis.NewClient in the caller genuinely connects
	// (§11.4.5 real evidence, not a port-open false-green).
	readiness := opts.Readiness
	if readiness == nil && opts.Booter == nil {
		readiness = defaultReadiness
	}
	if readiness != nil {
		if err := waitReady(ctx, cfg, readiness, budget(opts.HealthBudget), poll(opts.pollInterval)); err != nil {
			return nil, err
		}
	}

	res.Booted = true
	return res, nil
}

// defaultReadiness is the production readiness probe: it proves the booted
// Postgres accepts a real pgx connection+ping AND the booted Redis answers a
// real PING, using the just-rewritten cfg. Both clients are closed immediately
// (this is a probe, not the long-lived connection the caller will open).
func defaultReadiness(ctx context.Context, cfg *config.Config) error {
	db, err := database.New(cfg.Database)
	if err != nil {
		return fmt.Errorf("postgres not ready: %w", err)
	}
	pingErr := db.Ping(ctx)
	db.Close()
	if pingErr != nil {
		return fmt.Errorf("postgres ping not ready: %w", pingErr)
	}
	rds, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		return fmt.Errorf("redis not ready: %w", err)
	}
	rds.Close()
	return nil
}

func waitReady(ctx context.Context, cfg *config.Config, probe func(context.Context, *config.Config) error, budget, interval time.Duration) error {
	deadline := time.Now().Add(budget)
	var last error
	for {
		if last = probe(ctx, cfg); last == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("infraboot: infra not ready within %s: %w", budget, last)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("infraboot: readiness wait cancelled: %w", ctx.Err())
		case <-time.After(interval):
		}
	}
}

// rewriteConfig points cfg at the booted Postgres + Redis endpoints. This is
// the load-bearing step: without it the process would still try the original
// compose-DNS hostnames and fail (the original defect).
func rewriteConfig(cfg *config.Config, pgPort, redisPort int, pgPassword string) {
	cfg.Database.Host = bootHost
	cfg.Database.Port = pgPort
	cfg.Database.User = defaultPgUser
	cfg.Database.Password = pgPassword
	cfg.Database.DBName = defaultPgDBName
	cfg.Database.SSLMode = "disable"

	cfg.Redis.Enabled = true
	cfg.Redis.Host = bootHost
	cfg.Redis.Port = redisPort
	cfg.Redis.Password = ""
}

func waitHealthy(ctx context.Context, b Booter, pgPort, redisPort int, budget, interval time.Duration) error {
	deadline := time.Now().Add(budget)
	var lastPg, lastRedis error
	for {
		lastPg = b.HealthCheckTCP(bootHost, strconv.Itoa(pgPort))
		lastRedis = b.HealthCheckTCP(bootHost, strconv.Itoa(redisPort))
		if lastPg == nil && lastRedis == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("infraboot: infra did not become healthy within %s (postgres:%d=%v redis:%d=%v)",
				budget, pgPort, lastPg, redisPort, lastRedis)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("infraboot: health wait cancelled: %w", ctx.Err())
		case <-time.After(interval):
		}
	}
}

func resolveComposeFile(override string) (string, error) {
	candidates := []string{}
	if override != "" {
		candidates = append(candidates, override)
	}
	if env := os.Getenv("HELIX_AUTOBOOT_COMPOSE_FILE"); env != "" {
		candidates = append(candidates, env)
	}
	bases := []string{}
	if cwd, err := os.Getwd(); err == nil {
		bases = append(bases, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		bases = append(bases, filepath.Dir(exe))
	}
	for _, base := range bases {
		for _, rel := range composeFileCandidates {
			candidates = append(candidates, filepath.Join(base, rel))
		}
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		abs := c
		if !filepath.IsAbs(abs) {
			if a, err := filepath.Abs(c); err == nil {
				abs = a
			}
		}
		if st, err := os.Stat(abs); err == nil && !st.IsDir() {
			return abs, nil
		}
	}
	return "", fmt.Errorf("infraboot: autoboot compose file not found (looked for %v; set HELIX_AUTOBOOT_COMPOSE_FILE)", composeFileCandidates)
}

func enabled(override *bool) bool {
	if override != nil {
		return *override
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("HELIX_AUTOBOOT_INFRA")))
	switch v {
	case "0", "false", "no", "off":
		return false
	default: // unset or any truthy value → enabled (out-of-the-box default)
		return true
	}
}

func resolvePort(override int, env string, def int) int {
	if override > 0 {
		return override
	}
	if v := strings.TrimSpace(os.Getenv(env)); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 && p < 65536 {
			return p
		}
	}
	return def
}

func budget(d time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return defaultHealthBudget
}

func poll(d time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return time.Second
}
