package secrets

import (
	"encoding/json"
	"fmt"
	"time"
)

// Manager coordinates secret operations across encryption and storage layers
type Manager struct {
	config    *Config
	storage   *Storage
	encryptor *Encryptor
}

// NewManager creates a new secrets manager instance
func NewManager(config *Config) (*Manager, error) {
	storage, err := NewStorage(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	encryptor, err := NewEncryptor()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize encryptor: %w", err)
	}

	return &Manager{
		config:    config,
		storage:   storage,
		encryptor: encryptor,
	}, nil
}

// InitDatabase initializes encryption for the specified database
func (m *Manager) InitDatabase(database, keyPath string) error {
	return m.encryptor.InitializeKey(database, keyPath)
}

// SetSecret stores an encrypted secret
func (m *Manager) SetSecret(name, value, database, note string, expiresAt *time.Time) error {
	if database == "" {
		database = m.config.GetDefaultDatabase()
	}

	// Check if secret already exists to preserve creation time
	var createdAt time.Time
	if existing, err := m.GetSecret(name, database); err == nil {
		createdAt = existing.CreatedAt
	} else {
		createdAt = time.Now()
	}

	secret := &Secret{
		Name:      name,
		Value:     value,
		Note:      note,
		CreatedAt: createdAt,
		UpdatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	data, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("failed to marshal secret: %w", err)
	}

	encrypted, err := m.encryptor.Encrypt(data, database)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	return m.storage.Set(database, name, encrypted)
}

// GetSecret retrieves and decrypts a secret
func (m *Manager) GetSecret(name, database string) (*Secret, error) {
	if database == "" {
		database = m.config.GetDefaultDatabase()
	}

	encrypted, err := m.storage.Get(database, name)
	if err != nil {
		return nil, err
	}

	data, err := m.encryptor.Decrypt(encrypted, database)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret: %w", err)
	}

	var secret Secret
	if err := json.Unmarshal(data, &secret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}

	return &secret, nil
}

// ListSecrets returns a list of all secrets in the database (without values)
func (m *Manager) ListSecrets(database string) ([]*SecretListItem, error) {
	if database == "" {
		database = m.config.GetDefaultDatabase()
	}

	keys, err := m.storage.List(database)
	if err != nil {
		return nil, err
	}

	var secrets []*SecretListItem
	for _, key := range keys {
		secret, err := m.GetSecret(key, database)
		if err != nil {
			// Skip corrupted secrets but continue listing others
			continue
		}

		secrets = append(secrets, secret.ToListItem())
	}

	return secrets, nil
}

// DeleteSecret removes a secret from the database
func (m *Manager) DeleteSecret(name, database string) error {
	if database == "" {
		database = m.config.GetDefaultDatabase()
	}

	return m.storage.Delete(database, name)
}

// ListDatabases returns all available databases
func (m *Manager) ListDatabases() ([]string, error) {
	return m.storage.ListDatabases()
}

// DeleteDatabase removes an entire database
func (m *Manager) DeleteDatabase(database string) error {
	return m.storage.DeleteDatabase(database)
}

// GetDefaultDatabase returns the default database name from config
func (c *Config) GetDefaultDatabase() string {
	if c.DefaultDatabase != "" {
		return c.DefaultDatabase
	}
	return "default"
}