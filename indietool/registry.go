package indietool

import (
	"context"
	"reflect"

	"indietool/cli/domains"
	"indietool/cli/providers"
)

// Provider defines the interface for service provider integrations
type Provider interface {
	// Identification
	Name() string

	// Authentication & Setup
	Validate(ctx context.Context) error

	IsEnabled() bool
	SetEnabled(bool)

	// AsRegistrar returns the registrar interface for domain operations
	AsRegistrar() domains.Registrar
}

// Registry manages multiple provider instances
type Registry struct {
	providers Providers
}

type Providers struct {
	Cloudflare *providers.CloudflareProvider
	Porkbun    *providers.PorkbunProvider
	Namecheap  *providers.NamecheapProvider
	GoDaddy    *providers.GoDaddyProvider
}

func GetProviders[T any](registry *Registry) []T {
	var result []T

	v := reflect.ValueOf(&registry.providers).Elem()
	targetType := reflect.TypeOf((*T)(nil)).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		if !field.IsNil() && field.Type().Implements(targetType) {
			if provider, ok := field.Interface().(Provider); ok && provider.IsEnabled() {
				if typed, ok := field.Interface().(T); ok {
					result = append(result, typed)
				}
			}
		}
	}

	return result
}

func NewRegistry(cfg *Config) (*Registry, error) {
	registry := &Registry{
		providers: Providers{},
	}

	// Initialize providers directly with config
	if cfg.Providers.Cloudflare != nil {
		registry.providers.Cloudflare = providers.NewCloudflare(*cfg.Providers.Cloudflare)
	}

	if cfg.Providers.Porkbun != nil {
		registry.providers.Porkbun = providers.NewPorkbun(*cfg.Providers.Porkbun)
	}

	if cfg.Providers.Namecheap != nil {
		registry.providers.Namecheap = providers.NewNamecheap(*cfg.Providers.Namecheap)
	}

	if cfg.Providers.GoDaddy != nil {
		registry.providers.GoDaddy = providers.NewGoDaddy(*cfg.Providers.GoDaddy)
	}

	return registry, nil
}

// List returns the names of all configured providers
func (r *Registry) List() []string {
	var names []string

	if r.providers.Cloudflare != nil {
		names = append(names, "cloudflare")
	}
	if r.providers.Porkbun != nil {
		names = append(names, "porkbun")
	}
	if r.providers.Namecheap != nil {
		names = append(names, "namecheap")
	}
	if r.providers.GoDaddy != nil {
		names = append(names, "godaddy")
	}

	return names
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, bool) {
	switch name {
	case "cloudflare":
		if r.providers.Cloudflare != nil {
			return r.providers.Cloudflare, true
		}
	case "porkbun":
		if r.providers.Porkbun != nil {
			return r.providers.Porkbun, true
		}
	case "namecheap":
		if r.providers.Namecheap != nil {
			return r.providers.Namecheap, true
		}
	case "godaddy":
		if r.providers.GoDaddy != nil {
			return r.providers.GoDaddy, true
		}
	}
	return nil, false
}

// GetEnabledProviders returns providers that are configured and enabled
func (r *Registry) GetEnabledProviders() []Provider {
	var enabled []Provider

	if r.providers.Cloudflare != nil && r.providers.Cloudflare.IsEnabled() {
		enabled = append(enabled, r.providers.Cloudflare)
	}
	if r.providers.Porkbun != nil && r.providers.Porkbun.IsEnabled() {
		enabled = append(enabled, r.providers.Porkbun)
	}
	if r.providers.Namecheap != nil && r.providers.Namecheap.IsEnabled() {
		enabled = append(enabled, r.providers.Namecheap)
	}
	if r.providers.GoDaddy != nil && r.providers.GoDaddy.IsEnabled() {
		enabled = append(enabled, r.providers.GoDaddy)
	}

	return enabled
}
