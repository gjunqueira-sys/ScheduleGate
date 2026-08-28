package license

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newServer(t *testing.T) (*ServerOptions, *httptest.Server) {
	t.Helper()
	o := &ServerOptions{
		SecretKey:    "server-test-secret",
		AdminToken:   "admin-token",
		WebhookToken: "webhook-token",
		Now:          func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC) },
	}
	ts := httptest.NewServer(o.Routes())
	t.Cleanup(ts.Close)
	return o, ts
}

func doRequest(t *testing.T, ts *httptest.Server, method, path, token, body, contentType string) *http.Response {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, ts.URL+path, reader)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return out
}

func TestHealth(t *testing.T) {
	_, ts := newServer(t)
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestMint_RequiresAuth(t *testing.T) {
	_, ts := newServer(t)
	resp := doRequest(t, ts, "POST", "/api/v1/mint", "", `{"email":"a@b.com"}`, "application/json")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestMint_RejectsBadToken(t *testing.T) {
	_, ts := newServer(t)
	resp := doRequest(t, ts, "POST", "/api/v1/mint", "wrong", `{"email":"a@b.com"}`, "application/json")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestMint_Success(t *testing.T) {
	_, ts := newServer(t)
	resp := doRequest(t, ts, "POST", "/api/v1/mint", "admin-token",
		`{"email":"buyer@example.com","tier":"pro","expiry":"2027-08-16"}`, "application/json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	key, _ := body["license_key"].(string)
	if !strings.HasPrefix(key, "SG-") {
		t.Errorf("license_key = %q, want SG- prefix", key)
	}
	SecretKey = "server-test-secret"
	got := ValidateKey(key)
	if !got.Valid {
		t.Fatalf("minted key does not validate: %+v", got)
	}
	if got.Subject != "buyer@example.com" {
		t.Errorf("subject = %q, want buyer@example.com", got.Subject)
	}
	if got.Tier != TierPro {
		t.Errorf("tier = %v, want pro", got.Tier)
	}
}

func TestMint_DefaultsExpiryToOneYear(t *testing.T) {
	o, ts := newServer(t)
	resp := doRequest(t, ts, "POST", "/api/v1/mint", "admin-token",
		`{"email":"buyer@example.com"}`, "application/json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if exp := body["expiry"]; exp != "2027-08-16" {
		t.Errorf("default expiry = %v, want 2027-08-16 (Now + 1yr)", exp)
	}
	_ = o
}

func TestMint_LifetimeDoesNotExpire(t *testing.T) {
	_, ts := newServer(t)
	resp := doRequest(t, ts, "POST", "/api/v1/mint", "admin-token",
		`{"email":"buyer@example.com","tier":"lifetime"}`, "application/json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	key, _ := body["license_key"].(string)
	SecretKey = "server-test-secret"
	got := ValidateKey(key)
	if !got.Valid {
		t.Fatalf("lifetime key does not validate: %+v", got)
	}
}

func TestMint_RejectsUnknownTier(t *testing.T) {
	_, ts := newServer(t)
	resp := doRequest(t, ts, "POST", "/api/v1/mint", "admin-token",
		`{"email":"a@b.com","tier":"gold"}`, "application/json")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestMint_RejectsMissingEmail(t *testing.T) {
	_, ts := newServer(t)
	resp := doRequest(t, ts, "POST", "/api/v1/mint", "admin-token",
		`{"tier":"pro"}`, "application/json")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestMint_FailsWhenSecretUnset(t *testing.T) {
	o := &ServerOptions{AdminToken: "admin-token"}
	ts := httptest.NewServer(o.Routes())
	defer ts.Close()
	resp := doRequest(t, ts, "POST", "/api/v1/mint", "admin-token",
		`{"email":"a@b.com"}`, "application/json")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}

func TestGumroadWebhook_MintsKey(t *testing.T) {
	_, ts := newServer(t)
	form := url.Values{"sale[email]": {"webhook@example.com"}, "sale[product_name]": {"ScheduleGate Pro"}}
	resp := doRequest(t, ts, "POST", "/api/v1/webhooks/gumroad", "webhook-token", form.Encode(), "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	key, _ := body["license_key"].(string)
	SecretKey = "server-test-secret"
	got := ValidateKey(key)
	if !got.Valid {
		t.Fatalf("webhook-minted key does not validate: %+v", got)
	}
	if got.Subject != "webhook@example.com" {
		t.Errorf("subject = %q, want webhook@example.com", got.Subject)
	}
}

func TestGumroadWebhook_RejectsBadToken(t *testing.T) {
	_, ts := newServer(t)
	resp := doRequest(t, ts, "POST", "/api/v1/webhooks/gumroad", "wrong", "sale%5Bemail%5D=a%40b.com", "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestGumroadWebhook_TokenViaQuery(t *testing.T) {
	_, ts := newServer(t)
	form := url.Values{"sale[email]": {"qparam@example.com"}}
	resp := doRequest(t, ts, "POST", "/api/v1/webhooks/gumroad?token=webhook-token", "", form.Encode(), "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	key, _ := body["license_key"].(string)
	SecretKey = "server-test-secret"
	got := ValidateKey(key)
	if !got.Valid {
		t.Fatalf("webhook-minted key does not validate: %+v", got)
	}
	if got.Subject != "qparam@example.com" {
		t.Errorf("subject = %q, want qparam@example.com", got.Subject)
	}
}

func TestGumroadWebhook_RejectsBadQueryToken(t *testing.T) {
	_, ts := newServer(t)
	form := url.Values{"sale[email]": {"a@b.com"}}
	resp := doRequest(t, ts, "POST", "/api/v1/webhooks/gumroad?token=wrong", "", form.Encode(), "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestGumroadWebhook_NoTokenAtAll(t *testing.T) {
	_, ts := newServer(t)
	resp := doRequest(t, ts, "POST", "/api/v1/webhooks/gumroad", "", "sale%5Bemail%5D=a%40b.com", "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestLicenseKeyEmailBody(t *testing.T) {
	body := licenseKeyEmailBody(MintResponse{
		LicenseKey: "SG-TESTKEY",
		Tier:       "pro",
		Expiry:     "2027-08-27",
	})
	for _, want := range []string{
		"SG-TESTKEY",
		"pro",
		"2027-08-27",
		"schedulegate license set SG-TESTKEY",
		"support@schedulegate.dev",
		"https://schedulegate.dev/terms",
		"https://schedulegate.dev/privacy",
		"https://schedulegate.dev/refund",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("license email missing %q", want)
		}
	}
}

func TestGumroadWebhook_MissingEmail(t *testing.T) {
	_, ts := newServer(t)
	resp := doRequest(t, ts, "POST", "/api/v1/webhooks/gumroad", "webhook-token", "", "application/x-www-form-urlencoded")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}
