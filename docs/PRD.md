# jevai — Product Requirements (v1)

**One line:** A self-hosted platform that puts a **calibrated typed judgment on every item in a
stream** — powered by TypeSafe's **Jev** model, served as clean HTMX web apps from one Go engine
on our own Mac.

_Repo: `Everything-with-jev-ai`. Owner: Kurnia Massidik. Status: dev scaffold in place._
_Canonical research: `researchanddevelopment/typesafe-ai/` (esp. `08-products.md`, `00-what-jev-is.md`)._

---

## 1. The unlock (why this exists)

A judgment used to cost ~1–5¢ and 3–10s (an LLM call). With Jev it costs **~$0.00004 and ~300ms**,
and comes back **typed + calibrated** (it tells you when it doesn't know). That makes one thing
possible that wasn't:

> **Judge *every* item — every row, listing, ad, claim — not a sample.**

That is the only defensible reason to build on Jev instead of an LLM, and every surface here is an
instance of it.

## 2. What we are NOT building

- **Not a chatbot.** Jev can't generate text; it returns typed decisions (Choice / Score / Noul).
- **Not `=JEV()` (Sheets add-in) first.** It's a great product but forces a JavaScript add-in
  (Apps Script) — parked until the web surfaces prove the engine. See §7.
- **Not WhatsApp triage.** Removed — needs a WhatsApp Business API (WABA) account, an external
  platform gate. (`typesafe-ai/08-products.md` #2.)

## 3. Architecture — one engine, many surfaces

```
Users ── Cloudflare tunnel ── Mac ── jevai engine (Go) ─┬─ web surfaces (HTMX + Tailwind)
                                                        ├─ judge/ : question-packs (Choice/Score/Noul)
                                                        ├─ jev/   : thin client → Jev API (one POST)
                                                        └─ ledger : per-user token metering (SQLite)
                                                             │
                                                        Jev API (TypeSafe, US) — one endpoint, one key
```

- **One Go engine** holds the Jev key (from `.env`, never per-user), applies **question-packs** to
  items, and **meters usage** per user. Same "one gateway, one key, meter it ourselves" shape as MAAS.
- **Surfaces are thin HTMX web apps** rendered by the engine (templ + Tailwind). No SPA, no JS build.
- **Deploys on the Mac** behind the Cloudflare tunnel, exactly like MAAS (new subdomain per surface).

## 4. Tech stack (decided)

| Layer | Choice |
|---|---|
| Backend | **Go** (stdlib `net/http` routing) |
| Templating | **templ** (type-safe Go HTML) |
| Frontend | **HTMX** (server-rendered fragments) — no Node/JS toolchain |
| Styling | **Tailwind** (standalone CLI binary) |
| Icons | inline **SVG** |
| Jev client | thin `net/http` client (Jev API is one POST — no SDK) |
| Data / ledger | **SQLite** (`modernc.org/sqlite`, pure Go) |
| Dev env | **Nix flake** dev shell |
| Runtime | **Docker** (static binary → distroless) |
| Look | matches **typesafe.ai** — Host Grotesk + Fragment Mono, paper/ink, hot-pink accent, retro window chrome |

## 5. The brain — Jev, platform-provided

- **One aisurplus-style key**: the TypeSafe key lives in the engine `.env`, read server-side.
  **Users never enter a key.** (Absent key = scaffold mode; `/health` still works.)
- Jev call: `POST https://api.typesafe.ai/v1/systemone`, `state` + typed `questions` → typed answers
  + `usage.total`. The engine reads tokens from each response to meter per user (§6).
- **Design rules baked in (from `00-what-jev-is.md` jagged edges):** ask narrow atomic questions;
  do all math/date/count comparisons **in code**; one item + only the needed fields per request
  (context rot); confidence-gate every action (auto-pass / review / "no idea").

## 6. Metering & billing

- The **ledger** records `(user, surface, tokensIn, tokensOut, judgments, at)` from each Jev
  response — we attribute per user because one key can't be attributed by TypeSafe.
- **Trial → gate at payment**, metered in total tokens (placeholder allowance, TBD §9).
- Margins are ~95% at Jev pricing; the gate is the money moment, after value is felt.

## 7. Surfaces (the products)

All are "items in → ranked/typed judgments out," rendered with HTMX. Numbers kept from the research
so cross-refs resolve.

| # | Surface | Input → output | Moat | JS? |
|---|---|---|---|---|
| **3** | **Listing hygiene** | CSV of SKUs → ranked fix list | marketplace rules | none |
| **4** | **Ad pre-flight** | creative + landing URL → risk report | policy + alignment | none |
| **5** | **Claim screening** | copy + rule-pack → flagged claims | regulatory rules (strongest) | none |
| 1 | `=JEV()` (deferred) | spreadsheet cell → judgment | distribution | needs JS add-in |

**First surface (recommendation):** pick one of #3/#4/#5 — all are pure Go + HTMX, zero JS, and each
is a "sell the output first" v0 (run it by hand for 2–3 buyers before polishing the app). My lean:
**listing hygiene (#3)** — clearest input (CSV), local wedge, easiest to demo. _Open decision §9._

## 8. Scaffold status (what's built)

- Repo, Nix flake, Dockerfile (multi-stage → distroless), docker-compose, justfile, `.air.toml`.
- Go engine: graceful `main.go`, `server` (routes: `GET /`, `GET /health`, `POST /demo/judge`),
  thin `jev` client, `judge` packs stub, `ledger` (in-memory; swap to SQLite next).
- Web: templ layout + landing themed to typesafe.ai, HTMX "try a judgment" demo, Tailwind tokens.
- **Next:** finish the landing UI, verify `just build` + Docker, wire a real surface + SQLite ledger.

## 9. Open decisions (need your call)

1. **First surface** — listing hygiene (#3, my rec), ad pre-flight (#4), or claim screening (#5)?
2. **Payment rails** — global cards (Stripe) vs local (Midtrans)? (Decides self-serve vs local.)
3. **Trial allowance + plan prices** — placeholders until set.
4. **Deploy target** — same Mac + Cloudflare tunnel as MAAS (new subdomain), confirm.

## 10. Build order

1. ✅ Dev scaffold (Go · templ · HTMX · Tailwind · Nix · Docker) that builds & serves the themed landing.
2. Wire the **SQLite ledger** + the **live Jev client** behind `/demo/judge` (needs `TYPESAFE_API_KEY`).
3. Build **surface #1** (per §9.1): input form → question-pack → ranked HTMX result, metered.
4. Deploy on the Mac behind the tunnel; run it by hand for 2–3 buyers (sell the output).
5. Add the trial meter + payment gate once §9.2–9.3 are decided.
6. Later: `=JEV()` Sheets add-in as a thin JS shell over the same engine.

## 11. Non-goals (v1)

- No chatbot / text generation. No `=JEV()` yet. No WhatsApp/WABA. No other AI providers (Jev only).
- No multi-region HA — single Mac + tunnel is fine for v0/pilots (move the API to cloud only if a
  global self-serve surface demands it).
