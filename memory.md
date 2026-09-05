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
- Phase 6 (27 Aug 2026): Legal pages published (effective 27 Aug 2026). Footer: GitHub, Docs, Buy Pro, Issues, Terms, Privacy, Refund + `support@schedulegate.dev`. License-key email includes support + legal URLs. Gumroad product description updated via browser with the legal section (Terms/Privacy/Refund/support). **Also aligned Gumroad refund policy from 30-day → 14-day** to match published legal pages. License-server email body change needs a Railway deploy to reach buyers. Website goes live on push to `main` (`web/**` → Vercel).
- Phase 7 (27 Aug 2026): 7.1 `go test ./...` pass. 7.2 `go vet ./...` pass. 7.3 website: Pro-only Buy CTA, Team/Lifetime Coming soon, no Enterprise, `unzip schedulegate.zip`, three CLI binaries, Community skips license set, `/pricing` → `/#pricing`, footer `/docs`, “No GUI required” is a tagline not a product. 7.4 zip script: three CLI bins, LICENSE + LICENSE-COMMERCIAL, no GUI, no MIT. 7.5 gate open (0–6 including 4.5 and 5.6). Next: dry-run then v1.0.6 release.
- Gumroad content-file gotcha (28 Aug 2026): the product deliverable is rendered **inside the Content-tab rich-text document** as a `.node-fileEmbed` blobject. Overwriting the contenteditable `innerHTML` (as done during the description rewrite) **deletes** the file node → the zip becomes detached (user reported "v1.0.6 file not attached/saved"). Fix: re-insert via toolbar `Upload files` → `Computer files` file chooser (NOT programmatic `input.setInputFiles`, which created a 0-byte record). Give the multipart PUT to S3 time to finish BEFORE clicking Save changes.
- Gumroad editor "0 byte" display quirk: the file node's Download link is a `/product_files_utility/...` proxy with no size, so the editor always renders "0 byte" even though the stored record has the true size. Authoritative check = embedded product JSON (`file_size` field). For our v1.0.6 node it is `14304026` bytes, `status: saved`, real S3 URL — confirmed correct. Do NOT re-delete based on the "0 byte" text.
- Phase 7 release (28 Aug 2026): dry-run release.yml (run 33132171165) passed, then real **v1.0.6** (run 33132435560) — GitHub Release + Vercel deploy published. commit 285928e `chore: bump version to v1.0.6` (versionsync apply + manualcheck clean). Zip verified from the actual Release: 7 files, CLI-only, AGPLv3 + LICENSE-COMMERCIAL. Live site: `/pricing` 307 → `/#pricing`, `/docs` 200. Gumroad updated in-browser: old v1.0.5 file deleted, `schedulegate-v1.0.6.zip` (13.6 MB) uploaded, Content description rewritten (v1.0.6 CLI package, quick start, what's new, legal links), fixed stray `14 days it` prefix in the Product-page description, and re-aligned refund policy to **14-day** (it had reverted to 30-day somewhere between the 27 Aug session and today) → both sets of changes re-saved with “Changes saved!” confirmed. Task 7.9 (second test purchase) skipped. **Open items: (1) Railway deploy of the license-server email-body change; (2) optional cleanup of old `schedulegate-1.0.2` product-file library record (not attached, harmless but stale); (3) optional end-to-end test purchase against v1.0.6 to confirm the downloadable zip is the real 13.6 MB CLI-only package.**

## Go-live marketing (28 Aug 2026)

Plan is in `Local_Files_do_not_track_do_not_commit/go-live-marketing-plan.md` (gitignored). Sell Pro $99 only. No GUI, no P6 XER claims. Reddit = help-first + disclose. Community is the demo.

`marketing-execution-kit.md` (gitignored, same folder) holds all approved-copy drafts + tracker. LinkedIn runs via the **company page, never the user's personal profile**; nothing gets posted without the user's explicit per-item Go.

LinkedIn Company Page (29 Aug 2026): ScheduleGate page created, page ID `143613032`, public URL `https://www.linkedin.com/company/schedulegate/`. Filled: logo (linkedin/logo-512.png), banner/cover (linkedin/cover-1584x396.png), tagline "Schedule quality checks at the speed of git push.", Industry=Software Development, size 2–10, Privately held, About (Overview) from `linkedin/page-copy.md`, founded 2026, 10 specialties, buttons Message=On + custom CTA "Visit website" → https://schedulegate.dev. **Gotcha: the page editor rejects Saves with "Another admin is trying to make changes" / HTTP 412 version-tag mismatch (`Update organization ... version tag X mismatches current version tag Y`) whenever a prior save from the same session bumped the page `.gitignore`-free version tag. Fix: fully leave the edit page (dashboard), reload fresh so the session's version tag matches, then redo the edit — cover-apply and Details saves both needed this after earlier failed attempts. Banner upload input is a hidden `<input id="org-admin-background-image-single-file-input">`; use `setInputFiles`, then the crop dialog's Apply. **First Page post PUBLISHED 29 Aug 2026** (approved v2 copy from kit §3, attached `terminal-demo-block.png`), share URN `urn:li:share:7499565231501680641`, URL `https://www.linkedin.com/feed/update/urn:li:share:7499565231501680641` (also visible at `company/schedulegate/posts`). Composer gotchas: pasting the site URL auto-creates a link card (title+og:description ASCII dump) — remove it via "Remove media" to restore the toolbar, then add the real image via "Add media" (file chooser) and confirm the crop dialog with Next; **declined** the "Share this post on your profile" dialog (Not now) to avoid touching the personal profile. Next: day-1 Reddit 5 comment-only threads + 3 DMs (recipient names needed), then day-3 r/projectmanagement post.

Reddit Ask #2 (29 Aug 2026, as `gil_sched`, brand-new 0-karma account) — **FINAL RESULT: 2 of 5 live, both in r/primavera.** The 2 r/projectmanagement comments were auto-removed by that sub's new-account/low-karma AutoMod and **dropped by user decision** (modmail/karmer-boost declined). Ask #4 (day-3 r/projectmanagement post) will hit the SAME gate — plan for karma/age or modmail there.
**Account facts (via `reddit.com/api/me.json`):** created 29 Aug 2026 20:41 UTC; `has_verified_email: true` (email verified ~21:34 UTC through the Reddit password-creation flow — the AutoMod "you still need a verified email" line goes stale once done); `comment_karma: 0`, `link_karma: 1`. **Lesson: r/projectmanagement's AutoMod gate is account age + comment karma, NOT email — reposts stay removed even with the email verified** (canary `p6olv6z`, verified-repost `p6omufs` both removed ~21:37).
- ✅ r/primavera "How are you reviewing changes across multiple P6 schedule updates?" (`1vpk1ou`) → comment `p6odze2` LIVE; initially posted identity-mapping/float-trend essay, then **edited** to a shorter human rewrite (per Gil: original read too AI). **Competitor reply received (21:17 UTC): `alex-sam2kb` pitched scheduletracker.app in reply** — the thread is being read by the market. **21:49 UTC replied as `p6op10e`** (approved): flag-not-force, lineage carries the trend, re-baseline marker trick; zero links — stays the neutral expert vs two toolmakers (OP + StrategicPM). Verified live via anonymous old.reddit permalink.
- ❌ r/projectmanagement "critical path vs critical chain in practice" (`1vumnfz`) → comment `p6oe56o` AUTO-REMOVED → DROPPED. Text was the rewritten one-person/near-critical-task bottle example (Gil flagged the first draft "didn't make sense").
- ✅ r/primavera "Need advise — building schedule comparison tool" (`1v69zv2`) → comment `p6ogfam` LIVE (~21:04 UTC): pair-diff is the commodity; win on activity-matching across updates (IDs change / WBS rework / splits) + change significance (float remaining, distance from CP); offline. **1st submit silently dropped** (collapsed composer) → retried via expanded composer + keyboard typing.
- ❌ r/projectmanagement "project plan never matches reality" (`1v3lm25`, manufacturing) → comment `p6ogppc` AUTO-REMOVED → DROPPED. Text: what-changed-since-last-update pass before status; let the plan read ERP for gate dates; "statusing gap, not planning gap".
r/ProjectControls supplier integration (`1oyss2j`) **skipped** — 10-month-old thread, 2 comments, OP clearly interview-prepping; a fresh account commenting reads as necro-spam and hurts the persona.
**Reddit composer gotcha (new shreddit UI):** the collapsed `faceplate-textarea-input` must be clicked first to expand the lexical RTE (`[data-lexical-editor="true"]`, only then present). `.fill()` on the RTE (a) replaces the entire box and (b) collapses `\n\n` into one giant paragraph — use `page.keyboard.type()` per paragraph + `keyboard.press('Enter')` between paragraphs instead.
**Reddit visibility check — AUTHORITATIVE ≠ profile/composer DOM.** Auto-removed comments STILL render on the user profile and in-thread DOM to the author. The only real check is an **anonymous fetch of the `old.reddit.com` comment permalink** (`https://old.reddit.com/r/<SUB>/comments/<THREAD>/-/<COMMENT>/`) — live comments render their body; removed ones say "there doesn't seem to be anything here". AutoMod removal notices live in the /notifications page (`notification-announcement` rows; click-through to `/notifications/a/<id>` shows full message). Account status JSON: `fetch('https://www.reddit.com/api/me.json', {credentials:'include'})` → `has_verified_email`, `comment_karma`, `created_utc`.

Ask log (29 Aug): Ask #2 done 2/5 (r/primavera live; r/projectmanagement dropped). Ask #3 waiting on names. Ask #4 blocked (karma). Ask #5 r/golang main post removed — no repost; Small Projects `p6ozigb` live.

I3 Reddit comments (31 Aug 2026, as `gil_sched`) — **FINAL: 1 of 2 live.** I2 (LinkedIn) was **CANCELLED by user** after confirming LinkedIn has no web path to comment as a Company Page on personal posts (both target authors = individuals); 2 final drafts were approved but never posted (on file in the execution kit for future use).
- ✅ **r/Construction "sloppy work" (`1vxbl5d`)** → comment LIVE as `gil_sched`: "schedule-modeling issue" (float/logic angle; existing comments only covered cost/back-charge). Verified in gil_sched profile + thread. Old.reddit UI; clicked the `form.usertext button[type=submit]` save → posted fine.
- ❌ **r/primavera "Identify Logic Ties Between Other Projects" (`1vgdlke`)** → comment ("still worth knowing how P6 flags these… dangling flag, not a real dependency") NEVER went live: 3× `POST /api/comment` all returned **HTTP 200** but the text is **NOT in gil_sched's comment history nor the thread** → held by Reddit spam filter (quick successive comments on a brand-new account). **STOPPED after 3 attempts to avoid duplicate risk. Do NOT retry/repost** — if the filter ever clears, multiple 200s could surface as dupes. Lesson: a 200 from `/api/comment` is NOT proof of a visible comment; verify via the anonymous old.reddit permalink / profile history. r/primavera may also be stricter with new accounts (its AutoMod ate the account's early comments too in Ask #2? — actually Ask #2's r/primavera comments were live; this may be subwide filtering, treat r/primavera as warm-to-filter).
- Note: the fully-read `1vgdlke` thread already covered Reports Wizard + Relationships-column + TASKPRED SQL (atticus2132000) and the AI-agent suggestion (Dishy22); my angle (don't trust AI blindly + data-date nuance) was additive, but the sub's filter blocked it regardless.

## Week 2 daily plan (31 Aug–6 Sep 2026)

Written 30 Aug: `Local_Files_do_not_track_do_not_commit/week-2-daily-plan.md` (gitignored).
Mon = finish weekend leftovers (LI comments, Reddit comments, 20 names, 5 DMs). Tue = export how-to or r/sideproject. **Wed = Medium — DONE (published Tue 1 Sep, pulled forward; see "Medium intro" below).** Thu = HN only if 4h free, else commandline/rooms/Missed Tasks. Fri = rooms. Sat = presence. Sun = measure + fork. Independent bank if a day slips. Approvals still gate every publish.

## RESUME HERE — go-live marketing (next actions)

### S2 LinkedIn Page comments (30 Aug 2026, afternoon) — 1/8 done

**Mechanism confirmed for commenting as the ScheduleGate Page:**
- LinkedIn **Pages cannot comment on a personal/member profile's posts** from the profile-activity view — no identity switcher appears (tested on Qudrat Ullah's profile; comment box was bound to Gilberto's personal profile only). **Do not attempt S2 on personal-profile posts.**
- On **company (Page) posts** the identity switcher IS available: each post has an "Open menu for switching identity when interacting with this post" button next to the reaction bar (shows current actor avatar). Dialog: "Comment, react, and repost as" → two radios (personal profile / ScheduleGate) → Save selection. After selecting ScheduleGate, the comment box becomes **"Comment as ScheduleGate…"** and comments post under the ScheduleGate brand (verified; the posted comment renders as "ScheduleGate's comment" with Like/Reply).
- **Net: S2 must target company-Page posts in the scheduling/DCMA space**, not personal posts. ScheduleLens (a competitor) is a valid target and was used for comment 1 — angle: useful, no link, no pitch. Guardrail met (no bashing, contrast only).

**Comment 1/8 (posted 30 Aug, live):** ScheduleLens DCMA 14-point company post, Angle #2:
> "On the high-float point — more than 44 working days of total float is usually missing successors, not 'build in some slack.' When I pull those activities up they typically have no downstream logic, so nothing is pulling them forward."

**Comments 2-4/8 (posted 30 Aug, live, all as ScheduleGate Page, no link):**
- **2) Planned Limited** "stale programme / controls gap" post → float-trend tell + weekly cycle (Angle #1/3).
- **3) P6ProjectPlanning** "most expensive activity is the one you forgot" post → present-but-unlinked, large float, nothing pulls forward (Angle #1/#2).
- **4) STRATUM COREX** "DCMA 14-Point Dashboard" post (direct competitor, neutral/additive) → auditable+offline calculation basis (Angles #4+#8).
- **5) P6ProjectPlanning** "What does a Project Planner actually do?" post → defend/forecast via what-changed discipline (Angle #3).
- **6) ScheduleLens** "version comparison / EOT evidence" post (different from comment 1's DCMA post) → what-changed vs reported, critical-path re-route (Angle #3).
- **7) Milestone** "Hidden Risk of Programme File Conversion" post → treat delivered file as source of truth, check the export not the tool (Angle #7).
- **8) STRATUM COREX** "Executive Dashboard" post (2nd on this competitor page) → forecast to the delta, float burn + progress as one story (Angle #3).

**S2 COMPLETE 8/8 (30 Aug 2026).** All live as ScheduleGate Page, no link/pitch. Pages used: ScheduleLens (x2), Planned Limited, P6ProjectPlanning (x2), STRATUM COREX (x2, competitor, neutral), Milestone. Non-competitors: Planned, P6PP, Milestone.

**Sourcing note (30 Aug):** Page-posts in scheduling space are scarce. Personal-profile posts are NOT commentable by a Page. Many consultancy pages (plan-controls-limited, planning-solutions-nw-ltd, planner-time-limited, advance-consultancy-limited, project-planning-and-scheduling) have NO organic posts. Reliable source of slugs: "Pages people also viewed" rails. Confirmed-having-posts so far: ScheduleLens, Planned Limited, P6ProjectPlanning (posts 1+3), STRATUM COREX (3 posts), Plan Ahead (1 hiring post). Remember to stop at 8, useful/2-4 sentences/no link/no pitch each.

### S1 harvest (30 Aug 2026, ~00:28–00:32 UTC) — logged to weekend-sprint.md posting log + kit §8

- **LinkedIn post 1** (3h old): **0 reactions, 0 comments** — nothing to reply to or DM. Post visible fine.
- **r/primavera `1vpk1ou`**: both comments live (`p6odze2` 4h, `p6op10e` 3h). **Live 2-way exchange w/ competitor alex-sam2kb (scheduletracker.app)** — our last message unanswered 3h; ball in their court. Check again before Sat end.
- **r/primavera `1v69zv2`**: `p6ogfam` live 3h, OP has NOT replied.
- **r/golang Small Projects `p6ozigb`**: live 2h, full body + both links render, no replies.
- **GitHub**: 1 star, 0 forks; no issues/discussions. **Traffic: 60 views/3 unique; 405 clones/76 unique cloners.** PR #48 (awesome-project-management) open, `mergeable_state: blocked` = no conflicts, awaiting maintainer review.

**Reading:** harvest was EMPTY of new activity → the "stay in threads / skip new posts" rule does NOT trigger. S2 (8 LI comments) and S3 (5 Reddit comments) remain the Saturday moves. LinkedIn 0/0 means the Page's engagement is zero — comments as the Page are exactly the fix.

### Status at 30 Aug 2026: Week-1 organic started. Weekend sprint written — run from `Local_Files_do_not_track_do_not_commit/weekend-sprint.md`. Approvals still gate every publish.

**Do not redo:** LinkedIn page + post 1, r/primavera comments, r/golang promo (spent), r/projectmanagement (blocked), GitHub topics, screenshots, Railway F1.

**This weekend, in order:**
1. ⚡ HARVEST DONE (see S1 above). Next: 8 LinkedIn Page comments (S2) + 5 Reddit comments in allowed subs (S3).
2. Ask #3 — 5 DMs. Need names. List file: `outreach-list.md`.
3. Sunday only if Sat is quiet: LinkedIn export how-to post, r/sideproject Show, Planning Planet / MPUG / PMSE help-first, awesome-list PRs if OK'd.
4. 90-minute fallback: harvest + 8 LI comments + 5 DMs. Skip every new Show post.

Ask #4 (r/projectmanagement Show) and Show HN / PH / ads stay blocked or week-2+.

**Next action (resume from 2 Sep):** W2 Wed Medium distribution is **COMPLETE** — LinkedIn Company Page share (posted as ScheduleGate page, `terminal-demo-block.png` + alt text, declined share-on-profile): `urn:li:share:7501080324739887105`; own first comment as Author posted (Metric 9 / High Float, no promo): `happy-to-be-argued-with-on-the-invalid-dates-metric-pam-calls-a-task-invalid-when-its-finish-is-12548356b029`; GitHub README one-liner added ("## Further reading"); leftover debug draft `da21defd98f1` deleted; kit §8 W2 + memory logged. Remaining W2: Thu 3 Sep surface/HN pick, Fri rooms, Sat presence, Sun measure + fork.

## Friday 4 Sep 2026 — harvest + I12 (final) + log

**Harvest — nothing landed anywhere.** X: X1 (30 views, 3 self-replies), X2 "Missed Tasks" (2 views), **X3 "High Float" posted live 3 Sep** (`https://x.com/schedulegate/status/2096029246040793337`); searches empty (+ bot follower xclaire1x already ignored). **LinkedIn Company Page (143613032): 3 posts — launch 29 Aug (56 imp), Medium share (9 imp), Metric 11 "Missed Tasks" post 9/3/2026 (1 imp, was unlogged → now in kit §8); 0 reactions/comments on all; 0 notifications.** GitHub: 1 star / 0 forks / 0 issues; traffic 134 views/7 unique; clones 551/122 unique; PR #48 (awesome-project-management) still open. Medium: 1 response (our own Author comment); 46 followers / 67 following.

**Reddit verification method CHANGED (walls went up):** anonymous permalink fetch (the old authoritative check — line 35) is now dead. old.reddit permalinks → login wall; new.reddit → "Prove your humanity" CAPTCHA; `api.reddit.com` + `m.reddit.com` → "You've been blocked by network security" (403); bare curl → 403. **New authoritative public check = the thread's RSS feed via curl** (browser UA + `Accept: application/atom+xml,application/rss+xml`): `curl -sL <UA> -H <Accept> "https://www.reddit.com/r/<SUB>/comments/<THREAD>/.rss"`. If `gil_sched` is absent from a freshly-updated feed as `author`, the comment is filter-held even though the logged-in author view + `api/me.json` show it. Feed `updated` timestamp confirms freshness.

**I12 (karma-only, zero-product) — BOTH attempts filter-held.** (1) r/primavera 1w4ynqt "One schedule update is only a snapshot": `p7vut5i` posted, RSS-verified held (all 10 other comments present, none ours), not retried. (2) Fallback r/Construction 1w6h8sw "Truck driver to project management?": Gil-approved 1154-char answer to the OP's asked lateral move (dispatch/scheduling) → **first submit got a "Take a break for 2 minutes" throttle** (fresh-account rate limit) → waited ~2 min, re-submitted, still throttled (49s) → waited 60s, final submit **succeeded** (comment `t1_p7vwjaw` visible in author DOM) but **RSS-verified HELD** (fresh feed 00:31:51Z, all 28 other comments present, zero `gil_sched`). No retry. `gil_sched` still 0 comment karma; held comments may auto-surface later as the account ages. **Lesson: fresh accounts get BOTH a submit throttle AND sub filter holds; comment content is irrelevant — hold is identity/karma-based. Do not interpret "logged-in DOM shows it" as public.** Thread 1w6h8sw permalink for later recheck: `.../comments/1w6h8sw/.../p7vwjaw/`.

**C7:** PMSE DONE Thu (Marcus Dale, Q 35106, manual signup in Gil's browser). Planning Planet + MPUG still pending manual email-verify + reCAPTCHA (needs Gil's browser session) → I6/I7/I8 room answers still pending.

**Logged:** kit §8 (W1 row traffic—134/7, 551/122; W2 row incl. I12 held ×2 + Metric 11 LinkedIn post + PMSE answer; X table → 3 originals + X3; I12 + harvest status bullets), week-2 daily plan Fri 4 Sep row.

**DECISION (4 Sep, Gil): AGE THE ACCOUNT.** Stop Reddit commenting entirely while the fresh-account flags cool. The two held comments (p7vut5i, p7vwjaw) stay in place server-side and may auto-surface as `gil_sched` matures. Revisit I12 no earlier than W3; do NOT modmail the subs for now.

**Next:** Sat 5 Sep presence (harvest 15, I2 3 Page comments, I3 2 Reddit comments — **note: I3 does not apply while Reddit is on hold**, substitute with what's on the Saturday plan, I14 X replies only — 45 min).

## X / Twitter campaign (drafted 1 Sep 2026)

Playbook: `Local_Files_do_not_track_do_not_commit/x-twitter-campaign.md` (gitignored). Wired into `go-live-marketing-plan.md` (channel priority 4), `marketing-execution-kit.md`, `week-2-daily-plan.md` (I13 profile / I14 replies / I15 launch; Thu option E).

**Not posted.** Brand account `@schedulegate` (fallback `_cli` / `_dev`), never personal — same rule as LinkedIn Page. Two jobs: intro thread (X1) + lead engagement (search → no-link replies → warm DM cap 3/day). Do not first-touch X with the Medium URL. Do not ship X1 the same morning as LinkedIn Medium share or Show HN.

Waiting on Gil: (1) handle + bio, (2) X1 thread Go/edit/wait, (3) X2–X8 batch, (4) standing-yes on I14 eight-angle replies. Do not create the account until (1). Do not tweet until (2).

## Medium intro (30 Aug 2026) — **PUBLISHED 1 Sep 2026**

Drafts in `Local_Files_do_not_track_do_not_commit/` (gitignored): `medium-article-dcma-schedule-quality.md` + `medium-posting-plan.md`. Gate: Gil approval. Links allowed: site, docs, GitHub, releases. **No Gumroad** in the story. Reddit still no Medium URL unless asked.

**Live:** https://medium.com/@giljunqueira/most-project-schedules-would-fail-a-defense-audit-455744c4294f · published **Tue 1 Sep 2026** (self-published, pulled forward from Wed). Five topics set: Project Management, Software Development, DevOps, Programming, Construction. **W2 Wed distribution DONE 2 Sep 2026:** LinkedIn Company Page share (`terminal-demo-block.png`) → `urn:li:share:7501080324739887105`; own first comment as Author (Metric 9 / High Float) → `happy-to-be-argued-with-on-the-invalid-dates-metric-pam-calls-a-task-invalid-when-its-finish-is-12548356b029`; GitHub README one-liner; §8 W2 logged; leftover debug draft `da21defd98f1` deleted. **Optional remaining:** add the display/cover image (`terminal-demo-block.png`) retroactively — it was NOT set at publish. **Medium gotcha (own first comment):** `dialog` responses editor is a `textbox` ("What are your thoughts?") — `.fill()` collapses `\n\n` into a single paragraph; the case reads fine either way. Comment posts via the dialog's "Respond" button (enabled once text entered), then shows as **Author** with a standalone comment permalink.

## Harvest re-verify (30 Aug 2026, ~13:45 local) — logged to kit §8 / weekend-sprint §3

- **Awesome-go PR #6633 MERGED** (gh-verified: state MERGED, mergedAt 2026-08-30T00:34:59Z). ScheduleGate is live in awesome-go README.md line 3727 `Software Packages > Other Software`. The `needs-maturity`/`needs-coverage` labels did NOT block. This was an unnoticed win → logged.
- **PR #48 (awesome-project-management) still OPEN** — no nagging.
- **Re-verified live threads (read-only):** r/primavera `1vpk1ou` — both `p6odze2`/`p6op10e` live; substantive 2-way exchange with competitor alex-sam2kb; last word is ours (`p6op10e`, re-baseline marker), ball in their court. r/primavera `1v69zv2` — `p6ogfam` live 21h, OP Friendly-Battle-6558 has NOT replied. r/golang Small Projects `1vxc255` — `p6ozigb` live 19h, no replies.
- **GitHub traffic (fresh):** 61 views / 4 unique; 489 clones / 91 unique; 1 star / 0 forks / 0 issues. **Site health:** schedulegate.dev, /docs, /terms, /#pricing all 200.
- **F1 confirmed already done** — the license-server email-body change (support@ + Terms/Privacy/Refund in key email) was committed 27 Aug (acf31d7) and deployed 29 Aug. Railway envs live on the service; no action needed.

## Zip contract

`scripts/release/zip-binaries.sh` requires: `schedulegate`, `schedulegate.exe`, `schedulegate-linux`, root `LICENSE`, root `LICENSE-COMMERCIAL`, `docs/user-manual.html`. Fails if any missing. No MIT placeholder.

## MARKETING RESET — v2 (5 Sep 2026) · READ THIS BEFORE ANY MARKETING WORK

**Weeks 1–2 result: $0, 0 leads, 0 partner conversations.** Measured output: LinkedIn Page 3 posts / 66 total impressions / 0 engagement; X 3 originals / 36 views / 1 human follower; Reddit `gil_sched` 0 karma, AutoMod-blocked, 2 comments filter-held (account unusable); Medium 1 article / 0 organic responses; GitHub 1 star / 7 unique visitors; **website has zero analytics installed**; **20 outreach names researched, 0 messages ever sent.**

**Root cause:** every channel was launched cold and anonymous (new Page, new Reddit account, new brand X handle, 2 pen names) — audience-of-zero, not a copy problem. The plan was ~80% guardrails / 20% action and let the highest-ROI item (outbound DMs) slip 6 days in a row. "DCMA 14-point" is a low-hundreds-of-searches/month niche — it cannot be won by broadcasting.

**v2 strategy: direct + service-led.** Top of funnel is now a **free DCMA 14-point audit** — Gil runs the assessment on their sanitised export and emails back the HTML report in 1 business day. This bypasses the two real blockers (can't install unsigned binaries on locked-down laptops; Community tier can't produce the HTML report). Ladder: free audit → $99 Pro → $500–750 paid audit → $1,200–2,500 compare/delay audit → $500–1,500/mo retainer.

**Locked decisions (Gil, 5 Sep):**
1. **Personal LinkedIn = DMs/connection requests only. Posts stay on the Company Page.** The Page never does outreach.
2. **Yes to done-for-you audits, free then paid.**
3. **NO product changes** — free tier stays 1 run/month terminal-only; no GUI ship; no backlink in generated HTML reports. (Flagged to revisit at week 8 if "I can't install that" dominates.)
4. **Yes to cookieless site analytics** (Vercel Analytics/Plausible) on schedulegate.dev only. **CLI stays 100% telemetry-free** — that remains the selling point.

**The one rule: 10 LinkedIn connection requests/day, every working day, as Gil.** Was 0/day for 14 days; that single fact explains the entire launch result.

**Channel verdicts:** START = 1:1 outreach (primary), free audits, paid audits, partnerships, multi-page SEO, site analytics. KEEP-but-demoted = Company Page (1–2 posts/wk, unmeasured), Medium (1/mo), Planning Planet/MPUG/PMSE **under Gil's real name**. PAUSE = Reddit (30-day cooldown, aging only), X (5 min/wk lead-scan, no originals). HOLD = Show HN (until a case study exists). **DROP = Product Hunt** (wrong audience). Ads only ever Google exact-match, later.

**Retire the pen names** `Maximiano2006` (Planning Planet) and `Marcus Dale` (PM Stack Exchange) — they cannot accumulate reputation, be followed, or be attributed. Don't delete the answers, just stop adding.

**Funnel fixes F1–F10** (website/README only, no product): analytics, robots.txt+sitemap.xml, JSON-LD, `/dcma-14-point` cornerstone, 14 per-metric pages (1/wk), `/audit` landing page + intake form (first-ever email capture), README first-screen rebuild (currently no badges, no screenshot), GitHub description/topics/social image, downloadable `examples/sample-schedule.xlsx`, ICP-B meta description.

**Metrics changed:** impressions/views deleted from the scorecard. Track requests sent → accepts → replies → **audit conversations → audits delivered** → quotes → revenue. Leading indicator = requests sent.

**Warmest existing contact:** ziv madar (`@zivon55` on X, USACE thread on Planning Planet comment 98888) — helped twice with no pitch. Connect on LinkedIn as Gil in week 3 and offer the audit.

**Hard line carried forward:** ITAR/CUI/classified → refuse the file, offer offline screen-share instead. Delete every client file within 7 days and say so in writing. Never show one client's schedule to anyone, ever.

**Active files** (`Local_Files_do_not_track_do_not_commit/`, gitignored): `go-live-marketing-plan.md` (v2, strategy), `week-3-4-plan.md` (14-day todo, 7–20 Sep), `audit-offer-playbook.md` (offer/delivery/pricing/objections), `outreach-list.md` (engine + 100-name target list + tracker), `partnerships.md` (5 partner types + copy + tracker), `marketing-execution-kit.md` (copy + historic log).
**Archived — do not execute:** `week-2-daily-plan.md`, `weekend-sprint.md`. **Paused:** `x-twitter-campaign.md`.

**Week 3–4 targets (7–20 Sep): 100 connection requests, 3 audits delivered, 2 quotable sentences, 10 partner approaches.** Not sales.
