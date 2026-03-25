package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/mattn/go-isatty"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Pre-render styled frames; initialized by theme.go's init() to guarantee
// StyleAccent is set before use (spinner.go sorts before theme.go
// alphabetically, so spinner's own init() would run too early).
var styledFrames []string

func isTTY() bool {
	return isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())
}

// RunWithSpinner displays an animated braille spinner on stderr while fn executes.
// It guarantees the spinner is visible for at least minDuration, even if fn
// completes faster. On completion it clears the spinner line entirely.
// The caller is responsible for printing success/error after this returns.
func RunWithSpinner(msg string, minDuration time.Duration, fn func() error) error {
	if !isTTY() {
		return fn()
	}

	var fnErr error
	fnDone := make(chan struct{})
	start := time.Now()

	go func() {
		fnErr = fn()
		close(fnDone)
	}()

	styledMsg := StyleMuted.Render(msg)
	frameIdx := 0

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Fprintf(os.Stderr, "\r\033[K    %s %s", styledFrames[frameIdx%len(styledFrames)], styledMsg)
		frameIdx++

		select {
		case <-fnDone:
			if time.Since(start) >= minDuration {
				fmt.Fprintf(os.Stderr, "\r\033[K")
				return fnErr
			}
		default:
		}
	}

	return fnErr
}

// Pause sleeps for the given duration. No-op in non-TTY environments.
func Pause(d time.Duration) {
	if !isTTY() {
		return
	}
	time.Sleep(d)
}
