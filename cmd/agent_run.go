package cmd

import (
	"fmt"
	"time"

	"github.com/amald/cx/internal/memory"
	"github.com/amald/cx/internal/project"
	"github.com/amald/cx/internal/ui"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var agentRunCmd = &cobra.Command{
	Use:   "agent-run",
	Short: "Track agent run invocations",
}

var (
	arLogType      string
	arLogSession   string
	arLogStatus    string
	arLogSummary   string
	arLogArtifacts string
	arLogDuration  int
	arLogPrompt    string
	arListSession  string
)

var agentRunLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Log an agent run",
	RunE:  runAgentRunLog,
}

var agentRunListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agent runs",
	RunE:  runAgentRunList,
}

func init() {
	agentRunLogCmd.Flags().StringVar(&arLogType, "type", "", "Agent type (required)")
	agentRunLogCmd.Flags().StringVar(&arLogSession, "session", "", "Session ID")
	agentRunLogCmd.Flags().StringVar(&arLogStatus, "status", "", "Result status: success, blocked, needs-input")
	agentRunLogCmd.Flags().StringVar(&arLogSummary, "summary", "", "Result summary")
	agentRunLogCmd.Flags().StringVar(&arLogArtifacts, "artifacts", "", "Comma-separated artifact paths")
	agentRunLogCmd.Flags().IntVar(&arLogDuration, "duration-ms", 0, "Duration in milliseconds")
	agentRunLogCmd.Flags().StringVar(&arLogPrompt, "prompt-summary", "", "First 200 chars of prompt")

	agentRunListCmd.Flags().StringVar(&arListSession, "session", "", "Filter by session ID")

	agentRunCmd.AddCommand(agentRunLogCmd)
	agentRunCmd.AddCommand(agentRunListCmd)
}

func runAgentRunLog(cmd *cobra.Command, args []string) error {
	if arLogType == "" {
		ui.PrintError("--type is required")
		return errExitCode1
	}
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository")
		return errExitCode1
	}
	db, err := memory.OpenProjectDB(rootDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("opening memory DB: %v", err))
		return errExitCode1
	}
	defer db.Close()

	run := memory.AgentRun{
		ID:            uuid.New().String()[:8],
		SessionID:     arLogSession,
		AgentType:     arLogType,
		PromptSummary: arLogPrompt,
		ResultStatus:  arLogStatus,
		ResultSummary: arLogSummary,
		Artifacts:     arLogArtifacts,
		DurationMs:    arLogDuration,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	if err := memory.SaveAgentRun(db, run); err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}
	ui.PrintSuccess(fmt.Sprintf("logged agent run: %s (%s)", arLogType, run.ID))
	return nil
}

func runAgentRunList(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository")
		return errExitCode1
	}
	db, err := memory.OpenProjectDB(rootDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("opening memory DB: %v", err))
		return errExitCode1
	}
	defer db.Close()

	runs, err := memory.ListAgentRuns(db, arListSession)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}
	for _, r := range runs {
		status := r.ResultStatus
		if status == "" {
			status = "pending"
		}
		fmt.Printf("%s  %s  [%s]  %s\n", r.CreatedAt, r.AgentType, status, r.ID)
	}
	if len(runs) == 0 {
		ui.PrintMuted("no agent runs found")
	}
	return nil
}
