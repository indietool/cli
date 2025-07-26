package secrets

import (
	"strings"
	"time"
)

// Secret represents a stored secret with metadata
type Secret struct {
	Name      string     `json:"name"`
	Value     string     `json:"value"`
	Note      string     `json:"note,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// SecretListItem represents a secret in list view (without the actual value)
type SecretListItem struct {
	Name      string     `json:"name"`
	Note      string     `json:"note,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Expired   bool       `json:"expired"`
}

// Config represents the secrets configuration
type Config struct {
	DefaultDatabase string `yaml:"default_database"`
	StorageDir      string `yaml:"storage_dir"`
	ClipboardTTL    int    `yaml:"clipboard_ttl_seconds"`
	MaskOutput      bool   `yaml:"output_masked"`
}

// ParseSecretIdentifier parses name[@database] syntax and returns the components
func ParseSecretIdentifier(identifier string) (name, database string) {
	parts := strings.SplitN(identifier, "@", 2)
	name = parts[0]
	if len(parts) > 1 {
		database = parts[1]
	} else {
		database = "" // Will use default database
	}
	return
}

// IsExpired checks if a secret has expired
func (s *Secret) IsExpired() bool {
	return s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt)
}

// ToListItem converts a Secret to a SecretListItem (without exposing the value)
func (s *Secret) ToListItem() *SecretListItem {
	return &SecretListItem{
		Name:      s.Name,
		Note:      s.Note,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		ExpiresAt: s.ExpiresAt,
		Expired:   s.IsExpired(),
	}
}

