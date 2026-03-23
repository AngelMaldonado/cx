package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/AngelMaldonado/cx/internal/memory"
	"github.com/AngelMaldonado/cx/internal/project"
	"github.com/AngelMaldonado/cx/internal/ui"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage project memory (observations, decisions, sessions)",
}

// Flags
var (
	memSaveType       string
	memSaveTitle      string
	memSaveContent    string
	memSaveChange     string
	memSaveFiles      string
	memSaveSpecs      string
	memSaveTags       string
	memSaveDeprecates string
	memSaveSource     string
	memSaveVisibility string
	memSaveAuthor     string
)

// Deprecate flags
var memDeprecateForce bool

var memorySaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save an observation or agent interaction",
	RunE:  runMemorySave,
}

var memoryDecideCmd = &cobra.Command{
	Use:   "decide",
	Short: "Record a technical decision",
	RunE:  runMemoryDecide,
}

var memorySessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Save a session summary",
	RunE:  runMemorySession,
}

var memorySearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search memories via FTS5",
	Args:  cobra.ExactArgs(1),
	RunE:  runMemorySearch,
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List memories with filters",
	RunE:  runMemoryList,
}

var memoryPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Export project memories to docs/memory/",
	RunE:  runMemoryPush,
}

var memoryPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Import memories from docs/memory/ into DB",
	RunE:  runMemoryPull,
}

var memoryLinkCmd = &cobra.Command{
	Use:   "link <id1> <id2>",
	Short: "Link two memory entities",
	Args:  cobra.ExactArgs(2),
	RunE:  runMemoryLink,
}

var memoryNoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Save or update a personal note",
	RunE:  runMemoryNote,
}

var memoryForgetCmd = &cobra.Command{
	Use:   "forget <id>",
	Short: "Delete a personal note",
	Args:  cobra.ExactArgs(1),
	RunE:  runMemoryForget,
}

var memoryDeprecateCmd = &cobra.Command{
	Use:   "deprecate <id>",
	Short: "Mark a memory as deprecated",
	Args:  cobra.ExactArgs(1),
	RunE:  runMemoryDeprecate,
}

// Note flags
var (
	memNoteType     string
	memNoteTitle    string
	memNoteContent  string
	memNoteTopicKey string
	memNoteProjects string
	memNoteTags     string
)

// Search/list flags
var (
	memSearchType        string
	memSearchAuthor      string
	memSearchChange      string
	memSearchAllProjects bool
	memSearchDeprecated  bool
	memListType          string
	memListChange        string
	memListRecent        string
	memListLimit         int
	memPushAll           bool
	memLinkRelation      string
)

// Decide flags
var (
	memDecideTitle        string
	memDecideContext      string
	memDecideOutcome      string
	memDecideAlternatives string
	memDecideRationale    string
	memDecideTradeoffs    string
	memDecideChange       string
	memDecideSpecs        string
	memDecideTags         string
	memDecideDeprecates   string
	memDecideVisibility   string
)

// Session flags
var (
	memSessionGoal         string
	memSessionAccomplished string
	memSessionNext         string
	memSessionDiscoveries  string
	memSessionBlockers     string
	memSessionFiles        string
	memSessionChange       string
)

