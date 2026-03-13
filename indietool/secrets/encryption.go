package secrets

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
)

// ErrKeyringUnavailable is returned by InitializeKey (auto mode) when the system
// keyring cannot be accessed. It carries enough context for the caller to present
// an SSH key selection UI and then retry with the chosen key.
type ErrKeyringUnavailable struct {
	Cause         error    // underlying keyring error
	DefaultPubKey string   // configured SSH pub key path (may use ~)
	DefaultExists bool     // whether DefaultPubKey exists on disk
	AvailableKeys []string // .pub files found in ~/.ssh/ (tilde-prefixed)
}

func (e *ErrKeyringUnavailable) Error() string {
	return fmt.Sprintf("system keyring unavailable: %v", e.Cause)
}

func (e *ErrKeyringUnavailable) Unwrap() error { return e.Cause }

// Encryptor handles encryption and decryption of secrets using age
type Encryptor struct {
	config *Config
}

// NewEncryptor creates a new encryptor instance
func NewEncryptor(config *Config) (*Encryptor, error) {
	return &Encryptor{config: config}, nil
}

// getBackends returns the backends to use based on config
func (e *Encryptor) getBackends() []KeyBackend {
	switch e.config.KeyBackend {
	case "keyring":
		return []KeyBackend{&KeyringBackend{}}
	case "age-ssh":
		return []KeyBackend{&AgeSSHBackend{config: e.config}}
	default:
		return []KeyBackend{&KeyringBackend{}, &AgeSSHBackend{config: e.config}}
	}
}

// HasKey checks if an encryption key already exists for the specified database
func (e *Encryptor) HasKey(database string) bool {
	for _, b := range e.getBackends() {
		if b.HasKey(database) {
			return true
		}
	}
	return false
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

	keyStr := identity.String()

	switch e.config.KeyBackend {
	case "keyring":
		if err := (&KeyringBackend{}).SetKey(database, keyStr); err != nil {
			return fmt.Errorf("failed to store key in keyring: %w", err)
		}
	case "age-ssh":
		if err := (&AgeSSHBackend{config: e.config}).SetKey(database, keyStr); err != nil {
			return fmt.Errorf("failed to store key via age-ssh backend: %w", err)
		}
	default:
		// auto: try keyring first; if unavailable, surface a typed error so the
		// caller (cmd layer) can present a backend-selection UI and retry.
		if keyringErr := (&KeyringBackend{}).SetKey(database, keyStr); keyringErr != nil {
			defaultPub := e.config.SSHPublicKeyPath
			_, statErr := os.Stat(expandPath(defaultPub))
			defaultExists := statErr == nil

			homeDir, _ := os.UserHomeDir()
			var availableKeys []string
			if entries, err := os.ReadDir(filepath.Join(homeDir, ".ssh")); err == nil {
				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pub") {
						availableKeys = append(availableKeys, "~/.ssh/"+entry.Name())
					}
				}
			}

			return &ErrKeyringUnavailable{
				Cause:         keyringErr,
				DefaultPubKey: defaultPub,
				DefaultExists: defaultExists,
				AvailableKeys: availableKeys,
			}
		}
	}

	return nil
}

// getIdentity retrieves the encryption identity for the specified database
func (e *Encryptor) getIdentity(database string) (*age.X25519Identity, error) {
	var lastErr error
	for _, b := range e.getBackends() {
		if b.HasKey(database) {
			keyStr, err := b.GetKey(database)
			if err != nil {
				lastErr = err
				continue
			}
			identity, err := age.ParseX25519Identity(keyStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse stored key: %w", err)
			}
			return identity, nil
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("encryption key not found for database '%s'", database)
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
