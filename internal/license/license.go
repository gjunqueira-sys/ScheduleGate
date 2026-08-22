package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Tier identifies a license entitlement level.
type Tier int

const (
	// TierCommunity is the free tier. No key required; heavily limited.
	TierCommunity Tier = iota
	// TierPro is the $99/yr tier. Unlocks all output formats and engines.
	TierPro
	// TierTeam is the $299/yr tier. Pro plus CI/CD support for up to 5 users.
	TierTeam
	// TierEnterprise is the $999/yr tier. Unlimited seats in one org.
	TierEnterprise
	// TierLifetime is a $199 one-time purchase, granted Pro entitlements forever.
	TierLifetime
)

var tierNames = map[Tier]string{
	TierCommunity:  "community",
	TierPro:        "pro",
	TierTeam:       "team",
	TierEnterprise: "enterprise",
	TierLifetime:   "lifetime",
}

// String returns the lowercase tier label used inside license keys.
func (t Tier) String() string {
	if name, ok := tierNames[t]; ok {
		return name
	}
	return "unknown"
}

// ParseTier converts a tier label to its Tier value.
func ParseTier(s string) (Tier, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pro":
		return TierPro, nil
	case "team":
		return TierTeam, nil
	case "enterprise":
		return TierEnterprise, nil
	case "lifetime":
		return TierLifetime, nil
	case "community":
		return TierCommunity, nil
	}
	return TierCommunity, fmt.Errorf("unknown tier %q", s)
}

// AllowsJSON reports whether the tier may use --json output. Pro and above.
func (t Tier) AllowsJSON() bool { return t >= TierPro }

// AllowsExports reports whether the tier may write HTML/CSV/Excel outputs.
func (t Tier) AllowsExports() bool { return t >= TierPro }

// AllowsCompare reports whether the tier may run the compare engine.
func (t Tier) AllowsCompare() bool { return t >= TierPro }

// AllowsRules reports whether the tier may run custom YAML pattern rules.
func (t Tier) AllowsRules() bool { return t >= TierPro }

// IsUnlimited reports whether the tier is exempt from the Community-tier
// monthly assessment limit.
func (t Tier) IsUnlimited() bool { return t >= TierPro }

// LicenseClaims is the signed payload embedded in a license key. The HMAC
// signature makes every claim tamper-evident, so a key carries its own
// entitlements and expiry without any server round-trip.
type LicenseClaims struct {
	Tier string `json:"tier"`
	Exp  string `json:"exp"`
	Iat  string `json:"iat"`
	Sub  string `json:"sub"`
}

// LicenseResult is the outcome of validating a license key.
type LicenseResult struct {
	Tier    Tier
	Expiry  time.Time
	Subject string
	Valid   bool
	Error   string
}

const (
	keyPrefix   = "SG-"
	gracePeriod = 7 * 24 * time.Hour
	sigSize     = sha256.Size
)

// ValidateKey verifies the HMAC signature and claims of an encoded license key.
//
// Key layout (after the "SG-" prefix): base64url(payload || hmac_sha256(payload)),
// where payload is the JSON-serialized LicenseClaims.
func ValidateKey(encodedKey string) LicenseResult {
	if encodedKey == "" {
		return LicenseResult{Tier: TierCommunity, Valid: false, Error: "no license key provided"}
	}
	if !strings.HasPrefix(encodedKey, keyPrefix) {
		return LicenseResult{Tier: TierCommunity, Valid: false, Error: "invalid license key: missing SG- prefix"}
	}
	if SecretKey == "" {
		return LicenseResult{Tier: TierCommunity, Valid: false, Error: "license validation is not configured for this build"}
	}

	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(encodedKey, keyPrefix))
	if err != nil {
		return LicenseResult{Tier: TierCommunity, Valid: false, Error: "invalid license key: malformed encoding"}
	}
	if len(raw) <= sigSize {
		return LicenseResult{Tier: TierCommunity, Valid: false, Error: "invalid license key: too short"}
	}

	payload := raw[:len(raw)-sigSize]
	signature := raw[len(raw)-sigSize:]

	mac := hmac.New(sha256.New, []byte(SecretKey))
	mac.Write(payload)
	if !hmac.Equal(mac.Sum(nil), signature) {
		return LicenseResult{Tier: TierCommunity, Valid: false, Error: "invalid license key: signature mismatch"}
	}

	var claims LicenseClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return LicenseResult{Tier: TierCommunity, Valid: false, Error: "invalid license key: unreadable claims"}
	}

	tier, err := ParseTier(claims.Tier)
	if err != nil {
		return LicenseResult{Tier: TierCommunity, Valid: false, Error: "invalid license key: " + err.Error()}
	}

	expiry, err := time.Parse("2006-01-02", claims.Exp)
	if err != nil {
		return LicenseResult{Tier: TierCommunity, Valid: false, Error: "invalid license key: bad expiry date"}
	}

	// Lifetime licenses never expire.
	if tier != TierLifetime && time.Now().After(expiry.Add(gracePeriod)) {
		return LicenseResult{
			Tier:    tier,
			Expiry:  expiry,
			Subject: claims.Sub,
			Valid:   false,
			Error:   fmt.Sprintf("license expired on %s", expiry.Format("2006-01-02")),
		}
	}

	return LicenseResult{
		Tier:    tier,
		Expiry:  expiry,
		Subject: claims.Sub,
		Valid:   true,
	}
}