func init() {
	// save flags
	memorySaveCmd.Flags().StringVar(&memSaveType, "type", "", "Entity type: observation or agent_interaction (required)")
	memorySaveCmd.Flags().StringVar(&memSaveTitle, "title", "", "Title (required)")
	memorySaveCmd.Flags().StringVar(&memSaveContent, "content", "", "Content body (required)")
	memorySaveCmd.Flags().StringVar(&memSaveChange, "change", "", "Active change name")
	memorySaveCmd.Flags().StringVar(&memSaveFiles, "files", "", "Comma-separated file paths")
	memorySaveCmd.Flags().StringVar(&memSaveSpecs, "specs", "", "Comma-separated spec areas")
	memorySaveCmd.Flags().StringVar(&memSaveTags, "tags", "", "Comma-separated tags")
	memorySaveCmd.Flags().StringVar(&memSaveDeprecates, "deprecates", "", "ID of entity this replaces")
	memorySaveCmd.Flags().StringVar(&memSaveSource, "source", "", "Source agent name")
	memorySaveCmd.Flags().StringVar(&memSaveVisibility, "visibility", "project", "Visibility: personal or project")
	memorySaveCmd.Flags().StringVar(&memSaveAuthor, "author", "agent", "Author name")

	// decide flags
	memoryDecideCmd.Flags().StringVar(&memDecideTitle, "title", "", "Decision title (required)")
	memoryDecideCmd.Flags().StringVar(&memDecideContext, "context", "", "Decision context (required)")
	memoryDecideCmd.Flags().StringVar(&memDecideOutcome, "outcome", "", "Chosen outcome (required)")
	memoryDecideCmd.Flags().StringVar(&memDecideAlternatives, "alternatives", "", "Alternatives considered (required)")
	memoryDecideCmd.Flags().StringVar(&memDecideRationale, "rationale", "", "Rationale (required)")
	memoryDecideCmd.Flags().StringVar(&memDecideTradeoffs, "tradeoffs", "", "Tradeoffs")
	memoryDecideCmd.Flags().StringVar(&memDecideChange, "change", "", "Active change name")
	memoryDecideCmd.Flags().StringVar(&memDecideSpecs, "specs", "", "Comma-separated spec areas")
	memoryDecideCmd.Flags().StringVar(&memDecideTags, "tags", "", "Comma-separated tags")
	memoryDecideCmd.Flags().StringVar(&memDecideDeprecates, "deprecates", "", "ID of entity this replaces")
	memoryDecideCmd.Flags().StringVar(&memDecideVisibility, "visibility", "project", "Visibility: personal or project")
	memoryDecideCmd.Flags().StringVar(&memSaveAuthor, "author", "agent", "Author name")

	// session flags
	memorySessionCmd.Flags().StringVar(&memSessionGoal, "goal", "", "Session goal (required)")
	memorySessionCmd.Flags().StringVar(&memSessionAccomplished, "accomplished", "", "What was accomplished (required)")
	memorySessionCmd.Flags().StringVar(&memSessionNext, "next", "", "Next steps (required)")
	memorySessionCmd.Flags().StringVar(&memSessionDiscoveries, "discoveries", "", "Discoveries made")
	memorySessionCmd.Flags().StringVar(&memSessionBlockers, "blockers", "", "Blockers encountered")
	memorySessionCmd.Flags().StringVar(&memSessionFiles, "files", "", "Comma-separated files modified")
	memorySessionCmd.Flags().StringVar(&memSessionChange, "change", "", "Active change name")
	memorySessionCmd.Flags().StringVar(&memSaveAuthor, "author", "agent", "Author name")

	// search flags
	memorySearchCmd.Flags().StringVar(&memSearchType, "type", "", "Filter by entity type")
	memorySearchCmd.Flags().StringVar(&memSearchAuthor, "author", "", "Filter by author")
	memorySearchCmd.Flags().StringVar(&memSearchChange, "change", "", "Filter by change")
	memorySearchCmd.Flags().BoolVar(&memSearchAllProjects, "all-projects", false, "Search across all projects")
	memorySearchCmd.Flags().BoolVar(&memSearchDeprecated, "include-deprecated", false, "Include deprecated entries")

	// list flags
	memoryListCmd.Flags().StringVar(&memListType, "type", "", "Filter by entity type")
	memoryListCmd.Flags().StringVar(&memListChange, "change", "", "Filter by change")
	memoryListCmd.Flags().StringVar(&memListRecent, "recent", "", "Recent duration (e.g., 7d, 24h)")
	memoryListCmd.Flags().IntVar(&memListLimit, "limit", 20, "Max results")

	// push flags
	memoryPushCmd.Flags().BoolVar(&memPushAll, "all", false, "Re-export all project memories")

	// link flags
	memoryLinkCmd.Flags().StringVar(&memLinkRelation, "relation", "", "Relation type: related-to, caused-by, resolved-by, see-also (required)")

	// note flags
	memoryNoteCmd.Flags().StringVar(&memNoteType, "type", "", "Note type")
	memoryNoteCmd.Flags().StringVar(&memNoteTitle, "title", "", "Title (required)")
	memoryNoteCmd.Flags().StringVar(&memNoteContent, "content", "", "Content (required)")
	memoryNoteCmd.Flags().StringVar(&memNoteTopicKey, "topic-key", "", "Topic key for upsert")
	memoryNoteCmd.Flags().StringVar(&memNoteProjects, "projects", "", "Comma-separated project names")
	memoryNoteCmd.Flags().StringVar(&memNoteTags, "tags", "", "Comma-separated tags")

	memoryCmd.AddCommand(memorySaveCmd)
	memoryCmd.AddCommand(memoryDecideCmd)
	memoryCmd.AddCommand(memorySessionCmd)
	memoryCmd.AddCommand(memorySearchCmd)
	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memoryPushCmd)
	memoryCmd.AddCommand(memoryPullCmd)
	memoryCmd.AddCommand(memoryLinkCmd)
	memoryCmd.AddCommand(memoryNoteCmd)
	memoryCmd.AddCommand(memoryForgetCmd)

	// deprecate flags
	memoryDeprecateCmd.Flags().BoolVarP(&memDeprecateForce, "force", "y", false, "Skip confirmation prompt")

	memoryCmd.AddCommand(memoryDeprecateCmd)
}

