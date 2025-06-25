# Config Package Usage

This package provides strongly-typed configuration structures for the indietool CLI.

## Configuration File Locations

The `LoadConfig()` function searches for configuration files in the following locations (first match wins):

1. `~/.indietool.yaml`
2. `~/.config/indietool.yaml`

You can also load from a specific path using `LoadConfigFromPath(path)` or integrate with your existing Viper setup using `LoadFromViper()`.

## Available Loading Functions

- **`LoadConfig()`** - Searches standard locations (`~/.indietool.yaml`, `~/.config/indietool.yaml`)
- **`LoadConfigWithDefaults()`** - Same as `LoadConfig()` but applies sensible defaults
- **`LoadConfigFromPath(path)`** - Load from a specific file path
- **`LoadConfigFromHome()`** - Load from `~/.indietool.yaml`
- **`LoadConfigFromCurrentDir()`** - Load from `./config.yaml`
- **`LoadFromViper()`** - Load from initialized Viper instance
- **`LoadFromViperWithDefaults()`** - Same as `LoadFromViper()` but applies defaults

## Basic Usage

### Automatic Config Discovery (Recommended)

```go
package main

import (
    "fmt"
    "log"
    "indietool/cli/config"
)

func main() {
    // Load config from standard locations with defaults
    cfg, err := config.LoadConfigWithDefaults()
    if err != nil {
        log.Fatal(err)
    }

    // Check if config is valid
    if !cfg.Valid() {
        log.Fatal("No valid configuration found")
    }

    // Use the config...
    enabledRegistrars := cfg.GetEnabledRegistrars()
    fmt.Printf("Enabled registrars: %v\n", enabledRegistrars)
    fmt.Printf("Config loaded from: %s\n", cfg.Path)
}
```

### Load from Specific Path

```go
func main() {
    // Load from a specific file path
    cfg, err := config.LoadConfigFromPath("/path/to/my-config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Use the config...
}
```

### With Existing Viper Setup

```go
package main

import (
    "fmt"
    "log"
    "indietool/cli/config"
)

func main() {
    // Load config using the existing Viper setup
    cfg, err := config.LoadFromViperWithDefaults()
    if err != nil {
        log.Fatal(err)
    }

    // Check which registrars are enabled
    enabledRegistrars := cfg.GetEnabledRegistrars()
    fmt.Printf("Enabled registrars: %v\n", enabledRegistrars)

    // Type-safe access to specific registrar config
    if cfConfig := cfg.GetCloudflareConfig(); cfConfig != nil && cfConfig.Enabled {
        fmt.Printf("Cloudflare API Key: %s\n", cfConfig.APIKey)
        fmt.Printf("Cloudflare Email: %s\n", cfConfig.Email)
    }

    // Safe check for enabled registrars
    if cfg.IsRegistrarEnabled("cloudflare") {
        cfConfig := cfg.GetCloudflareConfig()
        // cfConfig is guaranteed to be non-nil here
        fmt.Printf("Using Cloudflare with API key: %s\n", cfConfig.APIKey)
    }

    // Check if registrar is configured (regardless of enabled status)
    if cfg.HasRegistrarConfig("namecheap") {
        ncConfig := cfg.GetNamecheapConfig()
        if ncConfig.Enabled {
            fmt.Printf("Namecheap is enabled\n")
        } else {
            fmt.Printf("Namecheap is configured but disabled\n")
        }
    }

    // Validate configuration
    if errors := cfg.ValidateConfig(); len(errors) > 0 {
        fmt.Printf("Config validation errors: %v\n", errors)
    }
}
```

### Direct File Loading

```go
package main

import (
    "fmt"
    "log"
    "indietool/cli/config"
)

func main() {
    // Load from standard locations
    cfg, err := config.LoadConfig() // searches ~/.indietool.yaml, ~/.config/indietool.yaml
    if err != nil {
        log.Fatal(err)
    }

    // Or load from home directory specifically
    // cfg, err := config.LoadConfigFromHome() // ~/.indietool.yaml

    // Or load from current directory
    // cfg, err := config.LoadConfigFromCurrentDir() // ./config.yaml

    // Use the config
    for _, registrar := range cfg.GetEnabledRegistrars() {
        regConfig := cfg.GetRegistrarConfig(registrar)
        fmt.Printf("Registrar %s config: %+v\n", registrar, regConfig)
    }
}
```

## Configuration Structure

The config package expects a YAML structure like:

```yaml
domains:
  registrars:
    cloudflare:
      api_key: "your-cloudflare-api-key"
      email: "your-email@example.com"
      enabled: true
    namecheap:
      api_key: "your-namecheap-api-key"
      api_secret: "your-namecheap-secret"
      username: "your-username"
      sandbox: false
      enabled: true
    # ... other registrars
  management:
    expiry_warning_days: [30, 7, 1]
```

## Key Features

- **Type Safety**: All configuration values are strongly typed
- **Optional Registrars**: Registrars are pointers with `omitempty` - only configured registrars appear in YAML
- **Graceful Failure**: Config loading failures don't crash the application
- **Validation**: Built-in validation for required fields and valid values
- **Valid() Method**: Check if configuration was successfully loaded with `config.Valid()`
- **Path Tracking**: Know exactly where configuration was loaded from
- **Viper Integration**: Works seamlessly with existing Viper setup
- **Convenience Methods**: Helper methods for common operations
- **Default Values**: Automatic application of sensible defaults
- **Null Safety**: All registrar access methods handle nil pointers gracefully

## Available Registrars

- **Cloudflare**: `api_key`, `email`, `enabled`
- **Namecheap**: `api_key`, `api_secret`, `username`, `sandbox`, `enabled`
- **Porkbun**: `api_key`, `api_secret`, `enabled`
- **GoDaddy**: `api_key`, `api_secret`, `environment`, `enabled`

## Helper Methods

### Configuration Access

- `Valid()` - Check if configuration was successfully loaded from a file
- `GetEnabledRegistrars()` - Returns list of enabled registrar names
- `IsRegistrarEnabled(name)` - Check if specific registrar is enabled
- `HasRegistrarConfig(name)` - Check if registrar is configured (regardless of enabled status)
- `GetRegistrarConfig(name)` - Get config map for specific registrar
- `ValidateConfig()` - Validate configuration and return any errors

### Type-Safe Registrar Access

- `GetCloudflareConfig()` - Returns `*CloudflareConfig` or `nil`
- `GetNamecheapConfig()` - Returns `*NamecheapConfig` or `nil`
- `GetPorkbunConfig()` - Returns `*PorkbunConfig` or `nil`
- `GetGoDaddyConfig()` - Returns `*GoDaddyConfig` or `nil`

### Configuration Management

- `SetCloudflareConfig(config *CloudflareConfig)` - Set Cloudflare config
- `SetNamecheapConfig(config *NamecheapConfig)` - Set Namecheap config
- `SetPorkbunConfig(config *PorkbunConfig)` - Set Porkbun config
- `SetGoDaddyConfig(config *GoDaddyConfig)` - Set GoDaddy config
