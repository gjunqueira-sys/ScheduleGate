package license

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// GetStoredKey returns the license key persisted in the config file, or ""
// when none has been stored.
func GetStoredKey() string {
	return viper.GetString("license.key")
}

// StoreKey validates a key and, if valid, persists it in the config file.
// It returns the validation result so callers can surface tier information.
func StoreKey(key string) (LicenseResult, error) {
	result := ValidateKey(key)
	if !result.Valid {
		return result, fmt.Errorf("%s", result.Error)
	}
	viper.Set("license.key", key)
	if err := saveConfig(); err != nil {
		return result, err
	}
	return result, nil
}

// ClearKey removes the stored license key, returning the user to Community.
func ClearKey() error {
	viper.Set("license.key", "")
	return saveConfig()
}

// CurrentLicense returns the effective license for the current run: the
// validated stored key, or the free Community tier when no key is present.
func CurrentLicense() LicenseResult {
	key := GetStoredKey()
	if key == "" {
		return LicenseResult{Tier: TierCommunity, Valid: true}
	}
	return ValidateKey(key)
}

// saveConfig persists viper's in-memory state to the config file. When no
// config file exists yet (first run), it writes a fresh one at the default
// location so later runs can read it back.
func saveConfig() error {
	if err := viper.WriteConfig(); err == nil {
		return nil
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return homeErr
	}
	return viper.WriteConfigAs(filepath.Join(home, ".schedulegate.yaml"))
}

// ---- Monthly usage tracking (Community tier: 1 assessment/month) ----

// CurrentMonthKey returns the current year-month key, e.g. "2026-08".
func CurrentMonthKey() string {
	return time.Now().Format("2006-01")
}

// CheckMonthlyUsage reports whether the Community-tier monthly allowance has
// been exhausted. A fresh calendar month resets the counter. It returns the
// count recorded so far this month so callers can show a useful message.
func CheckMonthlyUsage() (allowed bool, count int, err error) {
	currentMonth := CurrentMonthKey()
	storedMonth := viper.GetString("license.usage_month")
	storedCount := viper.GetInt("license.usage_count")
	if storedMonth != currentMonth {
		storedCount = 0
	}
	return storedCount < 1, storedCount, nil
}

// IncrementMonthlyUsage records one completed assessment for the current
// month. Counters belonging to previous months are discarded.
func IncrementMonthlyUsage() error {
	currentMonth := CurrentMonthKey()
	count := 0
	if viper.GetString("license.usage_month") == currentMonth {
		count = viper.GetInt("license.usage_count")
	}
	viper.Set("license.usage_month", currentMonth)
	viper.Set("license.usage_count", count+1)
	return saveConfig()
}
