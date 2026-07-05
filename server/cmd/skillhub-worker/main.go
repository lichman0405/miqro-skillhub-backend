// Command skillhub-worker runs background job processing.
//
// Phase 12 adds agent CI/CD worker execution.
// The worker polls for PENDING pipeline runs and executes them using
// registered runner adapters. The local deterministic runner handles
// required checks; LLM checks are optional and require AGENTCI_LLM_* env vars.
//
// LLM runner configuration comes from environment variables:
//
//	AGENTCI_LLM_BASE_URL  (optional)
//	AGENTCI_LLM_API_KEY   (never logged)
//	AGENTCI_LLM_MODEL     (optional)
//	AGENTCI_LLM_PROVIDER  (optional)
//
// Concurrency safety: ClaimPending atomically updates status from PENDING to
// RUNNING via a conditional UPDATE, so multiple workers never execute the
// same pipeline run.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"miqro-skillhub/server/internal/adapters/agentrunner"
	"miqro-skillhub/server/internal/adapters/postgres"
	"miqro-skillhub/server/internal/adapters/storagefactory"
	"miqro-skillhub/server/internal/config"
	"miqro-skillhub/server/sdk/skillhub/agentci"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGTERM/SIGINT.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "skillhub-worker: shutting down...")
		cancel()
	}()

	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "skillhub-worker: config: %v\n", err)
		os.Exit(1)
	}

	connStr := cfg.DatabaseURL

	// Connect to database.
	db, err := postgres.NewDB(ctx, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skillhub-worker: database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Build agent CI service with postgres repos.
	agentciSvc := agentci.NewService(
		postgres.NewPipelineDefinitionRepo(db),
		postgres.NewPipelineRunRepo(db),
		postgres.NewCheckRunRepo(db),
		postgres.NewCheckStepRepo(db),
		postgres.NewCheckArtifactRepo(db),
		postgres.NewGatePolicyRepo(db),
		postgres.NewAgentWorkerRepo(db),
		nil, // log store not yet wired
	)

	// Object storage via unified storage factory.
	objStore, err := storagefactory.New(ctx, *cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skillhub-worker: WARNING: cannot create object storage: %v — checks will lack file content\n", err)
		objStore = nil
	}

	// Register the local deterministic runner.
	localRunner := agentrunner.NewLocalRunner()

	// Wire version file reader using version/file repos + object storage for content.
	localRunner.SetVersionFileReader(func(ctx context.Context, versionID, skillID int64) ([]agentci.PackageFileEntry, error) {
		versionRepo := postgres.NewSkillVersionRepo(db)
		fileRepo := postgres.NewSkillFileRepo(db)
		version, verr := versionRepo.FindByID(ctx, versionID)
		if verr != nil {
			return nil, fmt.Errorf("find version: %w", verr)
		}
		if version == nil {
			return nil, fmt.Errorf("version %d not found", versionID)
		}
		files, ferr := fileRepo.FindByVersionID(ctx, versionID)
		if ferr != nil {
			return nil, fmt.Errorf("find files: %w", ferr)
		}
		entries := make([]agentci.PackageFileEntry, 0, len(files))
		for _, f := range files {
			var content []byte
			if f.StorageKey != "" && objStore != nil {
				rc, getErr := objStore.GetObject(ctx, f.StorageKey)
				if getErr == nil {
					content, _ = io.ReadAll(rc)
					rc.Close()
				}
			}
			entries = append(entries, agentci.PackageFileEntry{
				Path:        f.FilePath,
				Content:     content,
				Size:        f.FileSize,
				ContentType: f.ContentType,
			})
		}
		return entries, nil
	})

	agentciSvc.RegisterRunner(localRunner)

	// Create the executor.
	exec := agentrunner.NewExecutor(agentciSvc)
	exec.RegisterRunner(localRunner)

	fmt.Fprintf(os.Stderr, "skillhub-worker: agent CI worker started (runner=%s, llm=%s)\n",
		localRunner.Name(), agentrunner.RedactedLLMConfig())

	// Polling interval.
	pollInterval := 30 * time.Second
	if v := os.Getenv("AGENTCI_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= time.Second {
			pollInterval = d
		}
	}

	fmt.Fprintf(os.Stderr, "skillhub-worker: polling for pipeline runs every %s...\n", pollInterval)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Process immediately on start, then on each tick.
	poll := func() {
		pendingRuns, err := agentciSvc.FindPendingRuns(ctx, 10)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skillhub-worker: error listing pending runs: %v\n", err)
			return
		}
		if len(pendingRuns) == 0 {
			return
		}

		fmt.Fprintf(os.Stderr, "skillhub-worker: found %d pending runs\n", len(pendingRuns))

		for _, run := range pendingRuns {
			// Claim the run (atomically swap PENDING → RUNNING).
			claimed, err := agentciSvc.ClaimPendingRun(ctx, run.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "skillhub-worker: error claiming run %d: %v\n", run.ID, err)
				continue
			}
			if claimed == nil {
				// Already claimed by another worker.
				continue
			}

			// Execute the pipeline run.
			fmt.Fprintf(os.Stderr, "skillhub-worker: executing pipeline run %d (skill=%d, checks=%d)\n",
				claimed.ID, claimed.SkillID, claimed.CheckCount)

			if err := exec.ExecutePipelineRun(ctx, claimed.ID); err != nil {
				fmt.Fprintf(os.Stderr, "skillhub-worker: error executing run %d: %v\n", claimed.ID, err)
			}
		}
	}

	// Initial poll.
	poll()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "skillhub-worker: stopped.")
			return
		case <-ticker.C:
			poll()
		}
	}
}
