package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gjunqueira-sys/ScheduleGate/internal/license"
	"github.com/gjunqueira-sys/ScheduleGate/internal/ui"
	"github.com/spf13/cobra"
)

// licenseCmd is the parent command for managing license keys.
var licenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Manage your ScheduleGate license key",
	Long: `View, install, generate, or remove a ScheduleGate license key.

A license key unlocks Pro features (JSON/HTML/CSV/Excel output, the compare
engine, and custom YAML rules). Without a key you run on the free Community
tier, which is limited to 1 assessment per month with terminal output only.`,
}

// licenseInfoCmd prints the current license status.
var licenseInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show the current license status",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		lic := currentLicense
		tier := strings.ToUpper(lic.Tier.String())

		fmt.Println(ui.RenderLogo())
		fmt.Println(ui.RenderTitle("LICENSE STATUS"))

		lines := []string{tier}
		if lic.Valid && lic.Subject != "" {
			lines = append(lines, lic.Subject)
		}
		if lic.Valid && !lic.Expiry.IsZero() {
			lines = append(lines, fmt.Sprintf("expires %s", lic.Expiry.Format("2006-01-02")))
		}
		fmt.Println(ui.CardStyle.Render(
			lipgloss.JoinVertical(lipgloss.Center, lines...),
		))

		key := license.GetStoredKey()
		if key != "" {
			fmt.Printf("Stored key: %s\n", maskKey(key))
			if !lic.Valid {
				fmt.Printf("%s %s\n", ui.FailBadge.String(), lic.Error)
			}
		} else {
			fmt.Println("No license key installed — running on the free Community tier.")
		}
		fmt.Println()
	},
}

// licenseSetCmd stores a license key after validating it.
var licenseSetCmd = &cobra.Command{
	Use:   "set [key]",
	Short: "Validate and store a license key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := license.StoreKey(args[0])
		if err != nil {
			fmt.Printf("%s %v\n", ui.FailBadge.String(), err)
			os.Exit(1)
		}
		currentLicense = result
		fmt.Printf("%s License installed: %s tier\n", ui.PassBadge.String(), strings.ToUpper(result.Tier.String()))
		fmt.Printf("   Subject: %s\n", result.Subject)
		fmt.Printf("   Expires: %s\n", result.Expiry.Format("2006-01-02"))
		if result.Tier == license.TierLifetime {
			fmt.Println("   (lifetime license — never expires)")
		}
	},
}

// licenseClearCmd removes the stored license key.
var licenseClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove the stored license key",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := license.ClearKey(); err != nil {
			fmt.Printf("%s %v\n", ui.FailBadge.String(), err)
			os.Exit(1)
		}
		currentLicense = license.CurrentLicense()
		fmt.Printf("%s License removed — back on the Community tier.\n", ui.PassBadge.String())
	},
}

// licenseGenerateCmd is founder-facing: mint offline license keys. It requires
// the build to embed the license secret.
var licenseGenerateCmd = &cobra.Command{
	Use:    "generate",
	Short:  "Generate a license key (founder tool)",
	Hidden: true,
	Args:   cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		tier, _ := cmd.Flags().GetString("tier")
		email, _ := cmd.Flags().GetString("email")
		months, _ := cmd.Flags().GetInt("months")
		expiryFlag, _ := cmd.Flags().GetString("expiry")

		expiry := time.Now().AddDate(0, months, 0).Format("2006-01-02")
		if expiryFlag != "" {
			expiry = expiryFlag
		}

		key, err := license.GenerateLicenseKey(tier, email, expiry)
		if err != nil {
			fmt.Printf("%s %v\n", ui.FailBadge.String(), err)
			os.Exit(1)
		}

		keyOnly, _ := cmd.Flags().GetBool("key-only")
		if keyOnly {
			fmt.Println(key)
			return
		}

		fmt.Printf("%s Generated %s license for %s (expires %s):\n\n",
			ui.PassBadge.String(), strings.ToUpper(tier), email, expiry)
		fmt.Println("  " + key)
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(licenseCmd)
	licenseCmd.AddCommand(licenseInfoCmd)
	licenseCmd.AddCommand(licenseSetCmd)
	licenseCmd.AddCommand(licenseClearCmd)
	licenseCmd.AddCommand(licenseGenerateCmd)

	licenseGenerateCmd.Flags().String("tier", "", "License tier: pro, team, enterprise, or lifetime")
	licenseGenerateCmd.Flags().String("email", "", "Customer email (embedded as the key subject)")
	licenseGenerateCmd.Flags().Int("months", 12, "Validity period in months from today")
	licenseGenerateCmd.Flags().String("expiry", "", "Exact expiry date in YYYY-MM-DD (overrides --months)")
	licenseGenerateCmd.Flags().Bool("key-only", false, "Print only the raw key, for scripts/CI")
	_ = licenseGenerateCmd.MarkFlagRequired("tier")
	_ = licenseGenerateCmd.MarkFlagRequired("email")
}

// maskKey obscures all but the trailing 6 characters of a license key.
func maskKey(key string) string {
	const visible = 6
	if len(key) <= visible {
		return "******"
	}
	return strings.Repeat("*", len(key)-visible) + key[len(key)-visible:]
}

// tierOf returns the resolved license tier for the current process, used by
// command-level feature gating.
func tierOf() license.Tier {
	return currentLicense.Tier
}
