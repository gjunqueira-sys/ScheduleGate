package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// HelpFunc returns a custom help function for Cobra commands.
func HelpFunc(cmd *cobra.Command, args []string) {
	// Logo
	fmt.Println(RenderLogo())
	fmt.Println(lipgloss.NewStyle().Bold(true).Padding(0, 2).Render("DCMA 14-Point Assessment Tool"))
	fmt.Println(RenderVersion())
	fmt.Println()

	if cmd.Short != "" {
		fmt.Println(lipgloss.NewStyle().Foreground(ColorNeutral).Render(cmd.Short))
		fmt.Println()
	}

	// Usage
	fmt.Println(SectionHeaderStyle.Render("USAGE"))
	if cmd.Runnable() {
		fmt.Printf("  %s %s\n", cmd.CommandPath(), "[flags]")
	} else {
		fmt.Printf("  %s %s\n", cmd.CommandPath(), "[command]")
	}

	// Commands
	if len(cmd.Commands()) > 0 {
		fmt.Println(SectionHeaderStyle.Render("COMMANDS"))
		for _, c := range cmd.Commands() {
			if !c.Hidden && c.IsAvailableCommand() {
				fmt.Printf("  %-15s %s\n",
					lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(c.Name()),
					lipgloss.NewStyle().Foreground(ColorNeutral).Render(c.Short))
			}
		}
	}

	// Flags
	if cmd.Flags().HasFlags() {
		fmt.Println(SectionHeaderStyle.Render("FLAGS"))
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}

			var flagName string
			if f.Shorthand != "" {
				flagName = fmt.Sprintf("-%s, --%s", f.Shorthand, f.Name)
			} else {
				flagName = fmt.Sprintf("    --%s", f.Name)
			}

			// Render
			fmt.Printf("  %-30s\n", FlagNameStyle.Render(flagName))

			// Wraps description if needed
			desc := f.Usage
			if f.DefValue != "" {
				desc += fmt.Sprintf(" (default: %s)", f.DefValue)
			}
			fmt.Printf("      %s\n\n", FlagDescStyle.Render(desc))
		})
	}
}

// UsageFunc can be used to override the default usage error message.
func UsageFunc(cmd *cobra.Command) error {
	HelpFunc(cmd, nil)
	return nil
}
