// Package notify provides desktop notification functionality for match events.
// Currently supports macOS, Linux, and Windows via the beeep library.
package notify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gen2brain/beeep"
	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/assets"
	"github.com/jaslrobinson/golazo/internal/constants"
	"github.com/jaslrobinson/golazo/internal/data"
)

// notificationTitle returns the scoring-type-appropriate notification title:
// football's touchdown/field_goal/safety (see espncfb/map.go) each get their
// own title, everything else (soccer's "goal", or an unrecognized type)
// falls back to constants.NotificationTitleGoal.
func notificationTitle(eventType string) string {
	switch strings.ToLower(eventType) {
	case "touchdown":
		return "🏈 TOUCHDOWN!"
	case "field_goal":
		return "🏈 FIELD GOAL!"
	case "safety":
		return "🏈 SAFETY!"
	default:
		return constants.NotificationTitleGoal
	}
}

var (
	iconPath     string
	iconPathOnce sync.Once
)

// getIconPath returns the path to the cached notification icon.
// The icon is embedded in the binary and written to the cache directory on first use.
// Returns empty string if caching fails (notification will work without icon).
func getIconPath() string {
	iconPathOnce.Do(func() {
		cacheDir, err := data.CacheDir()
		if err != nil {
			return
		}

		iconPath = filepath.Join(cacheDir, "icon.png")

		// Only write if file doesn't exist
		if _, err := os.Stat(iconPath); os.IsNotExist(err) {
			if err := os.WriteFile(iconPath, assets.Logo, 0644); err != nil {
				iconPath = "" // Reset on write failure
			}
		}
	})
	return iconPath
}

// Notifier defines the interface for sending desktop notifications.
// This allows for easy mocking in tests and potential future implementations.
type Notifier interface {
	// Goal sends a notification for a new goal event.
	Goal(event api.MatchEvent, homeTeam, awayTeam api.Team, homeScore, awayScore int) error
}

// DesktopNotifier implements Notifier using native desktop notifications.
type DesktopNotifier struct {
	enabled bool
}

// NewDesktopNotifier creates a new desktop notifier.
// Notifications are enabled by default.
func NewDesktopNotifier() *DesktopNotifier {
	return &DesktopNotifier{
		enabled: true,
	}
}

// SetEnabled enables or disables notifications.
func (n *DesktopNotifier) SetEnabled(enabled bool) {
	n.enabled = enabled
}

// Enabled returns whether notifications are currently enabled.
func (n *DesktopNotifier) Enabled() bool {
	return n.enabled
}

// Goal sends a desktop notification for a new goal event.
// Includes scorer name, minute, team, and current score.
// Always plays a terminal beep as a fallback notification.
func (n *DesktopNotifier) Goal(event api.MatchEvent, homeTeam, awayTeam api.Team, homeScore, awayScore int) error {
	if !n.enabled {
		return nil
	}

	// Play terminal beep via stderr (bypasses bubbletea's stdout capture)
	// This works even when the TUI is active
	_, _ = os.Stderr.WriteString("\a")

	// Build notification content
	title := notificationTitle(event.Type)
	message := formatGoalMessage(event, homeTeam, awayTeam, homeScore, awayScore)

	// Send notification via beeep (cross-platform)
	// Errors are ignored - OS notification is best-effort, beep already played
	// Icon shows golazo logo on Linux/Windows; macOS shows terminal app icon
	_ = beeep.Notify(title, message, getIconPath())

	return nil
}

// formatGoalMessage creates the notification message for a score.
// Football providers populate Description with the full play text instead
// of a clean Player/Assist split (see api.MatchEvent), so that takes
// priority when present. Format: "Description [Q1 8:55]\nHome 7 - 0 Away"
// for football, "Scorer (Assist) 34' [Team]\nHome 2-1 Away" for soccer.
func formatGoalMessage(event api.MatchEvent, homeTeam, awayTeam api.Team, homeScore, awayScore int) string {
	if event.Description != "" {
		clock := event.DisplayMinute
		if clock == "" {
			clock = fmt.Sprintf("%d'", event.Minute)
		}
		return fmt.Sprintf("%s [%s]\n%s %d - %d %s",
			event.Description,
			clock,
			homeTeam.ShortName,
			homeScore,
			awayScore,
			awayTeam.ShortName,
		)
	}

	scorer := "Unknown"
	if event.Player != nil {
		scorer = *event.Player
	}

	// Determine which team scored
	teamName := event.Team.ShortName
	if teamName == "" {
		teamName = event.Team.Name
	}

	// Build message with assist if available
	assistText := ""
	if event.Assist != nil && *event.Assist != "" {
		assistText = fmt.Sprintf(" (%s)", *event.Assist)
	}

	return fmt.Sprintf("%s%s %d' [%s]\n%s %d - %d %s",
		scorer,
		assistText,
		event.Minute,
		teamName,
		homeTeam.ShortName,
		homeScore,
		awayScore,
		awayTeam.ShortName,
	)
}
