# Cloudflare Registrar API Integration Plan

> **For Hermes:** Use subagent-driven-development to implement this plan task-by-task.
> **Execution model routing:** This slice is planned with **GLM 5.2** and executed with **Qwen 3.8** via opencode. When dispatching implementation work, drive opencode with the Qwen 3.8 model; confirm the exact model id against the opencode provider config at dispatch time, and pin it per session.

**Goal:** Let indietool users **buy, renew, and manage** domains through the Cloudflare Registrar API, turning Cloudflare from a read-only portfolio source into an actionable registrar.

**Architecture:** Wrap the Cloudflare Registrar API in two layers inside the existing `providers.CloudflareProvider`:
1. **Stable Registrar management API** (already partially wired via `cloudflare-go/v4` SDK v4.5.1 — `Registrar.Domains.List/Get/Update`) for list/get, auto-renew, privacy/lock, and renewal pricing.
2. **New Registrar *purchase* API (beta)** — `domain-search`, `domain-check`, `registrations`, `registration-status` — which **are not present in the vendored SDK v4.5.1**, so they go in as a thin `net/http` client (`registrar_purchase.go`) using the same config token/account, kept separate so adding SDK support later is trivial.

New CLI surface is gated behind a `domains.Purchaser` capability interface that only Cloudflare implements; other registrars are untouched (DRY/YAGNI, no breakage).

**Tech Stack:** Go 1.24, cobra, `cloudflare-go/v4` v4.5.1 (stable mgmt), `net/http` + `encoding/json` + `tidwall/gjson` (beta purchase), existing `output`/`domains` packages.

---

## Ground truth (verified from docs + code)

**Beta Registrar API (purchase) — https://developers.cloudflare.com/registrar/registrar-api/** (docs file `/tmp/cf-registrar-api.mdx`):
- `GET /accounts/{account_id}/registrar/domain-search?q=<kw>&limit=<n>` — cached discovery; `result.domains[] = {name, registrable, tier, pricing:{currency, registration_cost, renewal_cost}}`.
- `POST /accounts/{account_id}/registrar/domain-check` body `{"domains":["x.com",...]}` (<=20/req) — real-time availability + price; `registrable:false` carries `reason` (`domain_unavailable`, `extension_not_supported_via_api`, `extension_disallows_registration`, ...).
- `POST /accounts/{account_id}/registrar/registrations` body `{"domain_name":"x.dev"}` (+ optional `contacts` override) -> **201 Created** (completed) or **202 Accepted** (in progress). Billable, non-refundable. Defaults: `auto_renew:false`, `privacy_mode:redaction` (if TLD supports), default payment method charged.
- `GET /accounts/{account_id}/registrar/registrations/{domain}/registration-status` — poll states: `in_progress`, `succeeded`, `failed`, `action_required`, `blocked`. **Stop on `action_required`/`failed`; surface the action.**
- `GET /accounts/{account_id}/registrar/registrations/{domain}` — get registration resource.
- `Prefer: respond-async` header forces async (return 202 immediately).
- **Beta limitations:** renewals X, transfers X, contact updates X via API; only a subset of TLDs.

**Stable Registrar management API (SDK v4.5.1, verified in `registrar/domain.go`):**
- `DomainService.List / Get / Update` typed. `DomainUpdateParams` supports exactly three mutations: `auto_renew`, `locked`, `privacy`. Auto-renew toggle, lock, WHOIS privacy all via SDK.
- `Domain` typed struct lacks `renewal_price`, `auto_renew`, `privacy`, `registry_expires_at` — but these ARE in the raw JSON (`RawJSON()` / `gjson`), as the current `parseDomain` proves (`auto_renew` read via gjson). So renewal price and privacy can be parsed from `RawJSON()`.

