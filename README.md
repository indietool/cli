# `indietool`

Tired of bouncing between registrars, tracking domain renewals in spreadsheets, and copy-pasting secrets into `.env` files?

🎯 `indietool` is a CLI tool for indie builders that helps you

- 🌍 Hunt domain names across 50+ TLDs — in seconds
- 🗓️ Track expiries across registrars like Cloudflare & Porkbun
- ☁️ Manage DNS records across providers with auto-detection
- 🔐 Securely store API keys & secrets — OS keyring or SSH-key encrypted

No dashboards. No vendor lock-in. Just you and your terminal.

---

## 🚀 Quick Start

### Installation

```bash
# macOS/Linux (recommended)
go install github.com/indietool/cli@latest
```

### Try it in 30 seconds

```bash
# Check domain availability
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

**Problem:** Domains expire. You don’t want surprises.
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
A     ☁️@      192.168.1.1
A     www      192.168.1.2
CNAME ☁️api    example.com
MX    @        10 mail.example.com
```

_Note: ☁️ indicates Cloudflare proxied records, available only with the Cloudflare provider, for domains hosted on Cloudflare_

#### Get detailed view

```bash
indietool dns list example.com --wide
```

```
TYPE  NAME     CONTENT          TTL   PRIORITY  ID
A     ☁️@      192.168.1.1      300             abc123
A     www      192.168.1.2      300             def456
CNAME ☁️api    example.com      300             ghi789
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

- ✅ **Cloudflare** - Full CRUD operations with proxy status indicators
- ✅ **Porkbun** - Complete DNS record management (list, set, delete)
- ✅ **Namecheap** - Full CRUD support with batch operations
- ✅ **The Little Host** - Full DNS record management

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
| Encryption Key        | keyring   | OS Keyring (`Keychain`, `gnome-keyring`)          | ✅        |
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
✓ Auto-generated encryption key for database 'default'
✓ Secret 'stripe-key' stored successfully
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

#### Use in environment variable

```bash
export STRIPE_KEY=$(indietool secret get stripe-key -S)
```

---

## 🧠 FAQ

### ❓ Which providers are supported?

| Provider        | Domains | DNS | Secrets |
| --------------- | ------- | --- | ------- |
| Cloudflare      | ✅      | ✅  | ❌      |
| Porkbun         | ✅      | ✅  | ❌      |
| Namecheap       | ✅      | ✅  | ❌      |
| GoDaddy         | ✅      | ❌  | ❌      |
| The Little Host | ❌      | ✅  | ❌      |
| Local           | ❌      | ❌  | ✅      |

**Legend:**

- ✅ Full support
- 🚧 In development
- ❌ Not supported

**Notes:**

- **Domains**: Domain registration management, expiry tracking, nameserver updates
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

Not yet — currently macOS and Linux only. Windows support is planned.

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

- ❌ No Windows support (yet)
- 🧩 Registrar support: Cloudflare, Porkbun, Namecheap, GoDaddy
- ☁️ DNS management: GoDaddy implementation in progress
- 💻 CLI only — no web UI or GUI planned
- 🔄 Secrets not synced across machines (by design — use age-ssh for cross-host access via SSH agent forwarding)

---

## ❤️ Built for indie builders who just want to ship

Stop wasting time in control panels and spreadsheets.
Let `indietool` handle the busywork — so you can focus on building.

[Get Started →](https://indietool.dev)