func runMemorySave(cmd *cobra.Command, args []string) error {
	if memSaveType == "" || memSaveTitle == "" || memSaveContent == "" {
		ui.PrintError("--type, --title, and --content are required")
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

	m := memory.Memory{
		ID:         uuid.New().String()[:8],
		EntityType: memSaveType,
		Title:      memSaveTitle,
		Content:    memSaveContent,
		Author:     memSaveAuthor,
		Source:     memSaveSource,
		ChangeID:   memSaveChange,
		FileRefs:   memSaveFiles,
		SpecRefs:   memSaveSpecs,
		Tags:       memSaveTags,
		Deprecates: memSaveDeprecates,
		Visibility: memSaveVisibility,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := memory.SaveMemory(db, m); err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}
	ui.PrintSuccess(fmt.Sprintf("saved %s: %s (%s)", m.EntityType, m.Title, m.ID))
	return nil
}

func runMemoryDecide(cmd *cobra.Command, args []string) error {
	if memDecideTitle == "" || memDecideContext == "" || memDecideOutcome == "" || memDecideAlternatives == "" || memDecideRationale == "" {
		ui.PrintError("--title, --context, --outcome, --alternatives, and --rationale are required")
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

	content := fmt.Sprintf("## Context\n%s\n\n## Outcome\n%s\n\n## Alternatives\n%s\n\n## Rationale\n%s",
		memDecideContext, memDecideOutcome, memDecideAlternatives, memDecideRationale)
	if memDecideTradeoffs != "" {
		content += fmt.Sprintf("\n\n## Tradeoffs\n%s", memDecideTradeoffs)
	}

	m := memory.Memory{
		ID:         uuid.New().String()[:8],
		EntityType: "decision",
		Title:      memDecideTitle,
		Content:    content,
		Author:     memSaveAuthor,
		ChangeID:   memDecideChange,
		SpecRefs:   memDecideSpecs,
		Tags:       memDecideTags,
		Deprecates: memDecideDeprecates,
		Status:     "active",
		Visibility: memDecideVisibility,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := memory.SaveMemory(db, m); err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}
	ui.PrintSuccess(fmt.Sprintf("saved decision: %s (%s)", m.Title, m.ID))
	return nil
}

func runMemorySession(cmd *cobra.Command, args []string) error {
	if memSessionGoal == "" || memSessionAccomplished == "" || memSessionNext == "" {
		ui.PrintError("--goal, --accomplished, and --next are required")
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

	summary := fmt.Sprintf("## Goal\n%s\n\n## Accomplished\n%s\n\n## Next Steps\n%s",
		memSessionGoal, memSessionAccomplished, memSessionNext)
	if memSessionDiscoveries != "" {
		summary += fmt.Sprintf("\n\n## Discoveries\n%s", memSessionDiscoveries)
	}
	if memSessionBlockers != "" {
		summary += fmt.Sprintf("\n\n## Blockers\n%s", memSessionBlockers)
	}

	s := memory.Session{
		ID:         uuid.New().String()[:8],
		Mode:       "build",
		ChangeName: memSessionChange,
		Goal:       memSessionGoal,
		Summary:    summary,
	}
	if err := memory.SaveSession(db, s); err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}
	ui.PrintSuccess(fmt.Sprintf("saved session: %s", s.ID))
	return nil
}

func runMemorySearch(cmd *cobra.Command, args []string) error {
	opts := memory.SearchOpts{
		EntityType:        memSearchType,
		ChangeID:          memSearchChange,
		Author:            memSearchAuthor,
		IncludeDeprecated: memSearchDeprecated,
	}

	if memSearchAllProjects {
		globalDB, err := memory.OpenGlobalIndexDB()
		if err != nil {
			ui.PrintError(fmt.Sprintf("opening global index: %v", err))
			return errExitCode1
		}
		defer globalDB.Close()
		results, err := memory.SearchAllProjects(globalDB, args[0], opts)
		if err != nil {
			ui.PrintError(err.Error())
			return errExitCode1
		}
		for _, r := range results {
			fmt.Printf("[%s] %s: %s (%s)\n", r.ProjectName, r.EntityType, r.Title, r.ID)
		}
		if len(results) == 0 {
			ui.PrintMuted("no results")
		}
		return nil
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

	results, err := memory.SearchMemories(db, args[0], opts)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}
	for _, r := range results {
		fmt.Printf("%s: %s (%s)\n", r.EntityType, r.Title, r.ID)
	}
	if len(results) == 0 {
		ui.PrintMuted("no results")
	}
	return nil
}

func runMemoryList(cmd *cobra.Command, args []string) error {
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

	opts := memory.ListOpts{
		EntityType: memListType,
		ChangeID:   memListChange,
		Limit:      memListLimit,
	}
	if memListRecent != "" {
		var d time.Duration
		if strings.HasSuffix(memListRecent, "d") {
			days := strings.TrimSuffix(memListRecent, "d")
			var n int
			fmt.Sscanf(days, "%d", &n)
			d = time.Duration(n) * 24 * time.Hour
		} else {
			var parseErr error
			d, parseErr = time.ParseDuration(memListRecent)
			if parseErr != nil {
				ui.PrintError(fmt.Sprintf("invalid duration: %s", memListRecent))
				return errExitCode1
			}
		}
		opts.Recent = d
	}

	memories, err := memory.ListMemories(db, opts)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}
	for _, m := range memories {
		fmt.Printf("%s: %s (%s) [%s]\n", m.EntityType, m.Title, m.ID, m.CreatedAt)
	}
	if len(memories) == 0 {
		ui.PrintMuted("no memories found")
	}
	return nil
}

func runMemoryPush(cmd *cobra.Command, args []string) error {
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

	result, err := memory.Push(db, filepath.Join(rootDir, "docs"), memPushAll)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}
	ui.PrintSuccess(fmt.Sprintf("pushed %d memories to docs/memory/", result.Exported))
	return nil
}

