// Package data provides the data loading layer for the cx dashboard TUI.
// It wraps the internal/memory package and provides a clean interface for
// TUI views to fetch data from all three SQLite databases.
package data

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/amald/cx/internal/memory"
)

// Loader wraps the three SQLite databases and provides data access for TUI views.
// projectDB may be nil if the project has not been initialized.
// globalDB and personalDB may be nil if they cannot be opened.
type Loader struct {
	projectDB   *sql.DB
	globalDB    *sql.DB
	personalDB  *sql.DB
	projectPath string
}

// NewLoader opens all three databases. Call Close() when done.
// Errors opening globalDB or personalDB are not fatal — those DBs are optional.
// An error is returned only if both projectDB and globalDB fail to open,
// indicating a fundamental environment problem.
func NewLoader(projectPath string) (*Loader, error) {
	l := &Loader{projectPath: projectPath}

	// Project DB — may be nil if project not initialized.
	pdb, err := memory.OpenProjectDB(projectPath)
	if err == nil {
		l.projectDB = pdb
	}
	// projectDB open error is non-fatal; views will show empty state.

	// Global index DB — optional.
	gdb, err := memory.OpenGlobalIndexDB()
	if err == nil {
		l.globalDB = gdb
	}

	// Personal DB — optional.
	perdb, err := memory.OpenPersonalDB()
	if err == nil {
		l.personalDB = perdb
	}

	// If both project and global DBs failed, there is nothing to display.
	if l.projectDB == nil && l.globalDB == nil {
		return nil, fmt.Errorf("no databases available: project DB and global index DB both failed to open")
	}

	return l, nil
}

// Close closes all open DB connections.
func (l *Loader) Close() error {
	var errs []error
	if l.projectDB != nil {
		if err := l.projectDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing project DB: %w", err))
		}
	}
	if l.globalDB != nil {
		if err := l.globalDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing global index DB: %w", err))
		}
	}
	if l.personalDB != nil {
		if err := l.personalDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing personal DB: %w", err))
		}
	}
	return errors.Join(errs...)
}

// ProjectReady reports whether the project database is open and available.
func (l *Loader) ProjectReady() bool {
	return l.projectDB != nil
}

// LoadedData holds all data fetched in one poll cycle for the TUI.
type LoadedData struct {
	Stats         memory.MemoryStats
	Memories      []memory.Memory
	Sessions      []memory.Session
	AgentRuns     []memory.AgentRun
	Links         []memory.MemoryLink
	PersonalNotes []memory.PersonalNote
	LoadedAt      time.Time
}

// LoadAll fetches all data needed by the dashboard views in one pass.
// It uses sensible defaults (limit 200) for all list operations.
// Errors from individual loaders are accumulated but do not abort the whole load;
// partial data is returned alongside any error.
func (l *Loader) LoadAll() (*LoadedData, error) {
	d := &LoadedData{LoadedAt: time.Now()}
	var errs []error

	stats, err := l.LoadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("stats: %w", err))
	} else {
		d.Stats = stats
	}

	memories, err := l.LoadMemories(memory.ListOpts{Limit: 200})
	if err != nil {
		errs = append(errs, fmt.Errorf("memories: %w", err))
	} else {
		d.Memories = memories
	}

	sessions, err := l.LoadSessions(memory.SessionListOpts{Limit: 200})
	if err != nil {
		errs = append(errs, fmt.Errorf("sessions: %w", err))
	} else {
		d.Sessions = sessions
	}

	runs, err := l.LoadAgentRuns("")
	if err != nil {
		errs = append(errs, fmt.Errorf("agent runs: %w", err))
	} else {
		d.AgentRuns = runs
	}

	links, err := l.LoadLinks()
	if err != nil {
		errs = append(errs, fmt.Errorf("links: %w", err))
	} else {
		d.Links = links
	}

	notes, err := l.LoadPersonalNotes(200)
	if err != nil {
		errs = append(errs, fmt.Errorf("personal notes: %w", err))
	} else {
		d.PersonalNotes = notes
	}

	return d, errors.Join(errs...)
}

// LoadStats returns aggregate memory counts for the home stats panel.
// Returns a zero-value MemoryStats if the project DB is not available.
func (l *Loader) LoadStats() (memory.MemoryStats, error) {
	if l.projectDB == nil {
		return memory.MemoryStats{}, nil
	}
	return memory.CountMemories(l.projectDB)
}

// LoadMemories returns memories from the project DB according to opts.
// Returns an empty slice if the project DB is not available.
func (l *Loader) LoadMemories(opts memory.ListOpts) ([]memory.Memory, error) {
	if l.projectDB == nil {
		return nil, nil
	}
	return memory.ListMemories(l.projectDB, opts)
}

// LoadSessions returns sessions from the project DB according to opts.
// Returns an empty slice if the project DB is not available.
func (l *Loader) LoadSessions(opts memory.SessionListOpts) ([]memory.Session, error) {
	if l.projectDB == nil {
		return nil, nil
	}
	return memory.ListSessions(l.projectDB, opts)
}

// LoadAgentRuns returns agent runs from the project DB.
// If sessionID is non-empty, only runs for that session are returned.
// Returns an empty slice if the project DB is not available.
func (l *Loader) LoadAgentRuns(sessionID string) ([]memory.AgentRun, error) {
	if l.projectDB == nil {
		return nil, nil
	}
	return memory.ListAgentRuns(l.projectDB, sessionID)
}

// LoadPersonalNotes returns personal notes ordered by updated_at DESC.
// Returns an empty slice if the personal DB is not available.
func (l *Loader) LoadPersonalNotes(limit int) ([]memory.PersonalNote, error) {
	if l.personalDB == nil {
		return nil, nil
	}
	return memory.ListPersonalNotes(l.personalDB, limit)
}

// SearchMemories performs a full-text search against the project DB.
// Returns an empty slice if the project DB is not available.
func (l *Loader) SearchMemories(query string, opts memory.SearchOpts) ([]memory.MemoryResult, error) {
	if l.projectDB == nil {
		return nil, nil
	}
	return memory.SearchMemories(l.projectDB, query, opts)
}

// SearchAllProjects performs a federated full-text search across all registered
// projects via the global index DB.
// Returns an empty slice if the global index DB is not available.
func (l *Loader) SearchAllProjects(query string, opts memory.SearchOpts) ([]memory.ProjectMemoryResult, error) {
	if l.globalDB == nil {
		return nil, nil
	}
	return memory.SearchAllProjects(l.globalDB, query, opts)
}

// DeprecateMemory marks the memory with the given id as deprecated.
// Returns an error if the project DB is not available or the memory is not found.
func (l *Loader) DeprecateMemory(id string) error {
	if l.projectDB == nil {
		return fmt.Errorf("project not initialized")
	}
	return memory.DeprecateMemory(l.projectDB, id)
}

// GetMemory retrieves a single memory by ID from the project DB.
// Returns nil and an error if the project DB is not available or the memory is not found.
func (l *Loader) GetMemory(id string) (*memory.Memory, error) {
	if l.projectDB == nil {
		return nil, fmt.Errorf("project not initialized")
	}
	m, err := memory.GetMemory(l.projectDB, id)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// LoadLinks returns all memory links from the project DB.
// Returns an empty slice if the project DB is not available.
func (l *Loader) LoadLinks() ([]memory.MemoryLink, error) {
	if l.projectDB == nil {
		return nil, nil
	}
	return memory.ListMemoryLinks(l.projectDB)
}
