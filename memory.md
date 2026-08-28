# Memory

## Soft launch (26 August 2026)

- Phase 0 done (gitignore, leftover `--toggle`, Local_Files ignored).
- Phase 1 done: zip is CLI-only. No `schedulegate-gui`. No MIT. Ships `LICENSE` + `LICENSE-COMMERCIAL` + `user-manual.html`.
- `cmd/versionsync` and `cmd/manualcheck` do **not** assume GUI is in the zip — no code change needed.
- `release.yml` still *builds* the GUI; it is just not zipped. Leave that until a later phase unless asked.
- Phase 2 done: High Float is 44 working days / 5% in user-manual, assess-manual, README, website. Demo uses real DCMA thresholds (not 80% on every row). Dropped “configurable thresholds”, “Docker support”, Team “CI/CD templates”.
- Also fixed garbled Logic threshold in user-manual (`≤ 10%le; 5%` → `≤ 5%`) and assess-manual Logic 10%→5%, Leads 5%→0% so those pages match `metrics.go`.
- Phase 3 done: Pro is the only paid CTA. Team/Lifetime are Coming soon. Install uses `unzip schedulegate.zip` and lists the three CLI binaries. Community skips `license set`. CI example downloads the real zip and extracts `schedulegate-linux`. `/pricing` redirects to `/#pricing`. User manual is copied to `web/docs.html` at deploy (`/docs`).
- Phase 4 drafts written then approved 26 August 2026. Phase 5 done including 5.7 (27 Aug 2026).
- Phase 6 (27 Aug 2026): Legal pages published (effective 27 Aug 2026). Footer: GitHub, Docs, Buy Pro, Issues, Terms, Privacy, Refund + `support@schedulegate.dev`. License-key email includes support + legal URLs. Gumroad *template* has the same URLs. Live Gumroad product page still needs those URLs pasted (no `GUMROAD_*` env in this session). License-server email body change needs a Railway deploy to reach buyers. Website goes live on push to `main` (`web/**` → Vercel).

## Zip contract

`scripts/release/zip-binaries.sh` requires: `schedulegate`, `schedulegate.exe`, `schedulegate-linux`, root `LICENSE`, root `LICENSE-COMMERCIAL`, `docs/user-manual.html`. Fails if any missing. No MIT placeholder.