**Current code deficits (confirmed in tree):**
- `providers/cloudflare.go`: `GetDomain`, `UpdateAutoRenewal`, `GetRenewalInfo`, `GetNameservers`, `UpdateNameservers` are all **TODO stubs**. Only `ListDomains` works.
- `config add provider cloudflare` never collects `account_id` -> `ListDomains` passes empty `AccountID` and the registrar calls can't work. Must add `--account-id`.
- `domains.Registrar` interface has no purchase methods -> add capability interface.

---

## Prerequisite checks (user must satisfy; surfaced by `domain check`/`register`)

1. Cloudflare **account ID** (now required in config).
2. API token with **Registrar write** permission (dashboard -> API tokens).
3. **Default payment method** on the account.
4. **Default registrant contact** configured + **Domain Registration Agreement** accepted (dashboard -> domains/registrations).

`domain register` must fail fast with a clear message pointing at whichever prerequisite is unmet (surface `action_required`/blocked states).

---

## Proposed CLI surface

```
indietool domain check <name>...            # beta: real-time availability + price (<=20)
indietool domain register <name>            # beta: billable purchase; confirm price; poll
indietool domains renew <name> [--on|--off] # stable: show renewal cost; toggle auto-renew
indietool domain get <name>                 # stable: full details (expiry, auto-renew, privacy, lock, contact, renewal price)
indietool domain set <name> --auto-renew/--privacy/--locked [--on|--off]   # stable SDK
indietool domain transfer-out <name> --on|--off   # stable raw HTTP (optional stretch)
```

`... --json` and `--dry-run` honored per S0.2 conventions. Register always requires interactive `--yes`/confirm showing price.

---

## Task breakdown

### Task 1: Create `docs/plans/` dir + commit scaffolding
**Files:** create only. Objective: land an empty tracking commit in the worktree.
- `mkdir -p docs/plans`, add `.gitkeep`, commit `chore: add docs/plans directory`.

### Task 2: Add `account_id` to Cloudflare config
**Files:**
- Modify `cmd/indietool/cmd/config_add_provider_cloudflare.go`
- Modify `indietool/config.go` (validation: `account_id` required when registrar ops used)
- Test: `cmd/indietool/cmd/root_test.go` (add a case)

**Step 1:** failing test — `config add provider cloudflare --api-token X --account-id acc` persists `account_id`; without `--account-id` returns validation error.
**Step 2:** implement `--account-id` flag -> set `CloudflareConfig.AccountId`.
**Step 3:** pass + commit `feat(config): require account_id for cloudflare registrar`.

### Task 3: Implement `GetDomain` + `GetRenewalInfo` (stable read)
**Files:**
- Modify `providers/cloudflare.go` (`GetDomain`, `GetRenewalInfo`)
- Test: `providers/cloudflare_test.go` (new; use recorded/replay or a thin fake client)

**Step 1:** failing test asserting `GetDomain("example.dev")` returns expiry/auto_renew/privacy/lock parsed from `RawJSON()`; `GetRenewalInfo` returns `renewal_price` from `RawJSON()`.
**Step 2:** `GetDomain` -> `client.Registrar.Domains.Get(ctx, name, registrar.DomainGetParams{AccountID: cloudflare.F(acct)})`; parse via gjson into `ManagedDomain` (reuse/extend `parseDomain`). `GetRenewalInfo` -> read `renewal_price`+currency from `RawJSON()` into `DomainCost` (Currency "USD", RenewalPrice, TransferPrice 0 unless present).
**Step 3:** pass + commit `feat(providers/cf): implement get-domain + renewal info`.

### Task 4: Implement `UpdateAutoRenewal` (stable write)
**Files:** modify `providers/cloudflare.go`; test in same file.
**Step 1:** failing test — `UpdateAutoRenewal(ctx,"x.dev",true)` issues `DomainUpdateParams{AutoRenew:true}` (assert via fake).
**Step 2:** `client.Registrar.Domains.Update(ctx, name, registrar.DomainUpdateParams{AccountID: F(acct), AutoRenew: F(enabled)})`.
**Step 3:** pass + commit `feat(providers/cf): implement auto-renew toggle`.

