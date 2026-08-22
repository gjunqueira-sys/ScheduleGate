package license

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// ServerOptions configures the license key minting HTTP handler.
type ServerOptions struct {
	// SecretKey is the HMAC secret used to sign keys. Empty disables signing
	// and every mint request fails with a 500.
	SecretKey string
	// AdminToken authenticates POST /api/v1/mint requests (Bearer token).
	AdminToken string
	// WebhookToken authenticates POST /api/v1/webhooks/gumroad requests.
	// Defaults to AdminToken when empty.
	WebhookToken string
	// SMTP settings enable emailing a freshly minted key to the buyer.
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string
	// Now overridable for tests.
	Now func() time.Time
}

// MintRequest is the payload accepted by POST /api/v1/mint.
type MintRequest struct {
	Email  string `json:"email"`
	Tier   string `json:"tier"`   // pro | team | enterprise | lifetime
	Expiry string `json:"expiry"` // optional YYYY-MM-DD; defaults to 1 year out (never for lifetime)
}

type MintResponse struct {
	LicenseKey string `json:"license_key"`
	Tier       string `json:"tier"`
	Expiry     string `json:"expiry"`
}

func (o *ServerOptions) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o *ServerOptions) webhookToken() string {
	if o.WebhookToken != "" {
		return o.WebhookToken
	}
	return o.AdminToken
}

func validToken(expected, token string) bool {
	if expected == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func (o *ServerOptions) mint(tier, email, expiry string) (MintResponse, error) {
	if o.SecretKey == "" {
		return MintResponse{}, fmt.Errorf("license secret key is not configured")
	}
	if o.SecretKey != SecretKey {
		SecretKey = o.SecretKey
	}
	if strings.TrimSpace(email) == "" {
		return MintResponse{}, fmt.Errorf("email is required")
	}
	if tier == "" {
		tier = "pro"
	}
	if expiry == "" {
		if tier == "lifetime" {
			expiry = "2030-01-01" // arbitrary; lifetime keys never expire on validation
		} else {
			expiry = o.now().AddDate(1, 0, 0).Format("2006-01-02")
		}
	}
	key, err := GenerateLicenseKey(tier, email, expiry)
	if err != nil {
		return MintResponse{}, err
	}
	return MintResponse{LicenseKey: key, Tier: strings.ToLower(tier), Expiry: expiry}, nil
}

// Routes returns the http.Handler exposing the license minting API.
func (o *ServerOptions) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/v1/mint", o.handleMint)
	mux.HandleFunc("POST /api/v1/webhooks/gumroad", o.handleGumroadWebhook)
	return logRequests(mux)
}

func (o *ServerOptions) handleMint(w http.ResponseWriter, r *http.Request) {
	if !validToken(o.AdminToken, bearerToken(r)) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if o.SecretKey == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "license secret key is not configured"})
		return
	}
	var req MintRequest
	if err := decodeJSON(w, r, &req, 1<<16); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	resp, err := o.mint(req.Tier, req.Email, req.Expiry)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleGumroadWebhook mints a key for a confirmed Gumroad sale. Gumroad
// POSTs form-encoded or JSON payloads with sale[email] and sale[product_name]
// among other fields; we only need the buyer's email.
//
// Gumroad does not support custom request headers, so the webhook token is
// accepted either as a Bearer header (other clients) or as a ?token= query
// parameter on the webhook URL (Gumroad) — keep that URL secret/unguessable.
func (o *ServerOptions) handleGumroadWebhook(w http.ResponseWriter, r *http.Request) {
	token := firstNonEmpty(bearerToken(r), r.URL.Query().Get("token"))
	if !validToken(o.webhookToken(), token) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	email := firstNonEmpty(r.Form.Get("sale[email]"), r.Form.Get("email"))
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sale[email] missing from webhook payload"})
		return
	}
	resp, err := o.mint("pro", email, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	o.maybeEmailKey(email, resp)
	writeJSON(w, http.StatusOK, resp)
}

func (o *ServerOptions) maybeEmailKey(to string, resp MintResponse) {
	if o.SMTPHost == "" || o.SMTPPort == "" || o.SMTPFrom == "" {
		return
	}
	auth := smtp.PlainAuth("", o.SMTPUser, o.SMTPPass, o.SMTPHost)
	addr := fmt.Sprintf("%s:%s", o.SMTPHost, o.SMTPPort)
	subject := "Your ScheduleGate Pro license key"
	body := fmt.Sprintf("Thanks for buying ScheduleGate Pro.\n\nLicense key: %s\nTier: %s\nExpires: %s\n\nInstall with: schedulegate license set %s\n",
		resp.LicenseKey, resp.Tier, resp.Expiry, resp.LicenseKey)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", o.SMTPFrom, to, subject, body)
	if err := smtp.SendMail(addr, auth, o.SMTPFrom, []string{to}, []byte(msg)); err != nil {
		log.Printf("license server: failed to email key to %s: %v", to, err)
	}
}

func bearerToken(r *http.Request) string {
	parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("license server: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// RunServer starts the minting API from environment variables, blocking until
// shutdown. It is the production entrypoint used by cmd/license-server.
func RunServer() error {
	o := &ServerOptions{
		SecretKey:    os.Getenv("SG_SECRET"),
		AdminToken:   os.Getenv("SG_ADMIN_TOKEN"),
		WebhookToken: os.Getenv("SG_WEBHOOK_TOKEN"),
		SMTPHost:     os.Getenv("SG_SMTP_HOST"),
		SMTPPort:     os.Getenv("SG_SMTP_PORT"),
		SMTPUser:     os.Getenv("SG_SMTP_USER"),
		SMTPPass:     os.Getenv("SG_SMTP_PASS"),
		SMTPFrom:     os.Getenv("SG_SMTP_FROM"),
	}
	if o.SecretKey == "" {
		return fmt.Errorf("SG_SECRET environment variable is required (must match the ldflags secret used to build the CLI)")
	}
	if o.AdminToken == "" {
		return fmt.Errorf("SG_ADMIN_TOKEN environment variable is required")
	}
	port := os.Getenv("SG_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}
	log.Printf("license server listening on :%s", port)
	return http.ListenAndServe(":"+port, o.Routes())
}
