package license

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func setSecret(t *testing.T, s string) {
	t.Helper()
	SecretKey = s
}

func TestValidateKey_Valid(t *testing.T) {
	setSecret(t, "test-secret")
	key, err := GenerateLicenseKey("pro", "customer@example.com", "2027-01-01")
	if err != nil {
		t.Fatalf("GenerateLicenseKey returned error: %v", err)
	}

	got := ValidateKey(key)
	if !got.Valid {
		t.Fatalf("ValidateKey = %+v, want valid", got)
	}
	if got.Tier != TierPro {
		t.Errorf("Tier = %v, want pro", got.Tier)
	}
	if got.Subject != "customer@example.com" {
		t.Errorf("Subject = %q, want customer@example.com", got.Subject)
	}
	if !got.Expiry.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Expiry = %v, want 2027-01-01", got.Expiry)
	}
	if got.Error != "" {
		t.Errorf("Error = %q, want empty", got.Error)
	}
}

func TestValidateKey_AllTiers(t *testing.T) {
	setSecret(t, "test-secret")
	for _, tier := range []string{"pro", "team", "enterprise", "lifetime"} {
		t.Run(tier, func(t *testing.T) {
			key, err := GenerateLicenseKey(tier, "a@b.com", "2030-01-01")
			if err != nil {
				t.Fatalf("GenerateLicenseKey: %v", err)
			}
			got := ValidateKey(key)
			if !got.Valid {
				t.Fatalf("ValidateKey = %+v, want valid", got)
			}
			if got.Tier.String() != tier {
				t.Errorf("Tier = %q, want %q", got.Tier.String(), tier)
			}
		})
	}
}

func TestValidateKey_Expired(t *testing.T) {
	setSecret(t, "test-secret")
	key, err := GenerateLicenseKey("pro", "a@b.com", "2020-01-01")
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}
	got := ValidateKey(key)
	if got.Valid {
		t.Fatalf("ValidateKey = %+v, want invalid (expired)", got)
	}
	if !strings.Contains(got.Error, "expired") {
		t.Errorf("Error = %q, want to mention expiry", got.Error)
	}
}

func TestValidateKey_GracePeriod(t *testing.T) {
	setSecret(t, "test-secret")
	// A key that expires 3 days from now is inside the 7-day grace period.
	exp := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	key, err := GenerateLicenseKey("pro", "a@b.com", exp)
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}
	got := ValidateKey(key)
	if !got.Valid {
		t.Fatalf("ValidateKey = %+v, want valid during grace period", got)
	}
}

func TestValidateKey_LifetimeNeverExpires(t *testing.T) {
	setSecret(t, "test-secret")
	key, err := GenerateLicenseKey("lifetime", "a@b.com", "2000-01-01")
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}
	got := ValidateKey(key)
	if !got.Valid {
		t.Fatalf("ValidateKey = %+v, want valid (lifetime never expires)", got)
	}
	if got.Tier != TierLifetime {
		t.Errorf("Tier = %v, want lifetime", got.Tier)
	}
}

func TestValidateKey_Tampered(t *testing.T) {
	setSecret(t, "test-secret")
	key, err := GenerateLicenseKey("pro", "a@b.com", "2030-01-01")
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}

	// Flip a payload character: replace the tier with "team" by rebuilding.
	tampered := key[:len(key)-4] + "AAAA"
	got := ValidateKey(tampered)
	if got.Valid {
		t.Fatalf("ValidateKey = %+v, want invalid (tampered)", got)
	}
}

func TestValidateKey_WrongSecret(t *testing.T) {
	setSecret(t, "correct-secret")
	key, err := GenerateLicenseKey("pro", "a@b.com", "2030-01-01")
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}

	setSecret(t, "different-secret")
	got := ValidateKey(key)
	if got.Valid {
		t.Fatalf("ValidateKey = %+v, want invalid (wrong secret)", got)
	}
	if !strings.Contains(got.Error, "signature mismatch") {
		t.Errorf("Error = %q, want signature mismatch", got.Error)
	}
}

