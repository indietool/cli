# `indietool`

> **The fast builder's toolkit — less managing, more making**

Tired of bouncing between registrars, tracking domain renewals in spreadsheets, and copy-pasting secrets into `.env` files?

🎯 `indietool` is the fast builder's toolkit — less managing, more making. It helps you

- 🌍 Hunt domain names across 50+ TLDs — in seconds
- 🗓️ Track expiries across registrars like Cloudflare & Porkbun
- ☁️ Manage DNS records across providers with auto-detection
- 🔐 Securely store API keys & secrets — OS keyring or SSH-key encrypted

No dashboards. No vendor lock-in. Just you and your terminal.

## 🎬 See it in action

![indietool demo](docs/demo.gif)

> *Demo: `domain explore` -> `domains list` -> `dns set` -> `secret get` in under 30 seconds*

---

## 🚀 Quick Start

### Installation

```bash
# Homebrew (recommended)
brew install indietool/tap/indietool

# Go
go install github.com/indietool/cli@latest

# Binary releases (macOS, Linux, Windows)
# Download from https://github.com/indietool/cli/releases

# Shell completions
indietool completion bash > /usr/local/share/bash-completion/completions/indietool
indietool completion zsh > /usr/local/share/zsh/site-functions/_indietool
indietool completion fish > ~/.config/fish/completions/indietool.fish
```

### Try it in 30 seconds

```bash
# Check domain availability (no API keys needed!)
indietool domain explore myapp

# Save a test API key (auto-creates encryption key)
indietool secret set stripe-key "sk_test_..." --note "Stripe test key"

# Manage DNS records with automatic provider detection
indietool dns list example.com
```

---

## 🔁 Everyday Developer Flows

### Weekend Project Setup

```bash
# Check which domains are available
indietool domain explore myproject --tlds dev,com,ai

# Set up DNS records for your new domain
indietool dns set myproject.dev @ A 192.168.1.100
indietool dns set myproject.dev www CNAME myproject.dev
indietool dns set myproject.dev api A 192.168.1.101

# Store your API keys securely (auto-creates encryption key)
indietool secret set openai-key "sk-..." --note "OpenAI API key"
indietool secret set stripe-key "sk_test_..." --note "Stripe test key"

# Organize secrets by project using custom databases
indietool secret set api-key@myproject "key123" --note "Project-specific key"
```

### Production Deployment

```bash
# Check domain expiry before renewal
indietool domains list --provider cloudflare

# Verify DNS configuration before going live
indietool dns list myproject.com --wide

# Clean up old DNS records
indietool dns delete myproject.com old-api A

# Update production DNS records
indietool dns set myproject.com @ A 203.0.113.10
indietool dns set myproject.com www CNAME myproject.com

# Export secrets for deployment
export OPENAI_KEY=$(indietool secret get openai-key -S)
```

---

## 💡 Features

---

### 🔍 Find Available Domains Instantly

**Problem:** Manually checking domain names is slow and painful.
**Solution:** `indietool domain explore` checks 50+ TLDs in seconds.

```bash
indietool domain explore awesomeproject
```

```
DOMAIN                     STATUS     TLD         EXPIRY
awesomeproject.ai          Available  ai          -
awesomeproject.dev         Available  dev         -
awesomeproject.com         Taken      com         2026-07-06
...
50 domains checked: 45 available, 5 taken
```

Filter by specific TLDs:

```bash
indietool domain explore awesomeproject --tlds ai,dev,io,sh
```

Or pass a TLD list from file:

```bash
indietool domain explore myproject --tlds @tldfile
```

---

### 🔎 Direct Domain Lookup

Know the exact domain you're targeting?

```bash
indietool domain search awesomeproject.io
```

---

### 📊 Track All Your Domains in One Place

**Problem:** Domains expire. You don't want surprises.
**Solution:** View all domains across registrars in one simple table.

First, connect your registrar(s):

```bash
# Cloudflare
indietool config add provider cloudflare \
  --account-id YOUR_ACCOUNT_ID \
  --api-token YOUR_TOKEN \
  --email your@email.com

# Porkbun
indietool config add provider porkbun \
  --api-key YOUR_KEY \
  --api-secret YOUR_SECRET

# The Little Host (DNS only)
indietool config add provider thelittlehost --api-key tlh_YOUR_API_KEY
```

Then list your domains:

```bash
indietool domains list
```

```
NAME                PROVIDER    STATUS   EXPIRES  AUTO-RENEW  AGE
myawesomeapp.com    cloudflare  healthy  8mo      Yes         2y
sideproject.ai      cloudflare  healthy  1y       Yes         1y
```

