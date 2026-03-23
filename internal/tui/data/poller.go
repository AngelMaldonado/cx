package data

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const DefaultPollInterval = 5 * time.Second

// PollMsg is sent when the poll timer fires, signaling the app to refresh data.
type PollMsg struct {
	At time.Time
}

// PollCmd returns a tea.Cmd that waits for the poll interval then sends a PollMsg.
func PollCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return PollMsg{At: t}
	})
}
