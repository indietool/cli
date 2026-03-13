package secrets

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// KeyBackend is the interface for key storage backends
type KeyBackend interface {
	HasKey(database string) bool
	GetKey(database string) (string, error)
	SetKey(database string, key string) error
}

const keyringService = "indietool-secrets"

// KeyringBackend stores keys in the system keyring
type KeyringBackend struct{}

func (b *KeyringBackend) HasKey(db string) bool {
	keyName := fmt.Sprintf("db-key-%s", db)
	_, err := keyring.Get(keyringService, keyName)
	return err == nil
}

func (b *KeyringBackend) GetKey(db string) (string, error) {
	keyName := fmt.Sprintf("db-key-%s", db)
	key, err := keyring.Get(keyringService, keyName)
	if err != nil {
		return "", fmt.Errorf("encryption key not found for database '%s'", db)
	}
	return key, nil
}

func (b *KeyringBackend) SetKey(db, key string) error {
	keyName := fmt.Sprintf("db-key-%s", db)
	return keyring.Set(keyringService, keyName, key)
}

// AgeSSHBackend stores keys as age-encrypted files using an SSH public key
type AgeSSHBackend struct {
	config *Config
}

func (b *AgeSSHBackend) keyFilePath(db string) string {
	storageDir := expandPath(b.config.StorageDir)
	keysDir := filepath.Join(filepath.Dir(storageDir), "keys")
	return filepath.Join(keysDir, fmt.Sprintf("db-key-%s.age", db))
}

func (b *AgeSSHBackend) HasKey(db string) bool {
	_, err := os.Stat(b.keyFilePath(db))
	return err == nil
}

func (b *AgeSSHBackend) GetKey(db string) (string, error) {
	keyFile := b.keyFilePath(db)
	encData, err := os.ReadFile(keyFile)
	if err != nil {
		return "", fmt.Errorf("failed to read key file: %w", err)
	}

	privKeyPath := expandPath(b.config.SSHPrivateKeyPath)
	pemBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SSH private key: %w", err)
	}

	identity, err := agessh.ParseIdentity(pemBytes)
	if err != nil {
		var missingErr *ssh.PassphraseMissingError
		if errors.As(err, &missingErr) {
			identity, err = agessh.NewEncryptedSSHIdentity(missingErr.PublicKey, pemBytes, func() ([]byte, error) {
				fmt.Fprintf(os.Stderr, "Enter passphrase for %s: ", privKeyPath)
				pass, passErr := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(os.Stderr)
				return pass, passErr
			})
			if err != nil {
				return "", fmt.Errorf("failed to create SSH identity: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to parse SSH private key: %w", err)
		}
	}

	r, err := age.Decrypt(bytes.NewReader(encData), identity)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key file: %w", err)
	}

	keyData, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read decrypted key: %w", err)
	}

	return string(keyData), nil
}

func (b *AgeSSHBackend) SetKey(db, key string) error {
	pubKeyPath := expandPath(b.config.SSHPublicKeyPath)
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH public key: %w", err)
	}

	recipient, err := agessh.ParseRecipient(string(pubKeyBytes))
	if err != nil {
		return fmt.Errorf("failed to parse SSH public key: %w", err)
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return fmt.Errorf("failed to create age encryptor: %w", err)
	}

	if _, err := io.WriteString(w, key); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to finalize age encryption: %w", err)
	}

	keyFile := b.keyFilePath(db)
	if err := os.MkdirAll(filepath.Dir(keyFile), 0700); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	return os.WriteFile(keyFile, buf.Bytes(), 0600)
}