### Task 5: Add `domains.Purchaser` capability interface
**Files:** modify `domains/manage.go` (or new `domains/purchase.go`).
**Step 1:** define:
```go
type Availability struct {
    Name string             `json:"name"`
    Registrable bool        `json:"registrable"`
    Tier string             `json:"tier,omitempty"`
    Reason string           `json:"reason,omitempty"`
    Currency string         `json:"currency,omitempty"`
    RegistrationCost float64 `json:"registration_cost,omitempty"`
    RenewalCost float64     `json:"renewal_cost,omitempty"`
}
type RegistrationResult struct {
    DomainName string   `json:"domain_name"`
    State string        `json:"state"` // in_progress|succeeded|failed|action_required|blocked
    Completed bool      `json:"completed"`
    Error *string       `json:"error,omitempty"`
}
type Purchaser interface {
    Check(ctx context.Context, names []string) ([]Availability, error)
    Register(ctx context.Context, name string) (*RegistrationResult, error)
    RegistrationStatus(ctx context.Context, name string) (*RegistrationResult, error)
}
```
**Step 2:** helper `AsPurchaser()` on a registrar (mirror existing `AsRegistrar()` pattern; type-assert in cmd layer).
**Step 3:** unit test interface compiles; commit `feat(domains): define purchaser capability interface`.

### Task 6: Implement CF purchase client (beta, raw HTTP)
**Files:**
- Create `providers/registrar_purchase.go`
- Test: `providers/registrar_purchase_test.go` (httptest)

**Step 1:** failing tests for `Check` (POST body/parse incl. `reason`), `Register` (POST, 201 vs 202 handling), `RegistrationStatus` (poll parse).
**Step 2:** thin client: base `https://api.cloudflare.com/client/v4`, `Authorization: Bearer ***`; parse `errors[]` code/message, 20-domain chunking for `Check`, honor `Prefer: respond-async` toggle. Field mapping via struct tags; parse `pricing.registration_cost` / `renewal_cost`.
**Step 3:** pass + commit `feat(providers/cf): beta registrar purchase client`.

### Task 7: `domain check` command (beta)
**Files:** create `cmd/indietool/cmd/domain_check.go`; modify `cmd/indietool/cmd/domain.go`.
**Step 1:** failing command test (cobra run) for availability output incl. `--json`.
**Step 2:** command: require `cloudflare` configured; type-assert `Purchaser`; call `Check`; render via `output.Table` (reuse domain table config where possible) / JSON; print `reason` row for non-registrable.
**Step 3:** pass + commit `feat(cmd): domain check (availability + price)`.

### Task 8: `domain register` command (beta, billable)
**Files:** create `cmd/indietool/cmd/domain_register.go`.
**Step 1:** failing test — without `--yes`, prints price + asks confirm; with `--yes` calls `Register` then polls `RegistrationStatus` to terminal state, stopping on `action_required`/`failed` with actionable message.
**Step 2:** implement: `Check` first -> show price -> confirm (unless `--yes`) -> `Register` -> poll (e.g. 3s interval, max ~60s) -> terminal state handling; fail fast on missing prereqs; `--json` output.
**Step 3:** pass + commit `feat(cmd): domain register (beta, guarded)`.

### Task 9: `domains renew` command (stable auto-renew + price)
**Files:** create `cmd/indietool/cmd/domains_renew.go`.
**Step 1:** failing test — no flag prints renewal price + auto-renew status; `--on`/`--off` calls `UpdateAutoRenewal`.
**Step 2:** implement, surface beta limitation note: API renewal is auto-renew only; a manual early renew is dashboard-only.
**Step 3:** pass + commit `feat(cmd): domains renew (auto-renew + price)`.

