package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/constants"
	"github.com/jaslrobinson/golazo/internal/ui/design"
)

// getReplayIndicator returns the replay link indicator for a goal if available.
func getReplayIndicator(details *api.MatchDetails, goalLinks GoalLinksMap, minute int) string {
	if details == nil || goalLinks == nil {
		return ""
	}
	replayURL := goalLinks.GetReplayURL(details.ID, minute)
	if IsValidReplayURL(replayURL) {
		return CreateGoalLinkDisplay("", replayURL)
	}
	return ""
}

// buildEventContent structures event content with symbol+type adjacent to center time.
func buildEventContent(playerDetails string, replayIndicator string, symbol string, styledTypeLabel string, isHome bool) string {
	if isHome {
		result := playerDetails
		if replayIndicator != "" {
			result += " " + replayIndicator
		}
		return result + " " + symbol + " " + styledTypeLabel
	}
	result := styledTypeLabel + " " + symbol
	if replayIndicator != "" {
		result += " " + replayIndicator
	}
	return result + " " + playerDetails
}

// renderCenterAlignedEvent renders an event with time centered and content expanding outward.
func renderCenterAlignedEvent(minuteStr string, eventContent string, isHomeTeam bool, width int) string {
	timeStyle := lipgloss.NewStyle().Foreground(neonRed).Bold(true)
	styledTime := timeStyle.Render(minuteStr)

	timeWidth := len(minuteStr) + 2
	sideWidth := (width - timeWidth) / 2

	if isHomeTeam {
		leftContent := lipgloss.NewStyle().
			Width(sideWidth).
			Align(lipgloss.Right).
			Render(eventContent)
		rightContent := lipgloss.NewStyle().
			Width(sideWidth).
			Render("")

		return leftContent + " " + styledTime + " " + rightContent
	}

	leftContent := lipgloss.NewStyle().
		Width(sideWidth).
		Align(lipgloss.Right).
		Render("")
	rightContent := lipgloss.NewStyle().
		Width(sideWidth).
		Align(lipgloss.Left).
		Render(eventContent)

	return leftContent + " " + styledTime + " " + rightContent
}

// renderMatchDetailsPanelWithPolling renders the right panel with polling spinner support.
func renderMatchDetailsPanelWithPolling(width, height int, details *api.MatchDetails, liveUpdates []string, sp spinner.Model, loading bool, pollingSpinner *RandomCharSpinner, isPolling bool, goalLinks GoalLinksMap, rightPanelFocused bool, scrollOffset int) string {
	return renderMatchDetailsPanelFull(width, height, details, liveUpdates, sp, loading, true, pollingSpinner, isPolling, goalLinks, rightPanelFocused, scrollOffset)
}

// renderMatchDetailsPanelFull renders the right panel with match details using unified rendering.
// When rightPanelFocused, the Updates feed is windowed to scrollOffset (set via Tab + up/down/j/k
// in the live view — see handleLiveMatchesSelection) instead of being clipped by MaxHeight with no
// way to reach what's cut off, mirroring the Stats view's statsScrollOffset mechanism.
func renderMatchDetailsPanelFull(width, height int, details *api.MatchDetails, liveUpdates []string, sp spinner.Model, loading bool, showTitle bool, pollingSpinner *RandomCharSpinner, isPolling bool, goalLinks GoalLinksMap, rightPanelFocused bool, scrollOffset int) string {
	detailsPanelStyle := lipgloss.NewStyle().Padding(0, 1)
	if rightPanelFocused {
		detailsPanelStyle = detailsPanelStyle.BorderTop(true).BorderBottom(true).BorderForeground(neonCyan)
	}

	if details == nil {
		emptyMessage := lipgloss.NewStyle().
			Foreground(neonDim).
			Align(lipgloss.Center).
			Width(width - 6).
			PaddingTop(1).
			Render(constants.EmptySelectMatch)

		content := emptyMessage
		if showTitle {
			title := design.RenderHeader(constants.PanelMinuteByMinute, width-6)
			content = lipgloss.JoinVertical(lipgloss.Left, title, emptyMessage)
		}

		return detailsPanelStyle.
			Width(width).
			Height(height).
			MaxHeight(height).
			Render(content)
	}

	// Use unified rendering
	cfg := MatchDetailsConfig{
		Width:          width,
		Height:         height,
		Details:        details,
		GoalLinks:      goalLinks,
		ShowStatistics: false,
		ShowHighlights: false,
		LiveUpdates:    liveUpdates,
		PollingSpinner: pollingSpinner,
		IsPolling:      isPolling,
		Loading:        loading,
		Focused:        false,
	}

	headerContent, scrollableContent := RenderMatchDetails(cfg)

	var title string
	if showTitle {
		title = design.RenderHeader(constants.PanelMinuteByMinute, width-6)
	}

	// Window the scrollable Updates feed to the available height, same
	// technique as RenderStatsViewWithList: manual line-slicing rather than
	// letting the outer MaxHeight silently clip content with no way to
	// reach it.
	titleHeight := 0
	if title != "" {
		titleHeight = strings.Count(title, "\n") + 1
	}
	headerHeight := strings.Count(headerContent, "\n") + 1
	availableForScroll := max(height-titleHeight-headerHeight, minScrollableArea)

	scrollableLines := strings.Split(scrollableContent, "\n")
	visibleLines := scrollableLines
	if rightPanelFocused && len(scrollableLines) > availableForScroll {
		start := scrollOffset
		end := min(start+availableForScroll, len(scrollableLines))
		if start < len(scrollableLines) && start >= 0 {
			visibleLines = scrollableLines[start:end]
		} else if start >= len(scrollableLines) {
			visibleLines = []string{}
		}
	} else if len(scrollableLines) > availableForScroll {
		visibleLines = scrollableLines[:availableForScroll]
	}
	visibleContent := strings.Join(visibleLines, "\n")

	var panelContent string
	if title != "" {
		panelContent = lipgloss.JoinVertical(lipgloss.Left, title, headerContent, visibleContent)
	} else {
		panelContent = lipgloss.JoinVertical(lipgloss.Left, headerContent, visibleContent)
	}

	return detailsPanelStyle.
		Width(width).
		Height(height).
		MaxHeight(height).
		Render(panelContent)
}

