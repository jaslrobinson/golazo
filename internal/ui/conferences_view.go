package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/constants"
	"github.com/jaslrobinson/golazo/internal/ui/design"
)

// conferencesBoxWidth matches settingsBoxWidth's role: a fixed content width
// for this view, independent of terminal size, consistent with how the
// Settings view (which this replaces in the main menu) sizes itself.
const conferencesBoxWidth = 48

// RenderConferencesView renders a browsable list of conferences. Selecting
// one (Enter) fetches and opens its standings in the existing Standings
// dialog — this view has no data of its own beyond the static conference
// list; it's a way to reach standings without going through a match first.
func RenderConferencesView(width, height, selectedIdx int, conferences []api.League, bannerType constants.StatusBannerType) string {
	statusBanner := renderStatusBanner(bannerType, conferencesBoxWidth)
	if statusBanner != "" {
		statusBanner += "\n"
	}

	title := design.RenderHeader(constants.PanelConferences, conferencesBoxWidth)

	var listContent string
	if len(conferences) == 0 {
		listContent = dialogDimStyle.Render("No conferences available")
	} else {
		items := make([]string, 0, len(conferences))
		for i, conf := range conferences {
			if i == selectedIdx {
				items = append(items, menuItemSelectedStyle.Render(conf.Name))
			} else {
				items = append(items, menuItemStyle.Render(conf.Name))
			}
		}
		listContent = lipgloss.JoinVertical(lipgloss.Left, items...)
	}
	listContent = lipgloss.NewStyle().Width(conferencesBoxWidth).Render(listContent)

	help := neonDimStyle.Width(conferencesBoxWidth).Align(lipgloss.Center).Render(constants.HelpConferencesView)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		statusBanner,
		title,
		"",
		listContent,
		"",
		help,
	)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}
