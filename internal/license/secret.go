package license

// SecretKey is the HMAC-SHA256 signing key used to validate license keys.
// It is injected at build time via ldflags so it never lives in source control:
//
//	go build -ldflags "-X github.com/gjunqueira-sys/ScheduleGate/internal/license.SecretKey=<key>"
//
// When empty (dev builds), license-key validation is disabled and the binary
// runs as the free Community tier.
var SecretKey string
