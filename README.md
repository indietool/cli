# `indietool`

Tired of bouncing between registrars, tracking domain renewals in spreadsheets, and copy-pasting secrets into `.env` files?

🎯 `indietool` is a CLI tool for indie builders that helps you

- 🌍 Hunt domain names across 50+ TLDs — in seconds
- 🗓️ Track expiries across registrars like Cloudflare & Porkbun
- 🔐 Securely store API keys & secrets using your OS keyring

No dashboards. No vendor lock-in. Just you and your terminal.

---

## 🚀 Quick Start

### Installation

```bash
# macOS/Linux (recommended)
curl -sSL https://get.indietool.dev | sh

# Or download manually
wget https://github.com/indietool/indietool/releases/latest/download/indietool-linux-amd64
```

### Try it in 30 seconds

```bash
# Check domain availability
indietool domain explore myapp

# Initialize encrypted secrets
indietool secrets init

# Save a test API key
indietool secret set stripe-key "sk_test_..." --note "Stripe test key"
```

---

## 🔁 Everyday Developer Flows

### Weekend Project Setup

```bash
# Check which domains are available
indietool domain explore myproject --tlds dev,com,ai

# Store your API keys securely
indietool secret set openai-key "sk-..." --note "OpenAI API key"
indietool secret set stripe-key "sk_test_..." --note "Stripe test key"
```

### Production Deployment

```bash
# Check domain expiry before renewal
indietool domains list --provider cloudflare

# Export secrets for deployment
export OPENAI_KEY=$(indietool secret get openai-key --show --json | jq -r '.value')
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
```

Then list your domains:

```bash
indietool domains list
```

```
DOMAIN              PROVIDER     STATUS    EXPIRY
myawesomeapp.com    cloudflare   Active    2025-12-15
sideproject.ai      cloudflare   Active    2025-08-10
```

Need more info?

```bash
indietool domains list --wide
```

```
DOMAIN              COST    REGISTRAR
sideproject.ai      -       Cloudflare
...
Total annual cost: $82.24
```

---

### 🔐 Secure Local Secrets Without the Hassle

**Problem:** Secrets are either insecure or annoying to manage.
**Solution:** `indietool secrets` encrypts secrets using your OS keyring — no cloud, no sync, no complicated setup to manage.

#### How it works

| Component        | Stored At                                | Encrypted |
| ---------------- | ---------------------------------------- | --------- |
| Secrets Database | `~/.config/indietool/secrets/`           | ✅        |
| Encryption Key   | OS Keyring (`Keychain`, `gnome-keyring`) | ✅        |

#### Initialize encryption

```bash
indietool secrets init
✓ Encryption key initialized
```

#### Store a secret

```bash
indietool secret set stripe-key "sk_test_..." --note "Stripe test key"
✓ Secret stored successfully
```

#### Retrieve a secret

```bash
# Safe output (masked)
indietool secret get stripe-key

# Show actual value
indietool secret get stripe-key --show
```

#### List all secrets

```bash
indietool secret list
```

#### Use in environment variable

```bash
export STRIPE_KEY=$(indietool secret get stripe-key --show --json | jq -r '.value')
```

---

## 🧠 FAQ

### ❓ Where are my secrets stored?

Encrypted locally at `~/.config/indietool/secrets/`, using an encryption key stored in your OS keyring.

---

### ❓ What if I lose my computer?

Secrets are useless without your OS user account + keyring. Just run `indietool secrets init` on your new machine.

---

### ❓ Which registrars are supported?

Currently:

- ✅ Cloudflare
- ✅ Porkbun
  More coming soon (Namecheap, Google Domains, etc.)

---

### ❓ Does it work on Windows?

Not yet — currently macOS and Linux only. Windows support is planned.

---

## 🧯 Troubleshooting

### Secrets aren't saving?

- Run `indietool secrets init` again
- Ensure your keyring (Keychain or gnome-keyring) is unlocked

### "Permission denied" on init?

- Check file permissions on `~/.config/indietool`
- Try running with elevated permissions once: `sudo indietool secrets init`

### API key errors with registrars?

- Double check key/secret pair
- Some registrars require IP allowlisting or scopes

---

## 🚫 Limitations

- ❌ No Windows support (yet)
- 🧩 Only supports a few registrars (Cloudflare, Porkbun)
- 💻 CLI only — no web UI or GUI planned
- 🔄 Secrets not synced across machines (by design)

---

## ❤️ Built for indie builders who just want to ship

Stop wasting time in control panels and spreadsheets.
Let `indietool` handle the busywork — so you can focus on building.

[Get Started →](https://indietool.dev)