// extractTeamMarker extracts the [H] or [A] marker from the end of an update string.
func extractTeamMarker(update string) (string, bool) {
	if before, ok := strings.CutSuffix(update, " [H]"); ok {
		return before, true
	}
	if before, ok := strings.CutSuffix(update, " [A]"); ok {
		return before, false
	}
	return update, true
}

// extractMinuteFromUpdate extracts the minute string from a live update.
func extractMinuteFromUpdate(update string) (minute string, rest string) {
	parts := strings.SplitN(update, "' ", 2)
	if len(parts) != 2 {
		return "", update
	}

	firstPart := parts[0]
	lastSpace := strings.LastIndex(firstPart, " ")
	if lastSpace == -1 {
		return "", update
	}

	minute = firstPart[lastSpace+1:] + "'"
	prefix := firstPart[:lastSpace]
	rest = prefix + " " + parts[1]

	return minute, rest
}

// renderStyledLiveUpdate renders a live update string with appropriate colors.
func renderStyledLiveUpdate(update string, contentWidth int, details *api.MatchDetails, goalLinks GoalLinksMap) string {
	if len(update) == 0 {
		return update
	}

	cleanUpdate, isHome := extractTeamMarker(update)
	minute, contentWithoutMinute := extractMinuteFromUpdate(cleanUpdate)
	if minute == "" {
		minute = "0'"
		contentWithoutMinute = cleanUpdate
	}

	runes := []rune(contentWithoutMinute)
	symbol := string(runes[0])
	whiteStyle := lipgloss.NewStyle().Foreground(neonWhite)

	var styledContent string
	switch symbol {
	case "●": // Goal / CFB scoring play - gradient
		// Check for the bracketed label the update string was built with —
		// covers soccer's [GOAL]/[OWN GOAL] and CFB's [TOUCHDOWN]/[FIELD GOAL]/
		// [SAFETY]/[SCORE] (see formatEvent in internal/fotmob/live.go).
		label := "GOAL"
		for _, candidate := range []string{"OWN GOAL", "TOUCHDOWN", "FIELD GOAL", "SAFETY", "SCORE"} {
			if strings.Contains(contentWithoutMinute, "["+candidate+"]") {
				label = candidate
				break
			}
		}
		marker := fmt.Sprintf("[%s]", label)
		playerDetails, _ := extractPlayerAndType(contentWithoutMinute, marker)
		styledType := design.ApplyGradientToText(label)
		styledPlayer := whiteStyle.Render(playerDetails)

		replayIndicator := ""
		if details != nil && goalLinks != nil {
			minuteStr := strings.TrimSuffix(minute, "'")
			if minuteInt, err := strconv.Atoi(minuteStr); err == nil {
				replayIndicator = getReplayIndicator(details, goalLinks, minuteInt)
			}
		}

		styledContent = buildEventContent(styledPlayer, replayIndicator, symbol, styledType, isHome)
	case "▪": // Yellow card
		cardStyle := lipgloss.NewStyle().Foreground(neonYellow).Bold(true)
		playerDetails, _ := extractPlayerAndType(contentWithoutMinute, "[CARD]")
		styledContent = buildEventContent(whiteStyle.Render(playerDetails), "", symbol, cardStyle.Render("CARD"), isHome)
	case "■": // Red card
		cardStyle := lipgloss.NewStyle().Foreground(neonRed).Bold(true)
		playerDetails, _ := extractPlayerAndType(contentWithoutMinute, "[CARD]")
		styledContent = buildEventContent(whiteStyle.Render(playerDetails), "", symbol, cardStyle.Render("CARD"), isHome)
	case "↔": // Substitution
		styledContent = renderSubstitutionWithColorsNoMinute(contentWithoutMinute, isHome)
	case "·": // Other
		dimStyle := lipgloss.NewStyle().Foreground(neonDim)
		playerDetails, _ := extractPlayerAndType(contentWithoutMinute, "")
		styledContent = buildEventContent(dimStyle.Render(playerDetails), "", symbol, "", isHome)
	default:
		styledContent = whiteStyle.Render(contentWithoutMinute)
	}

	return renderCenterAlignedEvent(minute, styledContent, isHome, contentWidth)
}

