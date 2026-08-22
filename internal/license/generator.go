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

// GenerateLicenseKey creates an HMAC-signed license key for offline issuance.
// It is used by the "schedulegate license generate" command (founder-facing)
// and by tests. The resulting key embeds tier, expiry, and customer identity.
func GenerateLicenseKey(tierStr, email, expiryDate string) (string, error) {
	tier, err := ParseTier(tierStr)
	if err != nil {
		return "", err
	}
	if tier == TierCommunity {
		return "", fmt.Errorf("cannot generate a key for the community tier")
	}
	if strings.TrimSpace(email) == "" {
		return "", fmt.Errorf("email is required")
	}
	expiry, err := time.Parse("2006-01-02", expiryDate)
	if err != nil {
		return "", fmt.Errorf("expiry date must be in YYYY-MM-DD format: %w", err)
	}
	if SecretKey == "" {
		return "", fmt.Errorf("license secret key is not configured for this build")
	}

	claims := LicenseClaims{
		Tier: tier.String(),
		Exp:  expiry.Format("2006-01-02"),
		Iat:  time.Now().UTC().Format("2006-01-02"),
		Sub:  strings.TrimSpace(email),
	}

	return generateEncodedKey(claims)
}

// generateEncodedKey serializes claims, HMAC-signs them, and returns the
// "SG-<base64url(payload||signature)>" encoding. Shared by GenerateLicenseKey
// and tests that need to craft arbitrary claims.
func generateEncodedKey(claims LicenseClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, []byte(SecretKey))
	mac.Write(payload)
	signature := mac.Sum(nil)

	raw := append(payload, signature...)
	return keyPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}