func runMemoryPull(cmd *cobra.Command, args []string) error {
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

	result, err := memory.Pull(db, filepath.Join(rootDir, "docs"))
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}
	ui.PrintSuccess(fmt.Sprintf("pulled %d memories, skipped %d", result.Imported, result.Skipped))
	if len(result.Conflicts) > 0 {
		for _, c := range result.Conflicts {
			ui.PrintWarning(fmt.Sprintf("conflict: %s — local and shared versions differ", c.ID))
		}
	}
	return nil
}

func runMemoryLink(cmd *cobra.Command, args []string) error {
	if memLinkRelation == "" {
		ui.PrintError("--relation is required (related-to, caused-by, resolved-by, see-also)")
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

	link := memory.MemoryLink{
		FromID:       args[0],
		ToID:         args[1],
		RelationType: memLinkRelation,
	}
	if err := memory.SaveMemoryLink(db, link); err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}
	ui.PrintSuccess(fmt.Sprintf("linked %s -> %s (%s)", args[0], args[1], memLinkRelation))
	return nil
}

func runMemoryNote(cmd *cobra.Command, args []string) error {
	if memNoteTitle == "" || memNoteContent == "" {
		ui.PrintError("--title and --content are required")
		return errExitCode1
	}

	db, err := memory.OpenPersonalDB()
	if err != nil {
		ui.PrintError(fmt.Sprintf("opening personal DB: %v", err))
		return errExitCode1
	}
	defer db.Close()

	id := uuid.New().String()[:8]
	if memNoteTopicKey != "" {
		// Upsert by topic key
		var existingID string
		err := db.QueryRow("SELECT id FROM personal_notes WHERE topic_key = ?", memNoteTopicKey).Scan(&existingID)
		if err == nil {
			id = existingID
		}
	}

	_, err = db.Exec(`INSERT OR REPLACE INTO personal_notes (id, topic_key, title, content, tags, projects, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`,
		id, memNoteTopicKey, memNoteTitle, memNoteContent, memNoteTags, memNoteProjects)
	if err != nil {
		ui.PrintError(fmt.Sprintf("saving note: %v", err))
		return errExitCode1
	}

	ui.PrintSuccess(fmt.Sprintf("saved note: %s (%s)", memNoteTitle, id))
	return nil
}

func runMemoryForget(cmd *cobra.Command, args []string) error {
	db, err := memory.OpenPersonalDB()
	if err != nil {
		ui.PrintError(fmt.Sprintf("opening personal DB: %v", err))
		return errExitCode1
	}
	defer db.Close()

	id := args[0]
	result, err := db.Exec("DELETE FROM personal_notes WHERE id = ?", id)
	if err != nil {
		ui.PrintError(fmt.Sprintf("deleting note: %v", err))
		return errExitCode1
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		ui.PrintError(fmt.Sprintf("note %q not found", id))
		return errExitCode1
	}

	ui.PrintSuccess(fmt.Sprintf("deleted note: %s", id))
	return nil
}

func runMemoryDeprecate(cmd *cobra.Command, args []string) error {
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

	id := args[0]
	m, err := memory.GetMemory(db, id)
	if err != nil {
		ui.PrintError(fmt.Sprintf("memory %q not found", id))
		return errExitCode1
	}

	if !memDeprecateForce {
		confirmed, err := ui.NewConfirmPrompt(fmt.Sprintf("Deprecate memory %q (%s)?", m.Title, id))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := memory.DeprecateMemory(db, id); err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	ui.PrintSuccess(fmt.Sprintf("deprecated memory: %s (%s)", m.Title, id))
	return nil
}
