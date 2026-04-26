package secrets

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"indietool/cli/indietool/shamir"
)

const (
	DefaultShards    = 5
	DefaultThreshold = 3
	DefaultPrefix    = "shard"
)

func EncryptWithPassphrase(data []byte, passphrase string) ([]byte, error) {
	rcpt, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to create recipient: %w", err)
	}
	out := &bytes.Buffer{}
	w, err := age.Encrypt(out, rcpt)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize encryption: %w", err)
	}
	return out.Bytes(), nil
}

func DecryptWithPassphrase(data []byte, passphrase string) ([]byte, error) {
	id, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}
	r, err := age.Decrypt(bytes.NewReader(data), id)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	out := &bytes.Buffer{}
	if _, err := io.Copy(out, r); err != nil {
		return nil, fmt.Errorf("failed to read decrypted data: %w", err)
	}
	return out.Bytes(), nil
}

func SplitShards(data []byte, n, threshold int) ([][]byte, error) {
	return shamir.Split(data, n, threshold)
}

func CombineShards(parts [][]byte) ([]byte, error) {
	return shamir.Combine(parts)
}

func WriteShards(shards [][]byte, prefix, outDir string) ([]string, error) {
	if err := os.MkdirAll(outDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", outDir, err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read output directory %s: %w", outDir, err)
	}
	if len(entries) > 0 {
		return nil, fmt.Errorf("output directory %s is not empty", outDir)
	}

	var written []string
	for idx, b := range shards {
		name := fmt.Sprintf("%s.%d", prefix, idx)
		path := filepath.Join(outDir, name)
		enc := base64.StdEncoding.EncodeToString(b)
		if err := os.WriteFile(path, []byte(enc), 0600); err != nil {
			for _, p := range written {
				os.Remove(filepath.Join(outDir, p))
			}
			return nil, fmt.Errorf("failed to write shard %s: %w", name, err)
		}
		written = append(written, name)
	}
	return written, nil
}

func ReadShards(dir string) ([][]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("directory %s is empty", dir)
	}

	var parts [][]byte
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read shard file %s: %w", entry.Name(), err)
		}
		dec, err := base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(string(data)))
		if err != nil {
			return nil, fmt.Errorf("failed to decode shard file %s: %w", entry.Name(), err)
		}
		parts = append(parts, dec)
	}
	return parts, nil
}

func ReadPassphrase() string {
	return os.Getenv("INDIETOOL_SECRET_PASSWORD")
}
