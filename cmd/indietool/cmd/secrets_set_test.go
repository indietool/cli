package cmd

import (
	"strings"
	"testing"
)

func TestSecretSetStdinPreservesRawBytes(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)

	// Leading/trailing spaces and a trailing newline must survive untouched:
	// "echo foo |" pipelines are the classic silent-corruption trap.
	value := "  s3cret line1\nline2  \n"
	if _, err := runSecretsCmd(t, value, "secret", "set", "raw", "--stdin"); err != nil {
		t.Fatalf("secret set --stdin failed: %v", err)
	}

	got, err := newTestSecretsManager(t).GetSecret("raw", testDefaultDB)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.Value != value {
		t.Errorf("raw bytes not preserved:\nwant %q\ngot  %q", value, got.Value)
	}
}

func TestSecretSetStdinTrimFlag(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)

	if _, err := runSecretsCmd(t, "  hello \n", "secret", "set", "trimmed", "--stdin", "--trim"); err != nil {
		t.Fatalf("secret set --stdin --trim failed: %v", err)
	}

	got, err := newTestSecretsManager(t).GetSecret("trimmed", testDefaultDB)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.Value != "hello" {
		t.Errorf("expected %q, got %q", "hello", got.Value)
	}
}

func TestSecretSetStdinErrorsOnBothValueAndStdin(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)

	_, execErr := runSecretsCmd(t, "x", "secret", "set", "name", "argvalue", "--stdin")
	if execErr == nil {
		t.Fatal("expected error when both a positional value and --stdin are given")
	}
	if !strings.Contains(execErr.Error(), "not both") {
		t.Errorf("error should mention mutual exclusivity, got: %v", execErr)
	}
}

func TestSecretSetErrorsOnMissingValue(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)

	_, execErr := runSecretsCmd(t, "", "secret", "set", "name")
	if execErr == nil {
		t.Fatal("expected error when neither a value nor --stdin is given")
	}
	if !strings.Contains(execErr.Error(), "--stdin") {
		t.Errorf("error should point at --stdin, got: %v", execErr)
	}
}

func TestSecretSetStdinErrorsOnEmptyStdin(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)

	_, execErr := runSecretsCmd(t, "", "secret", "set", "empty", "--stdin")
	if execErr == nil {
		t.Fatal("expected error on empty stdin")
	}
	if !strings.Contains(execErr.Error(), "no secret value received") {
		t.Errorf("unexpected error text: %v", execErr)
	}
}

func TestSecretSetPositionalUnchanged(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)

	// Legacy form keeps its contract: the argument is stored untrimmed.
	if _, err := runSecretsCmd(t, "", "secret", "set", "legacy", "  spaced  "); err != nil {
		t.Fatalf("legacy set failed: %v", err)
	}

	got, err := newTestSecretsManager(t).GetSecret("legacy", testDefaultDB)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.Value != "  spaced  " {
		t.Errorf("legacy positional value changed: %q", got.Value)
	}
}
