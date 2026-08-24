# License Delivery & Store Deployment

How to sell ScheduleGate licenses and deliver `SG-` HMAC-signed keys automatically.
Companion to `docs/Schedule_Benchmark_CLI_Market_Analysis_Rev2.md` §7 (Week 1, Day 4).

## How it works

```
┌─────────────────┐   sale webhook   ┌──────────────────────────┐   mint   ┌──────────────────────┐
│  Gumroad /      │ ───────────────▶ │ license-server (this     │ ───────▶ │ SG-<payload+hmac> key │
│  Lemon Squeezy  │   sale[email]    │  project's mint API)     │          │ delivered to buyer   │
└─────────────────┘                  └──────────────────────────┘          │ (webhook response /  │
                                                                           │  optional SMTP email)│
                                                                           └──────────────────────┘
```

- Keys are **HMAC-SHA256 signed** (`SG-<base64url(payload ‖ hmac)>`), carrying tier + expiry in the payload, so the CLI validates them **fully offline** — no server round-trip at assessment time.
- The minting server (`cmd/license-server`) holds the **same secret** baked into released CLI binaries. A key minted with secret X is only accepted by a binary built with secret X.

## Critical invariant

> **`SG_SECRET` on the server MUST equal the `LICENSE_SECRET` ldflag used to build the released CLI binaries.**

If they drift, every issued key returns `invalid license key: signature mismatch` in the CLI. Pick one secret, store it in the store/Vercel secrets vault, and reuse it on every release build.

## Deploy the license server

Any host that can run a Go binary works (Vercel/Netlify functions, a small VM, Fly.io, Railway). Stdlib only, no external deps, static binary.

```bash
make build-license-server          # → bin/license-server
```

Environment variables:

| Variable | Required | Purpose |
|----------|----------|---------|
| `SG_SECRET` | yes | HMAC signing secret — MUST match CLI `LICENSE_SECRET` |
| `SG_ADMIN_TOKEN` | yes | Bearer token for `POST /api/v1/mint` (manual issuance) |
| `SG_WEBHOOK_TOKEN` | no | Bearer/`?token=` value for the store webhook; defaults to `SG_ADMIN_TOKEN` |
| `SG_SMTP_PASS` | no | Resend API key for emailing keys to buyers (via HTTP API, not SMTP) |
| `SG_SMTP_FROM` | no | From address for delivery emails (e.g. `noreply@schedulegate.dev`) |
| `SG_PORT` | no | Listen port (default `8080`; falls back to `PORT`, then `8080`) |

Endpoints:

| Endpoint | Auth | Purpose |
|----------|------|---------|
| `GET /health` | none | Health probe |
| `POST /api/v1/mint` | `Authorization: Bearer <admin>` | Mint a key: `{"email":"…","tier":"pro","expiry":"YYYY-MM-DD"}`. Tier defaults to `pro`, expiry to +1 year (lifetime → never). |
| `POST /api/v1/webhooks/gumroad` | `?token=` or Bearer | Mint a key from a Gumroad sale (reads `sale[email]`), reply body contains the key |

Test locally:

```bash
SG_SECRET=my-secret SG_ADMIN_TOKEN=tok123 bin/license-server
curl -X POST localhost:8080/api/v1/mint \
  -H "Authorization: Bearer tok123" -H "Content-Type: application/json" \
  -d '{"email":"buyer@example.com","tier":"pro"}'
```

## Option A — Gumroad

## Deploy on Railway (recommended)

Railway builds the repo's `Dockerfile` (multi-stage: `golang` builder → `alpine` runtime with CA certs), auto-injects `PORT`, and gives you a public HTTPS URL.

```bash
brew install railway            # CLI
railway login                   # opens browser auth
railway up                      # deploys from this directory
railway variables set SG_SECRET="<production secret>" SG_ADMIN_TOKEN="<random>" SG_WEBHOOK_TOKEN="<random>"
railway variables set SG_SMTP_PASS="re_..." SG_SMTP_FROM="noreply@schedulegate.dev"   # Resend API key, email delivery
railway domain                  # prints the public https://...railway.app URL
```

Then verify with `curl https://<your-domain>/health` → `{"status":"ok"}`.

1. Create the product (digital), e.g. **ScheduleGate Pro — $99/year**.
2. Do **not** enable Gumroad's built-in license-key generation (keys won't be `SG-` signed).
3. Set the Ping endpoint: *Settings → Advanced → Ping endpoint* — enter `https://<your-server>/api/v1/webhooks/gumroad?token=<SG_WEBHOOK_TOKEN>`. Gumroad cannot send custom headers, so auth rides in the `?token=` query param (constant-time compared); keep the URL unguessable. Bearer headers also work for other clients.
4. Set up email delivery — Gumroad does not surface the webhook response body to the buyer. The server emails the key via Resend's HTTP API using `SG_SMTP_PASS` (your Resend API key) and `SG_SMTP_FROM` (your verified sending domain).
5. Buy the product yourself once (a $1 test variant or use Gumroad's test purchase with a test card) and confirm:
   - webhook fired (`GET /health` logs show a hit),
   - a key was minted for your test email,
   - `schedulegate license set <key>` succeeds and unlocks Pro.

## Option B — Lemon Squeezy

1. Create the product and a variant with an **activation license**; choose "You will generate the license keys" / custom keys if offered, or rely on the webhook minting path below.
2. *Settings → Webhooks → Create* — target the same `/api/v1/webhooks/gumroad` endpoint (the handler reads `sale[email]`, which both stores send; Lemon Squeezy sends `customer_email` too — adjust the handler if you prefer to consume `customer_email`).
3. Subscribe to the `order_created` / `subscription_created` events.

> If you prefer per-store payload parsing, add a second handler: Lemon Squeezy posts JSON with `meta.event_name` and a `data.attributes.user_email`. The existing handler already reads `email` as a fallback for JSON bodies, but only parses form-encoded bodies today — extend `handleGumroadWebhook` (or add `handleLemonSqueezyWebhook`) before relying on it.

## Day-4 acceptance checklist (real test transaction)

- [ ] One secret value used for both the CLI release build and `SG_SECRET`
- [ ] `license-server` deployed, `/health` returns `{"status":"ok"}`
- [ ] Store webhook URL configured (`.../api/v1/webhooks/gumroad?token=<SG_WEBHOOK_TOKEN>`)
- [ ] Bought product with a throwaway card; webhook fired; key minted
- [ ] `schedulegate license set <key>` → PASS, PRO tier
- [ ] `schedulegate assess file.xlsx --json` works (Pro gate lifted)
- [ ] Same key rejected by a differently-signed build (sanity: `signature mismatch`)

## Security notes

- `SG_SECRET` never leaves the server/CI; the CLI embeds it via `-ldflags` (a determined user can extract it — acceptable for an offline dev tool, and the primary control is `ValidateKey`'s tamper-evidence, not secret secrecy).
- `--license-key` and `license set <key>` accept a key argument on the command line; shell history exposure is a minor risk worth noting in docs, not fixing for v1.
- Bearer tokens are compared in constant time (`crypto/subtle`).
- Request bodies are size-limited (16 KiB) on the mint endpoint.
