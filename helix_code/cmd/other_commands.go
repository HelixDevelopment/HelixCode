package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/database"
	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/notification"
	"dev.helix.code/internal/redis"
	"dev.helix.code/internal/server"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: trc("cmd_server_short", nil),
	Long:  trc("cmd_server_long", nil),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		// Speed programme P2-T07: config.Get() caches the process config.
		cfg, err := config.Get()
		if err != nil {
			fmt.Fprintln(os.Stderr, tr(ctx, "cmd_err_config", map[string]any{"Error": err.Error()}))
			return
		}

		var db *database.Database
		if cfg.Database.Host != "" {
			db, err = database.New(cfg.Database)
			if err != nil {
				fmt.Fprintln(os.Stderr, tr(ctx, "cmd_err_database_unavailable", map[string]any{"Error": err.Error()}))
			} else {
				defer db.Close()
			}
		}

		var rds *redis.Client
		if cfg.Redis.Enabled && cfg.Redis.Host != "" {
			rds, err = redis.NewClient(&cfg.Redis)
			if err != nil {
				fmt.Fprintln(os.Stderr, tr(ctx, "cmd_err_redis_unavailable", map[string]any{"Error": err.Error()}))
			} else {
				defer rds.Close()
			}
		}

		srv := server.New(cfg, db, rds)

		errChan := make(chan error, 1)
		go func() {
			if err := srv.Start(); err != nil {
				errChan <- err
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case err := <-errChan:
			fmt.Fprintln(os.Stderr, tr(ctx, "cmd_err_server", map[string]any{"Error": err.Error()}))
		case sig := <-quit:
			fmt.Println("\n" + tr(ctx, "cmd_server_received_signal", map[string]any{"Signal": sig.String()}))
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintln(os.Stderr, tr(ctx, "cmd_err_shutdown", map[string]any{"Error": err.Error()}))
		}
		fmt.Println(tr(ctx, "cmd_server_stopped", nil))
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: trc("cmd_version_short", nil),
	Long:  trc("cmd_version_long", nil),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		fmt.Println(tr(ctx, "cmd_version_platform_name", nil))
		fmt.Println(tr(ctx, "cmd_version_version", map[string]any{"Version": "1.0.0"}))
		fmt.Println(tr(ctx, "cmd_version_build", map[string]any{"Build": "2025.01.20"}))
		fmt.Println(tr(ctx, "cmd_version_providers", map[string]any{"Total": 29, "Cloud": 18, "Local": 11}))
		fmt.Println(tr(ctx, "cmd_version_token_context", map[string]any{"Context": "2M"}))
		fmt.Println(tr(ctx, "cmd_version_license", map[string]any{"License": "MIT"}))
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate [prompt]",
	Short: trc("cmd_generate_short", nil),
	Long:  trc("cmd_generate_long", nil),
	Run: func(cmd *cobra.Command, args []string) {
		ctx0 := context.Background()
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, tr(ctx0, "cmd_generate_need_prompt", nil))
			return
		}
		prompt := args[0]

		// Speed programme P2-T07: config.Get() caches the process config.
		cfg, err := config.Get()
		if err != nil {
			fmt.Fprintln(os.Stderr, tr(ctx0, "cmd_err_config", map[string]any{"Error": err.Error()}))
			return
		}

		mgr := llm.NewModelManager()
		defaultProvider := cfg.LLM.DefaultProvider
		if defaultProvider == "" {
			fmt.Fprintln(os.Stderr, tr(ctx0, "cmd_generate_no_default_provider", nil))
			fmt.Fprintln(os.Stderr, tr(ctx0, "cmd_generate_set_default_provider", nil))
			return
		}

		modelInfo, err := mgr.SelectOptimalModel(llm.ModelSelectionCriteria{
			TaskType:          "text-generation",
			QualityPreference: "balanced",
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, tr(ctx0, "cmd_generate_no_models", map[string]any{"Error": err.Error()}))
			return
		}

		entryKey := llm.ProviderType(defaultProvider)
		prov, err := mgr.GetProviderForModel(modelInfo.Name, entryKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, tr(ctx0, "cmd_generate_provider_unavailable", map[string]any{"Error": err.Error()}))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			Model: modelInfo.Name,
			Messages: []llm.Message{
				{Role: "user", Content: prompt},
			},
			MaxTokens:   4096,
			Temperature: 0.7,
		}
		response, err := prov.Generate(ctx, request)
		if err != nil {
			fmt.Fprintln(os.Stderr, tr(ctx0, "cmd_generate_failed", map[string]any{"Error": err.Error()}))
			return
		}
		fmt.Println(response.Content)
	},
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: trc("cmd_test_short", nil),
	Long:  trc("cmd_test_long", nil),
	Run: func(cmd *cobra.Command, args []string) {
		testArgs := []string{"test", "-v"}
		if len(args) > 0 {
			testArgs = append(testArgs, args...)
		} else {
			testArgs = append(testArgs, "./...")
		}
		c := exec.Command("go", testArgs...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			fmt.Fprintln(os.Stderr, tr(context.Background(), "cmd_test_failed", map[string]any{"Error": err.Error()}))
			os.Exit(1)
		}
	},
}

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: trc("cmd_worker_short", nil),
	Long:  trc("cmd_worker_long", nil),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		// Speed programme P2-T07: config.Get() caches the process config.
		cfg, err := config.Get()
		if err != nil {
			fmt.Fprintln(os.Stderr, tr(ctx, "cmd_err_config", map[string]any{"Error": err.Error()}))
			return
		}

		if cfg.Database.Host == "" {
			fmt.Println(tr(ctx, "cmd_worker_needs_database", nil))
			fmt.Println(tr(ctx, "cmd_worker_set_database", nil))
			return
		}

		fmt.Println(tr(ctx, "cmd_worker_config_summary", map[string]any{
			"HealthTTL":     cfg.Workers.HealthTTL,
			"MaxConcurrent": cfg.Workers.MaxConcurrentTasks,
		}))
		fmt.Println(tr(ctx, "cmd_worker_use_subcommands", nil))
	},
}

var notifyCmd = &cobra.Command{
	Use:   "notify [message]",
	Short: trc("cmd_notify_short", nil),
	Long:  trc("cmd_notify_long", nil),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, tr(context.Background(), "cmd_notify_need_message", nil))
			return
		}
		message := args[0]

		// Channels and rules come from the `notifications:` config block AND
		// the HELIX_* environment variables, with the environment winning —
		// see notification.NewEngineFromConfig for the precedence contract.
		// Before this, the block was discarded by viper and only the
		// environment produced a channel.
		//
		// A config that fails to load is not fatal here: notify's job is to
		// deliver a message, and the environment-only path it used before
		// still works without any config at all, so fall back to it rather
		// than refusing to send.
		cfg, cfgErr := config.Get()
		if cfgErr != nil {
			fmt.Fprintln(os.Stderr, tr(context.Background(), "cmd_notify_config_unavailable",
				map[string]any{"Error": cfgErr.Error()}))
			cfg = nil
		}
		engine := notification.NewEngineFromConfig(cfg)

		notif := &notification.Notification{
			Title:    tr(context.Background(), "cmd_notify_title", nil),
			Message:  message,
			Type:     notification.NotificationTypeInfo,
			Priority: notification.NotificationPriorityMedium,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := engine.SendNotification(ctx, notif); err != nil {
			fmt.Fprintln(os.Stderr, tr(ctx, "cmd_notify_failed", map[string]any{"Error": err.Error()}))
			return
		}
		fmt.Println(tr(ctx, "cmd_notify_dispatched", nil))
	},
}