Need more info?

```bash
indietool domains list --wide
```

```
NAME                PROVIDER    STATUS   EXPIRES  AUTO-RENEW  AGE   NAMESERVERS                          COST  UPDATED
myawesomeapp.com    cloudflare  healthy  8mo      Yes         2y    fred.ns.cloudflare.com,pam.ns.cl...  N/A   2y
sideproject.ai      cloudflare  healthy  1y       Yes         1y    fred.ns.cloudflare.com,pam.ns.cl...  N/A   1y
```

---

### 🛒 Buy & Manage Domains with Cloudflare Registrar (beta)

**Problem:** Buying a domain means leaving the terminal, and renewal settings rot.
**Solution:** `indietool` wraps the new Cloudflare Registrar API — real-time availability checks, guarded registration, and auto-renew management.

Prerequisites on your Cloudflare account:

1. **Account ID** — now required when configuring the provider (`--account-id`)
2. API token with **Registrar write** permission
3. A **default payment method** on the account
4. A **default registrant contact** and the **Domain Registration Agreement** accepted (dashboard → Domains/Registrations)

```bash
# Configure Cloudflare with your account ID
indietool config add provider cloudflare \
  --account-id YOUR_ACCOUNT_ID \
  --api-token YOUR_TOKEN

# Real-time availability + price (beta API, up to 20 domains)
indietool domain check myapp.dev
indietool domain check myapp.com myapp.dev myapp.app --json

# Register a domain (billable — shows the price and asks you to confirm)
indietool domain register myapp.dev
indietool domain register myapp.dev --yes       # skip the confirmation prompt
indietool domain register myapp.dev --dry-run   # price check only, no purchase

# Renewal info + auto-renew toggle (new Registrar API)
indietool domains renew myapp.dev
indietool domains renew myapp.dev --on
indietool domains renew myapp.dev --off

# Full details: expiry, status, auto-renew, lock, privacy mode
indietool domain get myapp.dev

# Toggle auto-renew
indietool domain set myapp.dev --auto-renew --on
# (privacy/lock are dashboard-only; see notes below)
```

Notes:

- All commands in this section use the **new Cloudflare Registrar API** (`/registrar/domain-check`, `/registrar/registrations`, ...). The legacy `/registrar/domains` endpoints are deprecated by Cloudflare (end-of-life 2026-09-27) and are **not** used.
- Only a subset of TLDs is supported by the API beta; renewals, transfers, and contact updates are not yet available through it.
- Registration is **billable and non-refundable** — `indietool` always shows the price and requires confirmation unless `--yes` is given.
- API registrations default to `auto_renew: off`; enable it with `indietool domains renew <domain> --on`.
- Renewal pricing is not exposed by the API, and manual early renewal (paying to extend before expiry) is dashboard-only; the API manages auto-renewal only.
- **Privacy & registrar lock are dashboard-only**: the API currently supports updating `auto_renew` only, so `domain set --privacy/--locked` fails fast with an error instead of silently doing nothing.

---

### ☁️ Manage DNS Records Across Providers

**Problem:** Managing DNS records across different providers is tedious and error-prone.
**Solution:** `indietool dns` automatically detects your DNS provider and lets you list and update records from the command line.

#### List DNS records

```bash
# Auto-detect provider and list records
indietool dns list example.com
```

```
DNS Provider: cloudflare
TYPE  NAME     CONTENT
A     @        192.168.1.1
A     www      192.168.1.2
CNAME api      example.com
MX    @        10 mail.example.com
```

_Note: Cloudflare proxied records are indicated with a cloud icon, available only with the Cloudflare provider, for domains hosted on Cloudflare_

#### Get detailed view

```bash
indietool dns list example.com --wide
```

```
TYPE  NAME     CONTENT          TTL   PRIORITY  ID
A     @        192.168.1.1      300             abc123
A     www      192.168.1.2      300             def456
CNAME api      example.com      300             ghi789
MX    @        mail.example.com 300   10        jkl012
```

#### Set DNS records

```bash
# Add an A record
indietool dns set example.com www A 192.168.1.100

# Add MX record with priority
indietool dns set example.com @ MX "10 mail.example.com" --priority 10

# Add TXT record for domain verification
indietool dns set example.com @ TXT "v=spf1 include:_spf.google.com ~all"
```

#### Delete DNS records