### Task 10: `domain get` + `domain set` commands (stable)
**Files:** create `cmd/indietool/cmd/domain_get.go`, `cmd/indietool/cmd/domain_set.go`.
**Step 1:** failing tests — `get` renders full details; `set --auto-renew --on` calls Update; `set --privacy`/`--locked` map to `DomainUpdateParams`.
**Step 2:** implement (map flags -> `DomainUpdateParams` fields).
**Step 3:** pass + commit `feat(cmd): domain get + set (auto-renew/privacy/lock)`.

### Task 11: (Stretch) `domain transfer-out` toggle
**Files:** create `cmd/indietool/cmd/domain_transfer_out.go`; add raw-HTTP method in `registrar_purchase.go`.
**Step 1/2/3:** TDD `POST /registrar/domains/{name}/transfer/out` and `.../transfer/out?enable=true` (enable/disable transfer-away). If unstable during impl, mark explicitly deferred. Commit `feat(cmd): domain transfer-out toggle`.

### Task 12: README + ROADMAP updates
**Files:** modify `README.md`, and `ROADMAP.md` at `/home/projects/indietool/ROADMAP.md` (note: outside this repo).
**Step 1:** README: add `domain check/register/renew/set/get` + Cloudflare prereqs (account id, registrar-write token, billing, default contact, DRA).
**Step 2:** ROADMAP: flip the open decision "Ship includes paid register?" to **API register (CF beta)**; add a note to the "In-CLI registrar checkout / payments" non-goal that CF Registrar API registration is now in (still no generic payments platform); tick new checklist line. Add a changelog entry (section 17).
**Step 3:** commit `docs: document cloudflare registrar commands + roadmap update`.