func TestValidateKey_NoSecretConfigured(t *testing.T) {
	setSecret(t, "")
	got := ValidateKey("SG-anything")
	if got.Valid {
		t.Fatalf("ValidateKey = %+v, want invalid (no secret)", got)
	}
}

func TestValidateKey_InvalidPrefix(t *testing.T) {
	setSecret(t, "test-secret")
	got := ValidateKey("XX-foo")
	if got.Valid {
		t.Fatalf("ValidateKey = %+v, want invalid", got)
	}
	if !strings.Contains(got.Error, "SG-") {
		t.Errorf("Error = %q, want mention of SG- prefix", got.Error)
	}
}

func TestValidateKey_EmptyKey(t *testing.T) {
	setSecret(t, "test-secret")
	got := ValidateKey("")
	if got.Valid {
		t.Fatalf("ValidateKey = %+v, want invalid", got)
	}
	if got.Tier != TierCommunity {
		t.Errorf("Tier = %v, want community", got.Tier)
	}
}

func TestValidateKey_InvalidBase64(t *testing.T) {
	setSecret(t, "test-secret")
	got := ValidateKey("SG-%%%not-base64%%%")
	if got.Valid {
		t.Fatalf("ValidateKey = %+v, want invalid", got)
	}
}

func TestValidateKey_TooShort(t *testing.T) {
	setSecret(t, "test-secret")
	got := ValidateKey("SG-AA")
	if got.Valid {
		t.Fatalf("ValidateKey = %+v, want invalid", got)
	}
}

func TestValidateKey_InvalidTier(t *testing.T) {
	setSecret(t, "test-secret")
	// Build a key with an unknown tier by crafting claims manually.
	claims := LicenseClaims{Tier: "platinum", Exp: "2030-01-01", Iat: "2026-01-01", Sub: "a@b.com"}
	raw, err := signClaims(claims)
	if err != nil {
		t.Fatalf("signClaims: %v", err)
	}
	got := ValidateKey(raw)
	if got.Valid {
		t.Fatalf("ValidateKey = %+v, want invalid (unknown tier)", got)
	}
}

func TestGenerate_CommunityRejected(t *testing.T) {
	setSecret(t, "test-secret")
	if _, err := GenerateLicenseKey("community", "a@b.com", "2030-01-01"); err == nil {
		t.Fatal("GenerateLicenseKey(community) = nil error, want error")
	}
}

func TestGenerate_MissingEmail(t *testing.T) {
	setSecret(t, "test-secret")
	if _, err := GenerateLicenseKey("pro", "", "2030-01-01"); err == nil {
		t.Fatal("GenerateLicenseKey with empty email = nil error, want error")
	}
}

func TestGenerate_BadDate(t *testing.T) {
	setSecret(t, "test-secret")
	if _, err := GenerateLicenseKey("pro", "a@b.com", "not-a-date"); err == nil {
		t.Fatal("GenerateLicenseKey with bad date = nil error, want error")
	}
}