// extractPlayerAndType extracts player details and type label from event content.
func extractPlayerAndType(content string, typeMarker string) (string, string) {
	if typeMarker == "" {
		runes := []rune(content)
		if len(runes) > 1 {
			return strings.TrimSpace(string(runes[1:])), ""
		}
		return "", ""
	}

	_, after, ok := strings.Cut(content, typeMarker)
	if !ok {
		runes := []rune(content)
		if len(runes) > 1 {
			return strings.TrimSpace(string(runes[1:])), ""
		}
		return "", ""
	}

	return strings.TrimSpace(after), typeMarker
}

// buildSubstitutionContent returns the styled event content for a substitution (no minute).
// Used by both the live view (from parsed update string) and the finished view (from api.MatchEvent).
// playerIn = coming on, playerOut = going off; empty strings are shown as "Unknown".
func buildSubstitutionContent(playerIn, playerOut string, isHome bool) string {
	dimStyle := lipgloss.NewStyle().Foreground(neonDim)
	outStyle := lipgloss.NewStyle().Foreground(neonRed)
	inStyle := lipgloss.NewStyle().Foreground(neonCyan)
	if playerIn == "" {
		playerIn = "Unknown"
	}
	if playerOut == "" {
		playerOut = "Unknown"
	}
	playerDetails := inStyle.Render("←"+playerIn) + " " + outStyle.Render("→"+playerOut)
	return buildEventContent(playerDetails, "", "↔", dimStyle.Render("SUB"), isHome)
}

// renderSubstitutionWithColorsNoMinute renders a substitution from a live-update string (with {OUT}/{IN} markers).
func renderSubstitutionWithColorsNoMinute(update string, isHome bool) string {
	dimStyle := lipgloss.NewStyle().Foreground(neonDim)
	outIdx := strings.Index(update, "{OUT}")
	inIdx := strings.Index(update, "{IN}")
	if outIdx == -1 || inIdx == -1 {
		return dimStyle.Render(update)
	}
	playerOut := strings.TrimSpace(update[outIdx+5 : inIdx])
	playerIn := strings.TrimSpace(update[inIdx+4:])
	return buildSubstitutionContent(playerIn, playerOut, isHome)
}

// renderLargeScore renders the score in a large, prominent format using block digits.
func renderLargeScore(homeScore, awayScore int, width int) string {
	digits := map[int][]string{
		0: {"█▀█", "█ █", "▀▀▀"},
		1: {" █ ", " █ ", " ▀ "},
		2: {"▀▀█", "█▀▀", "▀▀▀"},
		3: {"▀▀█", " ▀█", "▀▀▀"},
		4: {"█ █", "▀▀█", "  ▀"},
		5: {"█▀▀", "▀▀█", "▀▀▀"},
		6: {"█▀▀", "█▀█", "▀▀▀"},
		7: {"▀▀█", "  █", "  ▀"},
		8: {"█▀█", "█▀█", "▀▀▀"},
		9: {"█▀█", "▀▀█", "▀▀▀"},
	}

	dash := []string{"   ", "▀▀▀", "   "}

	getDigitPatterns := func(score int) [][]string {
		if score < 10 {
			return [][]string{digits[score]}
		}
		var patterns [][]string
		scoreStr := fmt.Sprintf("%d", score)
		for _, ch := range scoreStr {
			d := int(ch - '0')
			patterns = append(patterns, digits[d])
		}
		return patterns
	}

	homePatterns := getDigitPatterns(homeScore)
	awayPatterns := getDigitPatterns(awayScore)

	var lines []string
	scoreStyle := lipgloss.NewStyle().Foreground(neonRed).Bold(true)

	for i := range 3 {
		var homeLine strings.Builder
		for j, p := range homePatterns {
			if j > 0 {
				homeLine.WriteString(" ")
			}
			homeLine.WriteString(p[i])
		}

		var awayLine strings.Builder
		for j, p := range awayPatterns {
			if j > 0 {
				awayLine.WriteString(" ")
			}
			awayLine.WriteString(p[i])
		}

		line := homeLine.String() + "  " + dash[i] + "  " + awayLine.String()
		lines = append(lines, scoreStyle.Render(line))
	}

	scoreBlock := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(scoreBlock)
}