### Task 13: Full verification pass
**Step 1:** `cd /home/projects/indietool/cli.cf-registrar && go build ./... && go vet ./...`.
**Step 2:** `go test ./...` (expect: all new tests pass; the known pre-existing failures tracked in the skill are not regressions).
**Step 3:** manual smoke with a throwaway CF token if available (binaries at `/tmp/indietool-test` per skill pitfall #8); otherwise mark as pending live test.
**Step 4:** commit any final fixes `chore: verification pass`.

---

## Files likely to change
- `providers/cloudflare.go` (implement 5 TODO stubs + `AsPurchaser`)
- `providers/registrar_purchase.go` **(new)** — beta client
- `providers/registrar_purchase_test.go` **(new)**
- `providers/cloudflare_test.go` **(new)**
- `domains/manage.go` or `domains/purchase.go` **(new interface)**
- `cmd/indietool/cmd/domain_check.go`, `domain_register.go`, `domain_get.go`, `domain_set.go` **(new)**
- `cmd/indietool/cmd/domains_renew.go` **(new)**
- `cmd/indietool/cmd/domain_transfer_out.go` **(new, stretch)**
- `cmd/indietool/cmd/config_add_provider_cloudflare.go` (add `--account-id`)
- `indietool/config.go` (validation)
- `README.md`; `/home/projects/indietool/ROADMAP.md` (outside repo; update separately)

## Tests / validation
- Unit tests with `httptest.Server` + a fake SDK client (see provider tests pattern); no live tokens in git.
- Command tests via cobra `Execute` (see `root_test.go`).
- Full: `go build ./... && go vet ./... && go test ./...`.

## Risks / tradeoffs / open questions
- **Beta API churn:** `domain-register`/`check` depend on a documented-beta surface; keep it behind an explicit capability and a `--yes` guard. The stable mgmt half (auto-renew, get, set) is safe regardless.
- **Billable + non-refundable:** `domain register` must hard-require price display + confirm. No silent CI registration.
- **`auto_renew` defaults to false** on beta registration -> set it `true` by default after successful registration (or prompt) so users don't accumulate soon-to-expire domains.
- **Renewal semantics:** the API only does auto-renew; an immediate pre-expiry payment is dashboard-only. `domains renew` should say so.
- **Nameservers:** Cloudflare domains must use CF nameservers; `UpdateNameservers` on the registrar is intentionally limited — implement `GetNameservers` via the Zones API; leave `UpdateNameservers` out of scope for the registrar (already a DNS provider concern).
- **`account_id` requirement** changes the CF onboarding (config add now needs `--account-id`); acceptable since registrar ops are account-scoped. Consider auto-detecting via `GET /user/tokens/verify` in a later pass.
- **SDK vs raw HTTP:** kept separate so a future `cloudflare-go` release with beta support can replace `registrar_purchase.go` with SDK calls without touching commands (single seam).
- **PR strategy:** this branch is `feat/cloudflare-registrar` off `main`; the s0.2 (`--json`) branch is ahead of origin/main — confirm merge ordering with Ruiwen before opening the PR.
- **Model routing:** plan authored/refined with GLM 5.2; implementation dispatch via opencode pinned to Qwen 3.8.

## Execution handoff
Plan complete and saved. Ready to execute using subagent-driven-development — dispatch a fresh subagent per task through opencode (Qwen 3.8), with two-stage review (spec compliance then code quality). Confirm model id via `opencode models` before first dispatch. Shall I proceed?

---

## Post-plan addendum (2026-08-23): migrated management commands to the new Registrar API

During pre-merge review we found that the legacy registrar domain-management
endpoints used by Tasks 3, 4, 9 and 10 (`GET/PUT /accounts/{id}/registrar/domains[...]`,
the `cloudflare-go` v4 SDK `Registrar.Domains.List/Get/Update`) are **deprecated
by Cloudflare with end-of-life 2026-09-27**. The plan's "stable mgmt half" was
therefore not stable.

Migration applied on this branch:

| Command | Old (deprecated) | New |
| --- | --- | --- |
| `domains` list / `ListDomains` | `GET /registrar/domains` (SDK) | `GET /registrar/registrations` (cursor pagination, per_page=50) |
| `domain get` | `GET /registrar/domains/{name}` | `GET /registrar/registrations/{name}` |
| auto-renew toggle (`domains renew --on/--off`, `domain set --auto-renew`) | `PUT /registrar/domains/{name}` | `PATCH /registrar/registrations/{name}` body `{"auto_renew": bool}` |
| privacy/lock (`domain set --privacy/--locked`) | `PUT /registrar/domains/{name}` | **No API equivalent yet** — fails fast with a clear error; dashboard-only |

Response-shape consequences of the new registration schema
(`domain_name, status, created_at, expires_at, auto_renew, locked,
privacy_mode` — no `name`, no `name_servers`, no `renewal_price`):

- Parsing keys off `domain_name`; nameservers and renewal price are dropped
  gracefully (empty / nil), never fabricated.
- `privacy_mode: "redaction"` maps to privacy on; other values map to off.
- Renewal pricing is reported as unavailable via a sentinel error
  (`providers.ErrRenewalPricingUnavailable`) and surfaced by `domains renew`
  as an explicit note instead of a fabricated price.

Open items: Cloudflare may extend PATCH beyond `auto_renew` later; when it
does, re-enable `--privacy/--locked`. The async update workflow returns a
WorkflowStatus; terminal failures are surfaced, pending/in_progress are
treated as accepted. Live validation against the real (billable) API or the
official Registrar Sandbox is still pending.


## Addendum 2026-08-23: Registrar Sandbox toggle

- Added provider-level "sandbox" config (CLI: indietool config add provider cloudflare --sandbox; yaml: sandbox: true).
- When enabled, the registrar purchase/management client routes all requests to /accounts/{id}/registrar-sandbox/... (same host, identical workflow semantics; no billing, purchases persist).
- Scope: registrar endpoints only; the DNS/zones client is unaffected.
- Sandbox supports com/net only and requires full contact data on register (no Express Mode).
- Use the sandbox for pre-merge live verification instead of a local mock.
