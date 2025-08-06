package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

// ErrSecretDBNotFound is returned when the secrets database does not exist
var ErrSecretDBNotFound = errors.New("secrets database not found")

// Storage handles persistent storage of encrypted secrets using BadgerDB
type Storage struct {
	config  *Config
	baseDir string
}

// NewStorage creates a new storage instance
func NewStorage(config *Config) (*Storage, error) {
	baseDir := config.GetSecretsDir()
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create secrets directory: %w", err)
	}

	return &Storage{
		config:  config,
		baseDir: baseDir,
	}, nil
}

// getDBPath returns the path to the database directory for the specified database
func (s *Storage) getDBPath(database string) string {
	return filepath.Join(s.baseDir, database)
}

// openDB opens a BadgerDB instance for the specified database
func (s *Storage) openDB(database string, readonly bool) (*badger.DB, error) {
	dbPath := s.getDBPath(database)

	// Only check if database directory exists when opening in read-only mode
	// For write mode, BadgerDB will automatically create the directory
	if readonly {
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrSecretDBNotFound, database)
		}
	}

	opts := badger.DefaultOptions(dbPath)
	opts.ReadOnly = readonly
	opts.Logger = nil // Disable badger logging to keep output clean

	return badger.Open(opts)
}

// Set stores an encrypted value for the given key in the specified database
func (s *Storage) Set(database, key string, value []byte) error {
	db, err := s.openDB(database, false)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	return db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

// Get retrieves an encrypted value for the given key from the specified database
func (s *Storage) Get(database, key string) ([]byte, error) {
	db, err := s.openDB(database, true)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var value []byte
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		value, err = item.ValueCopy(nil)
		return err
	})

	if err == badger.ErrKeyNotFound {
		return nil, fmt.Errorf("secret '%s' not found in database '%s'", key, database)
	}

	return value, err
}

// List returns all keys in the specified database
func (s *Storage) List(database string) ([]string, error) {
	db, err := s.openDB(database, true)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var keys []string
	err = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			keys = append(keys, key)
		}
		return nil
	})

	return keys, err
}

// Delete removes a key from the specified database
func (s *Storage) Delete(database, key string) error {
	db, err := s.openDB(database, false)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	return db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// ListDatabases returns all available database names
func (s *Storage) ListDatabases() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // No databases exist yet
		}
		return nil, fmt.Errorf("failed to read secrets directory: %w", err)
	}

	var databases []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			databases = append(databases, entry.Name())
		}
	}

	return databases, nil
}

// DeleteDatabase removes an entire database
func (s *Storage) DeleteDatabase(database string) error {
	dbPath := s.getDBPath(database)
	return os.RemoveAll(dbPath)
}

// GetSecretsDir returns the directory where secrets are stored
func (c *Config) GetSecretsDir() string {
	if c.StorageDir == "" {
		return ""
	}

	return expandPath(c.StorageDir)
}

// expandPath expands ~ in paths to the user's home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