```bash
# Delete specific record by name and type
indietool dns delete example.com www A

# Delete all records for a name (with confirmation)
indietool dns delete example.com api

# Delete specific record by ID (when multiple records have same name)
indietool dns delete example.com test --id abc123

# Delete without confirmation
indietool dns delete example.com www A --force

# Delete root domain record
indietool dns delete example.com @ MX

# Combine filters for precision
indietool dns delete example.com api --type CNAME --id def456
```

#### Specify provider explicitly

```bash
# Use specific provider instead of auto-detection
indietool dns list example.com --provider cloudflare
indietool dns set example.com api A 192.168.1.50 --provider porkbun
indietool dns delete example.com old-record A --provider namecheap
```

#### Supported DNS providers

- **Cloudflare** - Full CRUD operations with proxy status indicators
- **Porkbun** - Complete DNS record management (list, set, delete)
- **Namecheap** - Full CRUD support with batch operations
- **The Little Host** - Full DNS record management

#### Auto-detection

`indietool` automatically detects your DNS provider by checking nameservers:

- No need to specify `--provider` in most cases
- Seamlessly works across different providers
- Falls back to manual provider selection if needed

---

### 🔐 Secure Local Secrets Without the Hassle

**Problem:** Secrets are either insecure or annoying to manage.
**Solution:** `indietool secrets` encrypts secrets using your OS keyring — no cloud, no sync, no complicated setup to manage.

#### How it works

| Component             | Backend   | Stored At                                        | Encrypted |
| --------------------- | --------- | ------------------------------------------------ | --------- |
| Secrets Database      | both      | `~/.config/indietool/secrets/`                   | ✅        |
| Encryption Key        | keyring   | OS Keychain / gnome-keyring                      | ✅        |
| Encryption Key        | age-ssh   | `~/.config/indietool/keys/db-key-<database>.age` | ✅        |

`indietool` supports two backends for storing the database encryption key:

- **keyring** (default) — uses your OS keyring. Works well for desktop sessions.
- **age-ssh** (recommended for servers / SSH sessions) — encrypts the key with your SSH public key and stores it as a file. Decryption uses your SSH private key or agent.

#### Choose a backend

```bash
# Explicit initialization with age-ssh (recommended for remote hosts)
indietool secrets init --backend age-ssh

# Specify a custom SSH key pair
indietool secrets init --backend age-ssh \
  --ssh-public-key ~/.ssh/id_rsa.pub \
  --ssh-private-key ~/.ssh/id_rsa

# Explicit initialization using the OS keyring
indietool secrets init --backend keyring
```

If `indietool` detects that the keyring is unavailable (e.g. in an SSH session), it will guide you through selecting an SSH key automatically on first use.

#### Store a secret (auto-initializes encryption)

**No setup required!** The first time you store a secret, `indietool` automatically creates an encryption key.

```bash
indietool secret set stripe-key "sk_test_..." --note "Stripe test key"
Auto-generated encryption key for database 'default'
Secret 'stripe-key' stored successfully
```

#### Organize secrets with custom databases

Use the `key@database` format to organize secrets by project or environment:

```bash
# Store in custom databases
indietool secret set api-key@production "prod_key_123"
indietool secret set api-key@staging "staging_key_456"
indietool secret set db-password@myproject "secret123"

# Retrieve from specific database
indietool secret get api-key@production -S
```

#### Retrieve a secret

```bash
# Safe output (masked)
indietool secret get stripe-key

# Show actual value (use -S or --show)
indietool secret get stripe-key -S
```

#### List all secrets

```bash
# List secrets in default database
indietool secret list

# List secrets in specific database
indietool secret list @production
indietool secret list @staging
```

#### Manage databases

List all your secret databases:

```bash
indietool secrets db list
```

```
Available secrets databases:
  default (default)
  production
  staging
  myproject
```

Delete a database and all its secrets:

```bash
# Interactive confirmation
indietool secrets db delete staging

# Force delete without confirmation
indietool secrets db delete staging --force
```

#### Use in scripts

```bash
# Inject secret into command
indietool secret exec stripe-key -- curl -H "Authorization: Bearer *** https://api.stripe.com/v1/charges

# Use in environment variable
export STRIPE_KEY=$(indietool secret get stripe-key -S)
```

---

## 🔒 Security & Privacy

**Where secrets live:**
- Secrets are encrypted with AES-256-GCM at `~/.config/indietool/secrets/`
- The encryption key is stored in your OS keyring (default) or as an age-encrypted file (`~/.config/indietool/keys/`)
- **No cloud, no sync, no telemetry** — everything stays on your machine