func TestParseTier(t *testing.T) {
	cases := map[string]Tier{
		"pro": TierPro, "Pro": TierPro, " PRO ": TierPro,
		"team": TierTeam, "enterprise": TierEnterprise, "lifetime": TierLifetime,
		"community": TierCommunity,
	}
	for in, want := range cases {
		got, err := ParseTier(in)
		if err != nil {
			t.Errorf("ParseTier(%q) returned error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseTier(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := ParseTier("nope"); err == nil {
		t.Error("ParseTier(\"nope\") = nil error, want error")
	}
}

func TestTierHelpers(t *testing.T) {
	if TierCommunity.AllowsJSON() || TierCommunity.AllowsExports() || TierCommunity.AllowsCompare() || TierCommunity.AllowsRules() || TierCommunity.IsUnlimited() {
		t.Error("Community tier should allow nothing")
	}
	for _, tier := range []Tier{TierPro, TierTeam, TierEnterprise, TierLifetime} {
		if !tier.AllowsJSON() || !tier.AllowsExports() || !tier.AllowsCompare() || !tier.AllowsRules() || !tier.IsUnlimited() {
			t.Errorf("%v tier should allow all premium features", tier)
		}
	}
}

// --- store tests (isolated via temp viper config) ---

func TestStoreAndRetrieveKey(t *testing.T) {
	setSecret(t, "test-secret")
	cfg := t.TempDir() + "/cfg.yaml"
	viper.Reset()
	viper.SetConfigFile(cfg)
	viper.SetConfigType("yaml")

	key, err := GenerateLicenseKey("team", "stored@example.com", "2030-01-01")
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}

	if _, err := StoreKey(key); err != nil {
		t.Fatalf("StoreKey: %v", err)
	}
	if got := GetStoredKey(); got != key {
		t.Errorf("GetStoredKey = %q, want %q", got, key)
	}

	lic := CurrentLicense()
	if !lic.Valid || lic.Tier != TierTeam {
		t.Errorf("CurrentLicense = %+v, want valid team", lic)
	}
}

func TestStoreInvalidKey(t *testing.T) {
	setSecret(t, "test-secret")
	viper.Reset()
	viper.SetConfigFile(t.TempDir() + "/cfg.yaml")
	viper.SetConfigType("yaml")

	if _, err := StoreKey("SG-bogus"); err == nil {
		t.Fatal("StoreKey(SG-bogus) = nil error, want error")
	}
}

func TestClearKey(t *testing.T) {
	setSecret(t, "test-secret")
	viper.Reset()
	viper.SetConfigFile(t.TempDir() + "/cfg.yaml")
	viper.SetConfigType("yaml")

	key, _ := GenerateLicenseKey("pro", "a@b.com", "2030-01-01")
	_, _ = StoreKey(key)

	if err := ClearKey(); err != nil {
		t.Fatalf("ClearKey: %v", err)
	}
	if got := CurrentLicense(); got.Valid && got.Tier != TierCommunity {
		t.Errorf("CurrentLicense after clear = %+v, want community", got)
	}
}

func TestMonthlyUsage_Limit(t *testing.T) {
	setSecret(t, "test-secret")
	viper.Reset()
	viper.SetConfigFile(t.TempDir() + "/cfg.yaml")
	viper.SetConfigType("yaml")

	allowed, count, err := CheckMonthlyUsage()
	if err != nil || !allowed || count != 0 {
		t.Fatalf("first CheckMonthlyUsage = (%v, %d, %v), want (true, 0, nil)", allowed, count, err)
	}
	if err := IncrementMonthlyUsage(); err != nil {
		t.Fatalf("IncrementMonthlyUsage: %v", err)
	}

	allowed, count, err = CheckMonthlyUsage()
	if err != nil || allowed || count != 1 {
		t.Fatalf("second CheckMonthlyUsage = (%v, %d, %v), want (false, 1, nil)", allowed, count, err)
	}
}

func TestMonthlyUsage_NewMonthResets(t *testing.T) {
	setSecret(t, "test-secret")
	viper.Reset()
	viper.SetConfigFile(t.TempDir() + "/cfg.yaml")
	viper.SetConfigType("yaml")

	_ = IncrementMonthlyUsage()
	// Simulate the previous month on disk.
	viper.Set("license.usage_month", "1999-01")
	viper.Set("license.usage_count", 99)

	allowed, count, err := CheckMonthlyUsage()
	if err != nil || !allowed || count != 0 {
		t.Fatalf("CheckMonthlyUsage after month rollover = (%v, %d, %v), want (true, 0, nil)", allowed, count, err)
	}
}

// signClaims is a test helper that produces a raw encoded key (without the
// SG- prefix) for arbitrary claims, mirroring the production generator.
func signClaims(claims LicenseClaims) (string, error) {
	return generateEncodedKey(claims)
}
