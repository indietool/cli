package secrets

import (
	"bytes"
	"fmt"
	"os"

	"filippo.io/age"
	"github.com/zalando/go-keyring"
)

const (
	KeyringService = "indietool-secrets"
)

// Encryptor handles encryption and decryption of secrets using age
type Encryptor struct{}

// NewEncryptor creates a new encryptor instance
func NewEncryptor() (*Encryptor, error) {
	return &Encryptor{}, nil
}

// InitializeKey initializes or loads an encryption key for the specified database
func (e *Encryptor) InitializeKey(database, keyPath string) error {
	var identity *age.X25519Identity
	var err error

	if keyPath != "" {
		// Load from specified path
		keyData, readErr := os.ReadFile(keyPath)
		if readErr != nil {
			return fmt.Errorf("failed to read key file: %w", readErr)
		}

		identity, err = age.ParseX25519Identity(string(keyData))
		if err != nil {
			return fmt.Errorf("failed to parse key: %w", err)
		}
	} else {
		// Generate new key
		identity, err = age.GenerateX25519Identity()
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}
	}

	// Store in keyring using zalando/go-keyring
	keyName := fmt.Sprintf("db-key-%s", database)
	err = keyring.Set(KeyringService, keyName, identity.String())
	if err != nil {
		return fmt.Errorf("failed to store key in keyring: %w", err)
	}

	return nil
}

// getIdentity retrieves the encryption identity for the specified database
func (e *Encryptor) getIdentity(database string) (*age.X25519Identity, error) {
	keyName := fmt.Sprintf("db-key-%s", database)
	keyData, err := keyring.Get(KeyringService, keyName)
	if err != nil {
		return nil, fmt.Errorf("encryption key not found for database '%s': run 'indietool secrets init' first", database)
	}

	identity, err := age.ParseX25519Identity(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse stored key: %w", err)
	}

	return identity, nil
}

// Encrypt encrypts data using the key for the specified database
func (e *Encryptor) Encrypt(data []byte, database string) ([]byte, error) {
	identity, err := e.getIdentity(database)
	if err != nil {
		return nil, err
	}

	var encrypted bytes.Buffer
	w, err := age.Encrypt(&encrypted, identity.Recipient())
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write data: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encryptor: %w", err)
	}

	return encrypted.Bytes(), nil
}

// Decrypt decrypts data using the key for the specified database
func (e *Encryptor) Decrypt(data []byte, database string) ([]byte, error) {
	identity, err := e.getIdentity(database)
	if err != nil {
		return nil, err
	}

	r, err := age.Decrypt(bytes.NewReader(data), identity)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	decrypted := new(bytes.Buffer)
	if _, err := decrypted.ReadFrom(r); err != nil {
		return nil, fmt.Errorf("failed to read decrypted data: %w", err)
	}

	return decrypted.Bytes(), nil
}