**Threat model:**
- `indietool` never phones home. Zero telemetry.
- Secrets are useless without your encryption key (OS keyring or SSH private key)
- If you lose your machine, secrets are unrecoverable (by design)
- For cross-host access, use `age-ssh` backend + SSH agent forwarding

**Provider tokens:**
- API keys for Cloudflare/Porkbun/etc. are stored in `~/.config/indietool/config.yaml`
- File permissions are restricted to your user (`chmod 600`)
- Consider using environment variables or `indietool secrets` for sensitive tokens in shared environments

---

## 🧠 FAQ

### ❓ Which providers are supported?

| Provider        | Domains | DNS | Secrets |
| --------------- | ------- | --- | ------- |
| Cloudflare      | ✅      | ✅  | ❌      |
| Porkbun         | ✅      | ✅  | ❌      |
| Namecheap       | ✅      | ✅  | ❌      |
| GoDaddy         | ✅      | 🚧  | ❌      |
| The Little Host | ❌      | ✅  | ❌      |
| Local           | ❌      | ❌  | ✅      |

**Legend:**

- ✅ Full support
- 🚧 In development (GoDaddy DNS coming soon)
- ❌ Not supported

**Notes:**

- **Domains**: Domain registration management, expiry tracking, nameserver updates
- **Cloudflare Registrar (beta)**: buy domains via API (`domain check` / `domain register`) and manage auto-renew (`domains renew`, `domain get`, `domain set`) through the new Registrar API; privacy/lock remain dashboard-only until the API supports them

**Sandbox testing**: point the Cloudflare provider at the Registrar Sandbox API (test environment, no billing) with "indietool config add provider cloudflare --sandbox" (or "sandbox: true" in the provider config). The sandbox mirrors the production Registrar API under /registrar-sandbox/... and supports only com/net; purchases are free but persist. It requires an API token with Registrar Sandbox permissions and full contact data on register (no Express Mode).
- **DNS**: DNS record management (list, create, update, delete) with ID-based targeting
- **Secrets**: Local encrypted secret storage (OS keyring or age-ssh backend)

---

### ❓ Where are my secrets stored?

Encrypted locally at `~/.config/indietool/secrets/`. The encryption key is stored in your OS keyring (default) or as an age-encrypted file at `~/.config/indietool/keys/` when using the age-ssh backend.

---

### ❓ What if I lose my computer?

Secrets are useless without your encryption key. With the default keyring backend, start fresh on the new machine. With the age-ssh backend, your key file (`~/.config/indietool/keys/`) and SSH private key are both required — back up the keys directory if you need portability.

---

### ❓ Does it work on Windows?

**Experimental.** Windows binaries are available in [releases](https://github.com/indietool/cli/releases), but testing is limited. macOS and Linux are fully supported.

If you're on Windows and hit issues, please [open an issue](https://github.com/indietool/cli/issues) — we want to make it work!

---

## 🧯 Troubleshooting

### Secrets aren't saving?

- Ensure your keyring (Keychain or gnome-keyring) is unlocked
- Check file permissions on `~/.config/indietool/secrets/`

### Secrets not working in SSH sessions?

GUI keyrings like GNOME Keyring require a graphical session to unlock, so they fail over SSH. Use the age-ssh backend instead:

```bash
indietool secrets init --backend age-ssh
```

This encrypts the database key with your SSH public key. To decrypt over SSH, connect with agent forwarding enabled:

```bash
ssh -A yourserver.example.com
# or add 'ForwardAgent yes' to your ~/.ssh/config
```

### "Permission denied" errors?

- Check file permissions on `~/.config/indietool`
- Ensure your user has write access to the config directory

### API key errors with registrars?

- Double check key/secret pair
- Some registrars require IP allowlisting or scopes

---

## 🚫 Limitations

- 🧩 Registrar support: Cloudflare, Porkbun, Namecheap, GoDaddy
- ☁️ DNS management: GoDaddy implementation in progress
- 💻 CLI only — no web UI or GUI planned
- 🔄 Secrets not synced across machines (by design — use age-ssh for cross-host access via SSH agent forwarding)
- 🪟 Windows support is experimental

---

## ❤️ Built for indie builders who just want to ship

- 🌐 [indietool.dev](https://indietool.dev)
- 📦 [GitHub Releases](https://github.com/indietool/cli/releases)
- 🐛 [Report issues](https://github.com/indietool/cli/issues)
- 💬 [Discussions & roadmap](https://github.com/indietool/cli/discussions)
