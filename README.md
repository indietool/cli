# indietool CLI

A command-line tool designed specifically for indie builders and small-time developers to streamline domain management and infrastructure tasks. indietool addresses the unique challenges faced by indie hackers who manage multiple apps and services across different providers.

## 🎯 Purpose

indietool tackles the infrastructure and operational challenges that indie builders face daily:

- **Domain & DNS Management**: Scattered domains across multiple registrars with inconsistent DNS configurations
- **Multi-cloud Sprawl**: Apps deployed across different providers without unified oversight
- **Operational Overhead**: Manual processes for domain discovery, availability checking, and resource management

By providing a unified CLI interface, indietool reduces cognitive overhead and automates repetitive tasks, allowing you to focus on building features rather than managing infrastructure.

## 🚀 Features

### ✅ Currently Available

- **Domain Availability Search**: Check the registration status of specific domains using RDAP (Registration Data Access Protocol)
- **Domain Exploration**: Discover available domains across popular TLDs favored by indie hackers
- **Concurrent Processing**: Fast, parallel domain checking for efficient bulk operations
- **Multiple Output Formats**: Human-readable tables and JSON output for automation
- **Custom TLD Lists**: Support for custom TLD specifications via command line or file input

### 🚧 Planned Features

- **DNS Management**: Centralized DNS record management across different providers
- **Infrastructure Dashboard**: Unified view of services across cloud providers
- **Cost Tracking**: Monitor spending across multiple services and platforms
- **Security Monitoring**: Track SSL certificates, domain expiration, and security compliance

## 📁 Project Structure

```
├── cmd/
│   └── indietool/
│       ├── main.go                 # Application entry point
│       └── cmd/
│           ├── root.go             # Root command and configuration
│           ├── domain.go           # Domain command group
│           ├── domain_search.go    # Domain availability search
│           ├── domain_explore.go   # Domain exploration across TLDs
│           └── dns.go              # DNS management (planned)
├── domains/
│   ├── search.go                   # Domain search logic and RDAP integration
│   └── explore.go                  # Domain exploration and result organization
├── output/
│   └── formatter.go                # Output formatting (JSON/human-readable)
├── go.mod                          # Go module dependencies
├── go.sum                          # Dependency checksums
└── indietool                       # Compiled binary
```

## 🛠 Installation

### Prerequisites

- Go 1.24.3 or later

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd indietool/cli

# Build the binary
go build -o indietool cmd/indietool/main.go

# (Optional) Install globally
go install cmd/indietool/main.go
```

## 📖 Usage

### Domain Commands

#### Search Specific Domains

Check the availability of one or more specific domains:

```bash
# Check a single domain
indietool domain search example.com

# Check multiple domains
indietool domain search example.com google.com mydomain.org

# Output in JSON format
indietool domain search example.com --json
```

**Example Output:**
```
Domain Availability Search Results
==================================

Domain: example.com
  Status: ✗ NOT AVAILABLE
  Details: client transfer prohibited

Domain: mydomain.org
  Status: ✓ AVAILABLE
```

#### Explore Domain Across TLDs

Discover available variations of a domain name across popular TLDs:

```bash
# Explore using default popular TLDs
indietool domain explore kopitiam

# Explore using custom TLDs
indietool domain explore mycompany --tlds com,org,dev,ai,io

# Load TLDs from a file
indietool domain explore startup --tlds @tlds.txt

# JSON output for automation
indietool domain explore webapp --json
```

**Example Output:**
```
Domain Exploration Results for "kopitiam"
=========================================

Summary: 50 domains checked
  ✓ Available: 23
  ✗ Taken: 25
  ⚠ Errors: 2

✓ AVAILABLE DOMAINS:
  kopitiam.app
  kopitiam.dev
  kopitiam.sh
  kopitiam.ai
  ...

✗ TAKEN DOMAINS:
  kopitiam.com (client transfer prohibited)
  kopitiam.org (registered)
  ...
```

### Configuration

indietool supports configuration via:

- **Config File**: `~/.indietool.yaml` (default) or specify with `--config`
- **Environment Variables**: Automatically loaded with `INDIETOOL_` prefix
- **Command Flags**: Override config and environment settings

## 🔧 Command Reference

### Global Flags

- `--config`: Specify custom configuration file path
- `--help`: Show help information

### Domain Commands

#### `indietool domain search [domains...]`

Search for specific domain availability.

**Arguments:**
- `domains...`: One or more domain names to check

**Flags:**
- `--json`: Output results in JSON format

#### `indietool domain explore [domain-name]`

Explore domain availability across multiple TLDs.

**Arguments:**
- `domain-name`: Base domain name (with or without TLD)

**Flags:**
- `--tlds`: Comma-separated TLD list or `@filename` for file input
- `--json`: Output results in JSON format

**Popular TLDs (Default):**
`com`, `net`, `org`, `dev`, `app`, `io`, `co`, `me`, `ai`, `sh`, `ly`, `gg`, `cc`, `tv`, `fm`, `tech`, `online`, `site`, `xyz`, `lol`, `wtf`, `cool`, `fun`, `live`, `blog`, `life`, `world`, `cloud`, `digital`, `email`, `studio`, `agency`, `design`, `media`, `social`, `team`, `tools`, `works`, `tips`, `guru`, `ninja`, `expert`, `pro`, `biz`, `info`, `name`, `ventures`, `solutions`, `services`, `consulting`

### DNS Commands (Planned)

```bash
# Planned DNS management commands
indietool dns list                  # List all DNS records
indietool dns add                   # Add DNS record
indietool dns update                # Update DNS record
indietool dns delete                # Delete DNS record
```

## 🧰 Dependencies

- **[Cobra](https://github.com/spf13/cobra)**: CLI framework for Go
- **[Viper](https://github.com/spf13/viper)**: Configuration management
- **[openrdap/rdap](https://github.com/openrdap/rdap)**: RDAP client for domain queries

## 🤝 Contributing

indietool is designed by indie builders, for indie builders. Contributions are welcome!

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🎯 Roadmap

- [ ] **DNS Management**: Complete DNS command implementation
- [ ] **Multi-provider Support**: Integrate with popular DNS providers (Cloudflare, Route53, etc.)
- [ ] **Infrastructure Monitoring**: Track services across different cloud providers
- [ ] **Cost Analytics**: Monitor and optimize spending across platforms
- [ ] **SSL Certificate Management**: Automated certificate lifecycle management
- [ ] **Deployment Automation**: Streamlined deployment across different providers
- [ ] **Team Collaboration**: Multi-user support with role-based access

---

**Built with ❤️ for the indie builder community**

*Reduce infrastructure overhead. Focus on shipping.*
