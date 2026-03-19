package secrets

import (
	"fmt"
	"time"
)

// ExportData is the portable JSON format for secrets export/import.
type ExportData struct {
	Version    int                  `json:"version"`
	ExportedAt time.Time            `json:"exported_at"`
	Databases  map[string][]*Secret `json:"databases"`
}

// ImportConflict records a secret that already existed during import.
type ImportConflict struct {
	Database string
	Name     string
}

const exportFormatVersion = 1

// ExportSecrets collects plaintext secrets according to spec.
//
// spec maps a database name to the set of secret names to export from it.
// A nil slice means "export all secrets from that database".
// Every key in spec must appear in the output, even if it has zero secrets.
func (m *Manager) ExportSecrets(spec map[string][]string) (*ExportData, error) {
	data := &ExportData{
		Version:    exportFormatVersion,
		ExportedAt: time.Now().UTC(),
		Databases:  make(map[string][]*Secret, len(spec)),
	}

	for db, names := range spec {
		if names == nil {
			// Export the whole database
			keys, err := m.storage.List(db)
			if err != nil {
				return nil, fmt.Errorf("failed to list secrets in database %q: %w", db, err)
			}
			names = keys
		}

		dbSecrets := make([]*Secret, 0, len(names))
		for _, name := range names {
			secret, err := m.GetSecret(name, db)
			if err != nil {
				return nil, fmt.Errorf("failed to read secret %q from database %q: %w", name, db, err)
			}
			dbSecrets = append(dbSecrets, secret)
		}
		data.Databases[db] = dbSecrets
	}

	return data, nil
}

// ImportSecrets imports secrets from ExportData into the local instance.
// Returns the count of imported secrets and any conflicts encountered.
// If force is false, halts on the first conflict before writing anything.
func (m *Manager) ImportSecrets(data *ExportData, force bool) (imported int, conflicts []ImportConflict, err error) {
	if data.Version != exportFormatVersion {
		return 0, nil, fmt.Errorf("unsupported export format version %d (expected %d)", data.Version, exportFormatVersion)
	}

	// Pre-scan for conflicts so we fail before writing anything when !force
	if !force {
		for db, secretsList := range data.Databases {
			for _, secret := range secretsList {
				if _, err := m.GetSecret(secret.Name, db); err == nil {
					return 0, []ImportConflict{{Database: db, Name: secret.Name}},
						fmt.Errorf("secret %q already exists in database %q; use --force to overwrite", secret.Name, db)
				}
			}
		}
	}

	for db, secretsList := range data.Databases {
		for _, secret := range secretsList {
			if _, getErr := m.GetSecret(secret.Name, db); getErr == nil {
				conflicts = append(conflicts, ImportConflict{Database: db, Name: secret.Name})
			}
			if err := m.SetSecret(secret.Name, secret.Value, db, secret.Note, secret.ExpiresAt); err != nil {
				return imported, conflicts, fmt.Errorf("failed to import secret %q into database %q: %w", secret.Name, db, err)
			}
			imported++
		}
	}

	return imported, conflicts, nil
}
