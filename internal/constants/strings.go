package constants

// Menu items
const (
	MenuStats       = "Finished Matches"
	MenuLiveMatches = "Live Matches"
	MenuSettings    = "Settings"
	MenuWorldCup    = "World Cup 2026"

	// MenuConferences replaces MenuWorldCup and MenuSettings in the
	// rendered menu (see internal/ui/menu.go) — both are soccer-specific
	// and orphaned since the ESPN college-football swap. The constants
	// above are kept, not deleted, since the underlying Settings/World Cup
	// views and their code still exist, just unreachable from the menu.
	MenuConferences = "Conferences"
)

// Panel titles
const (
	PanelLiveMatches       = "Live Matches"
	PanelFinishedMatches   = "Finished Matches"
	PanelMatchDetails      = "Match Details"
	PanelMatchList         = "Match List"
	PanelUpcomingMatches   = "Upcoming Matches"
	PanelMinuteByMinute    = "Minute-by-minute"
	PanelMatchStatistics   = "Match Statistics"
	PanelUpdates           = "Updates"
	PanelLeaguePreferences = "League Preferences"

	// American football panels
	PanelSituation     = "Game Situation"
	PanelPlayerLeaders = "Player Leaders"
	PanelMomentum      = "Win Probability"
	PanelConferences   = "Conferences"
)

// Empty state messages
const (
	EmptyNoLiveMatches     = "No live matches"
	EmptyNoFinishedMatches = "No finished matches"
	EmptySelectMatch       = "Select a match"
	EmptyNoUpdates         = "No updates"
	EmptyNoMatches         = "No matches available"
)

// Error messages
const (
	ErrorLoadFailed   = "Unable to load data"
	ErrorMatchDetails = "Unable to load match details"
	ErrorRetryHint    = "r: retry"
)

// Help text
const (
	HelpMainMenu             = "↑/↓: navigate  Enter: select  q: quit"
	HelpConferencesView      = "↑/↓: navigate  Enter: standings  Esc: back"
	HelpMatchesViewUnfocused = "↑/↓: navigate  Tab: scroll details  r: refresh  x: stats  s: standings  f: situation  p: leaders  w: momentum  n: rankings  /: filter  Esc: back  q: quit"
	HelpMatchesViewFocused   = "Tab: unfocus  ↑/↓: scroll  x: stats  s: standings  f: situation  p: leaders  w: momentum  n: rankings  r: refresh  Esc: back  q: quit"
	HelpSettingsView         = "↑/↓: navigate  ←/→: switch tabs  Space: toggle  /: filter  Enter: save  Esc: back"
	HelpStatsView            = "h/l: date range  j/k: navigate  Tab: focus details  ↑/↓: scroll when focused  r: refresh details  /: filter  Esc: back"
	HelpStatsViewUnfocused   = "Tab: focus details"
	HelpStatsViewFocused     = "Tab: unfocus  s: standings  f: situation  p: leaders  w: momentum  n: rankings  x: all statistics  ↑/↓: scroll"
	HelpStandingsDialog      = "Esc: close"
	HelpFormationsDialog     = "Tab/←/→: switch team  Esc: close"
	HelpStatisticsDialog     = "↑/↓: navigate  Esc: close"
	HelpTopScorersDialog     = "↑/↓: navigate  Esc: close"

	// American football dialogs
	HelpSituationDialog = "Esc: close"
	HelpLeadersDialog   = "↑/↓: navigate  Esc: close"
	HelpMomentumDialog  = "Esc: close"
	HelpRankingsDialog  = "↑/↓: navigate  Tab: switch poll  Esc: close"

	// Edge case user-facing hints
	ErrorNoStatistics = "No statistics available yet"
	ErrorNoStandings  = "No standings available"
	ErrorNoSituation  = "No live situation data available"
	ErrorNoLeaders    = "No player statistics available yet"
	ErrorNoMomentum   = "No win probability data available"
	ErrorNoRankings   = "No rankings available"
)

// Status text
const (
	StatusLive            = "LIVE"
	StatusFinished        = "FT"
	StatusNotStarted      = "VS"
	StatusNotStartedShort = "NS"
	StatusFinishedText    = "Finished"
)

// Loading text
const (
	LoadingFetching = "Fetching..."
)

// Notification text
const (
	// NotificationTitleGoal is the title shown in goal notifications.
	NotificationTitleGoal = "⚽ GOLAZO!"
)

// Stats labels
const (
	LabelStatus = "Status: "
	LabelScore  = "Score: "
	LabelLeague = "League: "
	LabelDate   = "Date: "
	LabelVenue  = "Venue: "
)
