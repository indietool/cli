# Streaming secret ingest (CLI) + native Hermes secret source — Implementation Plan

**Date:** 2026-09-10 · **Branch:** `feat/secret-stdin-exec` (worktree `/home/projects/indietool/cli.stdin-exec`, cut from main `116b2d1`) · **Executor:** Hermes (this session, direct execution per standing split)

**Goal:** `secret set --stdin` (raw-byte, trim-safe ingest) + `secret exec` (S1.4: inject a database's secrets as env vars for a child process), then spike the Hermes `command` secret source against the new path; land the recipe config-only if it holds, else a real plugin.

## Verified repo state (2026-09-10, main @ 116b2d1)

- `cmd/indietool/cmd/secrets_set.go`: `cobra.ExactArgs(2)`, value = `args[1]` untrimmed, `--note`/`--expires` flags, age-ssh fallback via `resolveKeyBackend(secretsConfig, keyringErr)` + retry — reuse that block unchanged.
- `Manager.SetSecret/GetSecret/ListSecrets/ExportSecrets` (`indietool/secrets/manager.go`); storage = Badger per-database under `Config.GetSecretsDir()`, age-ssh backends; `resolveKeyBackend()` prompts for an SSH pubkey on keyring failure.
- `secret get NAME@db -S` prints the raw value via `fmt.Print` (no trailing newline) → byte-exact `cmp` roundtrip works.
- `secret list --json` still prints the human table (known dogfood bug) → `exec` must use the Manager in-process, never parse list output.
- Pre-existing failure on main: `TestSecretsDirectoryCalculation` (indietool pkg) — NOT ours, per acceptance.
- Hermes (in-container, `/opt/hermes` read-only): builtins bitwarden/onepassword/**command** in `agent/secret_sources/registry.py`; contract in `base.py` — subclass `SecretSource`, implement `fetch(cfg, home_path) -> FetchResult`, config section `secrets: <name>:`, register via `PluginContext.register_secret_source()`. `CommandSource` (shape **bulk**): helper run once via `/bin/sh -c`, key travels as data in `HERMES_SECRET_KEY`, default 3s hard timeout, 1 MiB output cap, output parsed as KEY=VALUE dotenv map (one line per var; matching surrounding quotes stripped; line whitespace stripped). `fetch()` only RETURNS the map (registry applies) → safe to exercise without touching `os.environ`. Live Hermes config = `/opt/data/config.yaml` (we run inside the gateway container).
- `jq` NOT on this host — spike parser must be python3 (present).
- Bridge scripts under review: `scripts/sync_indietool_to_hermes_env.py` (+ migrate twin) in repo; `/opt/data/bin/hermes-secrets-{sync,migrate}` wrappers on host.

## Decisions

| # | Decision | Choice |
|---|----------|--------|
| D1 | `--stdin` semantics | Raw bytes from `cmd.InOrStdin()`; default **no trimming**; `--trim` = `strings.TrimSpace` |
| D2 | arg collision | `--stdin` + positional value → error; neither → error naming `--stdin`; `Args: cobra.RangeArgs(1,2)` |
| D3 | empty stdin | error (refuse to store an empty secret) |
| D4 | exec db selection | `secret exec [@db] -- <cmd> [args...]` via `ArgsLenAtDash()`; `@db` optional, else config default |
| D5 | collision policy | refuse (list offenders) if any secret name already in parent env, unless `--force` |
| D6 | invalid env names | warn + skip (never abort the whole exec) |
| D7 | exec output discipline | human mode: injected names → **stderr** (child owns stdout); `--json`: report object printed AFTER child exits |
| D8 | exit code | propagate child's exit code (signal → 128+sig) via an `exitError` handled in `Execute()` — no `os.Exit` inside RunE, keeps tests alive |
| D9 | Feature 2 | spike FIRST on a **throwaway** db (never real `@hermes`); if the `command` recipe roundtrips → config-only landing: document recipe, delete in-repo bridge scripts; plugin only if the recipe breaks |
| D10 | host cutover | after merge: enable `secrets: command:` in `/opt/data/config.yaml`, delete `/opt/data/bin` wrappers, live-verify, update skill |

## Tasks

### Task 1 — Test infra
`cmd/indietool/cmd/secrets_test_helpers_test.go`: helper writing an isolated config yaml (temp `storage_dir`, `key_backend: age-ssh`, throwaway ed25519 keypair via `ssh-keygen -t ed25519 -N "" -f …`), returning config path. No keyring, no `@hermes`.

### Task 2 — `secret set --stdin` / `--trim` + tests
Modify `secrets_set.go` (RangeArgs, flags, value resolution before `ParseSecretIdentifier`); tests in `secrets_set_test.go`: raw roundtrip (newline + spaces preserved), `--trim`, both-args error, missing-value error, empty-stdin error, legacy positional unchanged. Drive via `rootCmd.SetArgs/SetIn` + `Execute()`; assert stored bytes via Manager.

### Task 3 — `secret exec` + tests
New `cmd/indietool/cmd/secrets_exec.go` (+ minimal `Execute()` change in `root.go` for `exitError`); tests in `secrets_exec_test.go`: injection visible to child, parent-env collision refused then `--force` wins, invalid env name skipped, `--json` report, exit-code propagation.

### Task 4 — Spike: Hermes `command` source over the new path
Throwaway dir `/tmp/indietool-f2-spike/` (keypair + yaml + 3 secrets set via the NEW `--stdin` binary; values include ` #`, quotes, plain). Helper command: `<bin> --config <yaml> secret export @f2test --json | python3 toenv.py`. Import `CommandSource` straight from `/opt/hermes`, call `fetch()` (returns a map; registry applies — safe), assert resolved == source of truth (print names/lengths only, never values). Measure wall time vs the 3s default cap. Expected: exact roundtrip for newline-free values; multiline values = documented native-plugin-only limitation.

### Task 5 — Feature 2 landing (per spike outcome)
Config-only: recipe + security model into skill `references/hermes-dogfood.md`; delete `scripts/sync_indietool_to_hermes_env.py` + `migrate_hermes_env_to_indietool.py` from the repo (host wrapper deletion happens at cutover, Task 6).

### Task 6 — Acceptance gates + PR
- `echo -n "$VAL" | indietool secret set F2T@db --stdin` → `secret get -S` → `cmp` byte-exact (throwaway config; `-n` honored).
- `indietool secret exec -- sh -c 'env'` shows injected vars; count via grep ≥ 2; no value in argv (by construction; verified by printing argv inside the child).
- `go build ./... && go vet ./...` clean; `go test ./cmd/... ./indietool/secrets/` green (pre-existing `TestSecretsDirectoryCalculation` excluded per acceptance).
- Commit per task; push branch; open PR to `main` (repo convention: squash-merge).

## Outcome (2026-09-10)

- F1 shipped on this branch: `secret set --stdin/--trim` (bf0099a) + `secret exec` (bb408a1).
  All acceptance gates verified live: byte-exact `echo -n` roundtrip, newline preserved by
  default, values never in argv, injection count via env grep, exit-code + signal propagation,
  `--json` report, collision refusal / `--force`.
- F2 spike verdict: **config-only landing** — Hermes' real `CommandSource.fetch()` roundtrips
  indietool exports exactly (incl. `#`/quote-containing values), 0.20s vs the 3s cap. No plugin.
  Recipe recorded in the indietool-cli skill (`references/hermes-dogfood.md`); emitter added at
  `scripts/hermes-secret-source/toenv.py`.
- The legacy .env bridge scripts were never tracked by this repo — they live in the outer
  project dir (`/home/projects/indietool/scripts/`) with hardcoded-path wrappers in
  `/opt/data/bin/`. Decommissioning them (plus enabling `secrets: command:` in Hermes' config)
  is host-side cutover, pending owner approval; until then no dual maintenance exists in-repo.
