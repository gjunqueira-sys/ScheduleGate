package cmd

import (
	"fmt"
	"os"

	"github.com/gjunqueira-sys/ScheduleGate/internal/license"
	"github.com/gjunqueira-sys/ScheduleGate/internal/ui"
	"github.com/gjunqueira-sys/ScheduleGate/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// currentLicense is the effective license resolved during the root pre-run
// hook. Subcommands read it for feature gating and license status output.
var currentLicense license.LicenseResult

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "schedulegate",
	Short: "DCMA 14-Point Assessment Tool",
	Long: `A CLI tool to perform DCMA 14-point assessment on Microsoft Project schedules.
It supports Excel (.xlsx) and CSV formats.`,
	Version: version.Long(),
}

// licensePreRun resolves the effective license before any subcommand runs.
// An explicitly-provided --license-key is validated and, when valid, persisted
// so it survives future invocations. An invalid explicit key is a hard error;
// a missing key simply means the free Community tier.
func licensePreRun(cmd *cobra.Command, args []string) error {
	if keyFlag, err := cmd.Flags().GetString("license-key"); err == nil && keyFlag != "" {
		result, storeErr := license.StoreKey(keyFlag)
		if !result.Valid {
			return fmt.Errorf("license-key: %s", result.Error)
		}
		if storeErr != nil {
			return fmt.Errorf("storing license key: %w", storeErr)
		}
		currentLicense = result
		return nil
	}
	currentLicense = license.CurrentLicense()
	return nil
}

func init() {
	rootCmd.SetHelpFunc(ui.HelpFunc)
	rootCmd.SetUsageFunc(ui.UsageFunc)
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.PersistentPreRunE = licensePreRun
}

// RootCmd exposes the fully-assembled command tree (all init() registrations
// applied) for tooling such as cmd/manualcheck, which introspects the CLI
// surface to verify documentation freshness.
func RootCmd() *cobra.Command {
	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.schedulegate.yaml)")
	rootCmd.PersistentFlags().String("license-key", "", "License key to validate and store (SG-...)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".schedulegate" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".schedulegate")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
