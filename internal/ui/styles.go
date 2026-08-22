package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/gjunqueira-sys/ScheduleGate/internal/version"
)

// UI Theme Constants
var (
	// Colors
	ColorPrimary   = lipgloss.Color("33")  // Blue
	ColorSecondary = lipgloss.Color("99")  // Purple
	ColorSuccess   = lipgloss.Color("42")  // Green
	ColorDanger    = lipgloss.Color("196") // Red
	ColorWarning   = lipgloss.Color("214") // Orange
	ColorNeutral   = lipgloss.Color("240") // Grey/Slate
	ColorLight     = lipgloss.Color("255") // White

	// Base Styles
	BaseStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Header Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Background(ColorPrimary).
			Foreground(ColorLight).
			Padding(0, 1).
			MarginBottom(1)

	SectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				MarginTop(1).
				MarginBottom(0)

	// Component Styles
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorNeutral).
			Padding(1, 2).
			MarginBottom(1)

	// Status Badges
	PassBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorLight).
			Background(ColorSuccess).
			Padding(0, 1).
			SetString("PASS")

	FailBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorLight).
			Background(ColorDanger).
			Padding(0, 1).
			SetString("FAIL")

	// Help / Usage Styles
	CommandUsageStyle = lipgloss.NewStyle().
				Foreground(ColorNeutral)

	FlagNameStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess). // Using green for flag names
			Bold(true)

	FlagDescStyle = lipgloss.NewStyle().
			Foreground(ColorNeutral).
			PaddingLeft(2)
)

func Status(passing bool) string {
	if passing {
		return PassBadge.String()
	}
	return FailBadge.String()
}

func RenderTitle(text string) string {
	return TitleStyle.Render(text)
}

// RenderVersion returns a muted terminal line with the CLI version label.
func RenderVersion() string {
	return lipgloss.NewStyle().
		Foreground(ColorNeutral).
		PaddingLeft(2).
		Render(version.Display())
}
