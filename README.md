# indietools CLI

A powerful command-line tool designed specifically for indie builders and small-time developers to streamline domain management and infrastructure tasks. indietools addresses the unique challenges faced by indie hackers who manage multiple apps and services across different providers.

## 🎯 Purpose

indietools tackles the infrastructure and operational challenges that indie builders face daily:

- **Domain & DNS Management**: Scattered domains across multiple registrars with inconsistent DNS configurations
- **Multi-cloud Sprawl**: Apps deployed across different providers without unified oversight
- **Operational Overhead**: Manual processes for domain discovery, availability checking, and resource management

By providing a unified CLI interface, indietools reduces cognitive overhead and automates repetitive tasks, allowing you to focus on building features rather than managing infrastructure.

## 🚀 Features

### ✅ Currently Available

- **Domain Availability Search**: Check the registration status of specific domains using RDAP (Registration Data Access Protocol)
- **Domain Exploration**: Discover available domains across popular TLDs favored by indie hackers
- **Domain Management**: List and manage domains across multiple registrar providers
- **Provider Integration**: Support for Cloudflare, Porkbun, Namecheap, and GoDaddy (more coming)
- **Concurrent Processing**: Fast, parallel domain checking for efficient bulk operations
- **Multiple Output Formats**: Human-readable tables, JSON, and YAML output for automation
- **Flexible Display Options**: Compact and wide table views with customizable columns
- **Custom TLD Lists**: Support for custom TLD specifications via command line or file input
- **Configuration Management**: Automatic config file creation and provider setup

### 🚧 Planned Features

- **DNS Management**: Centralized DNS record management across different providers
- **Infrastructure Dashboard**: Unified view of services across cloud providers
- **Cost Tracking**: Monitor spending across multiple services and platforms
- **Security Monitoring**: Track SSL certificates, domain expiration, and security compliance

## 🛠 Installation

### Prerequisites

- Go 1.24.3 or later

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd indietools/cli

# Build the binary
go build -o dist/indietools cmd/indietools/main.go

# (Optional) Install globally
go install cmd/indietools/main.go
```

### First Run

On first run, indietools automatically creates a configuration file at `~/.config/indietools.yaml` with sensible defaults. You can immediately start using the tool without manual configuration.

## 📖 Usage

### Domain Commands

#### Search Specific Domains

Check the availability of one or more specific domains:

```bash
# Check a single domain
indietools domain search example.com

# Check multiple domains
indietools domain search example.com google.com mydomain.org

# Wide format with additional columns
indietools domain search example.com --wide

# Output in JSON format
indietools domain search example.com --json

# No colors for CI/automation
indietools domain search example.com --no-color
```

**Example Output:**

```
DOMAIN           STATUS      TLD
example.com      Taken       com
mydomain.org     Available   org

2 domains checked: 1 available, 1 taken
```

#### Explore Domain Across TLDs

Discover available variations of a domain name across popular TLDs:

```bash
# Explore using default popular TLDs
indietools domain explore buildhub

# Explore using custom TLDs
indietools domain explore mycompany --tlds com,org,dev,ai,io

# Load TLDs from a file
indietools domain explore startup --tlds @tlds.txt

# Wide format with cost and registrar info
indietools domain explore webapp --wide

# JSON output for automation
indietools domain explore webapp --json
```

**Example Output:**

```
DOMAIN           STATUS      TLD
buildhub.app     Available   app
buildhub.dev     Available   dev
buildhub.com     Taken       com
buildhub.org     Taken       org

50 domains checked: 23 available, 25 taken, 2 errors
```

#### Manage Domains

List and manage domains across your configured providers:

```bash
# List all domains from configured providers
indietools domains list

# Wide format with expiry dates and costs
indietools domains list --wide

# JSON output
indietools domains list --json
```

### Configuration

#### Provider Setup

Configure domain registrar providers:

```bash
# Add Cloudflare provider
indietools config add provider cloudflare \
  --account-id YOUR_ACCOUNT_ID \
  --api-token YOUR_TOKEN \
  --email your@email.com

# Add Porkbun provider
indietools config add provider porkbun \
  --api-key YOUR_KEY \
  --api-secret YOUR_SECRET

# Add Namecheap provider
# Optional: use --client-ip to specify your IP address (must be allowlisted on Namecheap).
            Auto-detected via an external service, https://ipinfo.io, if left as the default value, `auto`
indietools config add provider namecheap \
  --api-key YOUR_KEY \
  --username YOUR_USERNAME

# Add GoDaddy provider
indietools config add provider godaddy \
  --api-key YOUR_KEY \
  --api-secret YOUR_SECRET
```

#### Configuration File

indietools supports configuration via:

- **Config File**: `~/.config/indietools.yaml` (auto-created) or specify with `--config`
- **Command Flags**: Override config and environment settings

**Example Configuration:**

```yaml
domains:
  providers: ["cloudflare", "porkbun"]
  management:
    expiry_warning_days: [30, 7, 1]

providers:
  cloudflare:
    account_id: your_account_id_here
    api_token: your_token_here
    email: you@example.com
    enabled: true
  porkbun:
    api_key: your_key_here
    api_secret: your_secret_here
    enabled: true
```

## 🔧 Command Reference

### Global Flags

- `--config`: Specify custom configuration file path
- `--json`: Output results in JSON format (available on most commands)
- `--help`: Show help information

### Domain Commands

#### `indietools domain search [domains...]`

Search for specific domain availability.

**Arguments:**

- `domains...`: One or more domain names to check

**Flags:**

- `--wide, -w`: Show additional columns (registrar, cost, expiry, error details)
- `--json`: Output results in JSON format
- `--no-color`: Disable colored output
- `--no-headers`: Don't show column headers

#### `indietools domain explore [domain-name]`

Explore domain availability across multiple TLDs.

**Arguments:**

- `domain-name`: Base domain name (with or without TLD)

**Flags:**

- `--tlds`: Comma-separated TLD list or `@filename` for file input
- `--wide, -w`: Show additional columns (cost, expiry, error details)
- `--json`: Output results in JSON format
- `--no-color`: Disable colored output
- `--no-headers`: Don't show column headers

#### `indietools domains list`

List domains from configured providers.

**Flags:**

- `--wide, -w`: Show additional columns (expiry dates, costs, registrar details)
- `--json`: Output results in JSON format
- `--no-color`: Disable colored output
- `--no-headers`: Don't show column headers

#### `indietools config add provider [provider]`

Configure domain registrar providers.

**Available Providers:**

- `cloudflare`: Cloudflare Registrar
- `porkbun`: Porkbun
- `namecheap`: Namecheap
- `godaddy`: GoDaddy

**Popular TLDs (Default for explore):**
`com`, `net`, `org`, `dev`, `app`, `io`, `co`, `me`, `ai`, `sh`, `ly`, `gg`, `cc`, `tv`, `fm`, `tech`, `online`, `site`, `xyz`, `lol`, `wtf`, `cool`, `fun`, `live`, `blog`, `life`, `world`, `cloud`, `digital`, `email`, `studio`, `agency`, `design`, `media`, `social`, `team`, `tools`, `works`, `tips`, `guru`, `ninja`, `expert`, `pro`, `biz`, `info`, `name`, `ventures`, `solutions`, `services`, `consulting`

## 📝 License

This project is licensed under the Apache 2.0 License - see the LICENSE file for details.

## 🐛 Known Issues

- Some TLD registries may have rate limits affecting concurrent queries
- Provider API integration is still in development for some features
- Wide format columns may not display data for all providers yet

---

**Built with ❤️ for the indie builder community**

_Reduce infrastructure overhead. Focus on shipping._
