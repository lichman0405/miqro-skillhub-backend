// Command skillhub-worker runs background job processing.
//
// Phase 12 adds agent CI/CD worker execution.
// The worker can execute pipeline runs using the local deterministic runner.
//
// LLM runner configuration comes from environment variables:
//
//	AGENTCI_LLM_BASE_URL  (optional)
//	AGENTCI_LLM_API_KEY   (never logged)
//	AGENTCI_LLM_MODEL     (optional)
//	AGENTCI_LLM_PROVIDER  (optional)
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"miqro-skillhub/server/internal/adapters/agentrunner"
	"miqro-skillhub/server/internal/adapters/postgres"
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

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"
	}

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

	// Register the local deterministic runner.
	localRunner := agentrunner.NewLocalRunner()
	agentciSvc.RegisterRunner(localRunner)

	// Create the executor.
	exec := agentrunner.NewExecutor(agentciSvc)
	exec.RegisterRunner(localRunner)

	fmt.Fprintf(os.Stderr, "skillhub-worker: agent CI worker started (runner=%s, llm=%s)\n",
		localRunner.Name(), agentrunner.RedactedLLMConfig())

	// Minimal worker loop: poll for pending pipeline runs.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	fmt.Fprintln(os.Stderr, "skillhub-worker: polling for pipeline runs every 30s...")

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "skillhub-worker: stopped.")
			return
		case <-ticker.C:
			// In a full implementation, the worker would:
			//   1. Query ci_pipeline_runs WHERE status='PENDING'
			//   2. Execute each pending run via executor.ExecutePipelineRun()
			// For now, the executor is exposed for testability and
			// can be invoked programmatically.
			_ = exec
		}
	}
}
