# Porkbun Registrar Support (check / register / auto-renew / get / set)

**Date:** 2026-09-02
**Branch:** `feat/porkbun-registrar` (worktree `/home/projects/indietool/cli.porkbun-reg`)
**Status:** In progress

## Goal

Parity with the Cloudflare Registrar slice (PR #2) for Porkbun: real-time
availability checks, domain registration, auto-renewal management, and
single-domain detail — plus a CLI `--provider` route on `domain check` /
`domain register` so both purchase-capable registrars are reachable.

## API ground truth (Porkbun API v3.18, verified from the official OpenAPI
spec at https://porkbun.com/api/json/v3/spec + llms-full.txt)

- **Base URL:** `https://api.porkbun.com/api/json/v3` (docs curl examples use
  `api.porkbun.com`, NOT `porkbun.com`).
- **Auth:** header-based — `X-API-Key` / `X-Secret-API-Key` (body auth
  `apikey`/`secretapikey` also accepted; headers used here).
- **Sandbox:** same base URL; sandbox keys are prefixed `pk1_sb_` / `sk1_sb_`
  (create at porkbun.com/account/api, or anonymously via
  `POST /apikey/request {"sandbox": true}` — 20/IP/hour, seeded with $1000
  fake credit, `POST /sandbox/topup` to refill). No path-prefix routing
  needed (unlike Cloudflare's `/registrar-sandbox/` mirror).
- **`POST /domain/checkDomain/{domain}`** -> `{status, response: {avail:
  "yes"|"no", price, regularPrice, firstYearPromo, premium, minDuration,
  additional: {renewal: {price}, transfer: {price}}}, limits, ttlRemaining}`.
  One domain per request. Default rate limit **1 check / 10s / account**
  (configurable per key); 429 carries `Retry-After` and the response carries
  `ttlRemaining` (seconds until the window resets).
- **`POST /domain/create/{domain}`** — synchronous registration. Body:
  `{cost (integer pennies, MUST equal current price at min duration),
  agreeToTerms: "yes", whoisPrivacy?: bool, dryRun?: bool}`. Response 200:
  `{status: "SUCCESS", domain, cost, orderId, balance, limits, requestId}`.
  `dryRun: true` returns a `DryRunPreviewResponse` (200, `wouldSucceed`)
  without creating an order or consuming rate limit. Constraints: account
  email+phone verified, sufficient credit, **at least one prior registration
  on the account**, **premium domains cannot be registered via API**.
  Registrations use the **account's default registrant contact** — the API
  takes NO contact fields. Always registry-minimum duration (usually 1yr).
- **`POST /domain/updateAutoRenew/{domain}`** — body `{status: "on"|"off",
  domains?: []}`. This endpoint EXISTS in v3.18 — the provider's old
  "not supported by Porkbun API" stub was written against the older API and
  is now wrong.
- **`GET /domain/get/{domain}`** -> `{domain: {domain, status, tld,
  createDate, expireDate, securityLock (0/1), whoisPrivacy (0/1), autoRenew
  (0/1), apiAccess (0/1), notLocal (0/1)}}`; 404 when not found.
- **`POST /domain/listAll`** — rich list (start-offset pagination, 1000/page)
  with the same per-domain fields; not used here (SDK list still works).
- **Errors:** envelope `{status: "ERROR", message, code,
  next_action?: {type, hint, url}}`. 429 -> `RATE_LIMIT_EXCEEDED` with
  `Retry-After` header.
- Not wired (documented non-goals for this slice): `/domain/renew` (early
  renewal purchase), `/domain/transfer`, contacts (`/domain/getContacts`,
  `/domain/updateContacts`), glue records, URL forwarding, registration
  requirements.

## Design

- **Thin `net/http` client** `providers/porkbun_registrar.go` (mirrors
  `registrar_purchase.go`): the vendored `tuzzmaniandevil/porkbun-go v1.0.2`
  only covers list/NS/URL-forwarding and uses the older body-auth + old
  paths. New client keeps the SDK untouched; base URL overridable via
  `INDIETOOL_PORKBUN_API_BASE` for mock servers.
- **`Purchaser` capability on `PorkbunProvider`:**
  - `Check`: sequential `checkDomain` per name (1/request). Adaptive pacing:
    sleep `ttlRemaining` (+1s grace, capped 15s) between calls so the default
    1/10s limit doesn't 429 mid-batch; on 429 wait `Retry-After` (<=60s) once
    and retry. `Registrable = avail=="yes" && premium!="yes"` (premium is
    refused because the API cannot register premium names). Currency USD.
  - `Register`: fail fast when `contact != nil` (Porkbun registers with the
    account contact). Re-checks the domain internally to obtain the exact
    cost in pennies (price must match at creation time; re-checking is safer
    than trusting the earlier CLI check), refuses premium/unavailable, then
    `POST /domain/create` with `agreeToTerms: "yes"`. Returns terminal
    `succeeded` on 200 (Porkbun registration is synchronous — no polling).
  - `RegistrationStatus`: `GET /domain/get/{domain}` — found = succeeded,
    404 = in_progress (only reachable if a future async path appears).
- **Registrar fixes in `providers/porkbun.go`:**
  - `UpdateAutoRenewal`: real implementation via `updateAutoRenew` (replaces
    the stale "not supported" stub) — `domains renew x --on/--off` and
    `domain set x --auto-renew --on/--off` work for Porkbun.
  - `GetDomain`: single `GET /domain/get/{domain}` (replaces list-all+NS
    fan-out); maps locked/privacy/autoRenew; `domains renew` info display and
    `findRegistrarForDomain` get faster + richer.
  - `Check`/`Register`/`RegistrationStatus` added; DNS/NS/pricing stay on the
    existing SDK client (working, body-auth, don't touch).
- **CLI routing:** `domain check` / `domain register` gain `--provider
  (cloudflare|porkbun)`; empty = auto-detect the first enabled
  Purchaser-capable provider in registry order. `getCloudflarePurchaser` is
  generalized to `getPurchaser`. Register rejects `--contact-*` with
  `--provider porkbun` up front (account-contact model). `domains renew`,
  `domain get`, `domain set` already resolve the registrar generically.
- **Sandbox:** no code path needed — users point the porkbun provider at
  `pk1_sb_`/`sk1_sb_` keys. Documented in `config add provider porkbun` help.

## Tests

- `providers/porkbun_registrar_test.go`: httptest mock — auth headers,
  check available/unavailable/premium/ERROR/429-retry, register
  success/cost-flow/contact-refusal/ERROR, auto-renew on/off payloads,
  get 200/404 mapping.
- `cmd/indietool/cmd/domain_check_test.go` / `domain_register_test.go`:
  porkbun provider via env-overridden base URL + `--provider porkbun`,
  auto-detect when only porkbun configured, contact refusal for porkbun.
- Live sandbox E2E (manual): anonymous
  `POST /apikey/request {"sandbox": true}` -> checkDomain -> create (register)
  -> get -> updateAutoRenew against the real API with throwaway sandbox keys.

## Non-goals

- Early renewal purchase (`/domain/renew`) and transfer — future slice.
  `domains renew` remains the auto-renew toggle (CF parity).
- Contact management, glue, URL forwarding, webhooks, hosting.
- Replacing the vendored SDK for DNS/list/NS/pricing.
