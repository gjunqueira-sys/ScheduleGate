package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ASCII Art Logo — doom font, generated via pyfiglet
const LogoRaw = ` _____ _____  _   _  ___________ _   _ _      _____ _____   ___ _____ _____ 
/  ___/  __ \| | | ||  ___|  _  \ | | | |    |  ___|  __ \ / _ \_   _|  ___|
\ ` + "`" + `--. | /  \/| |_| || |__ | | | | | | | |    | |__ | |  \// /_\ \| | | |__  
 ` + "`" + `--. \ |    |  _  ||  __|| | | | | | | |    |  __|| | __ |  _  || | |  __| 
/\__/ / \__/\| | | || |___| |/ /| |_| | |____| |___| |_\ \| | | || | | |___ 
\____/ \____/\_| |_/\____/|___/  \___/\_____/\____/ \____/\_| |_/\_/ \____/ 
                        Schedule Quality. Quantified.`

// RenderLogo renders the ASCII logo with a cyan-to-blue gradient.
func RenderLogo() string {
	lines := strings.Split(LogoRaw, "\n")
	var sb strings.Builder

	// Gradient: Cyan 500 → Sky 400 → Blue 400 → Blue 500 → Blue 600; tagline in muted slate
	colors := []lipgloss.Color{
		lipgloss.Color("#06b6d4"), // Cyan 500
		lipgloss.Color("#22d3ee"), // Cyan 400
		lipgloss.Color("#38bdf8"), // Sky 400
		lipgloss.Color("#60a5fa"), // Blue 400
		lipgloss.Color("#3b82f6"), // Blue 500
		lipgloss.Color("#2563eb"), // Blue 600
		lipgloss.Color("#94a3b8"), // Slate 400 — tagline
	}

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		c := colors[i%len(colors)]
		style := lipgloss.NewStyle().Foreground(c).Bold(true)
		sb.WriteString("  ") // Indent
		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	return sb.String()
}
