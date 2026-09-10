package cmd

import (
	"encoding/json"
	"errors"
	"strings"
	"syscall"
	"testing"
)

func execShellArgs(t *testing.T, script string) []string {
	t.Helper()
	return []string{"secret", "exec", "--", "sh", "-c", script}
}

func TestSecretExecInjectsEnvVars(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)
	mgr := newTestSecretsManager(t)
	mustSetSecret(t, mgr, "INJECTED_A", "alpha")
	mustSetSecret(t, mgr, "INJECTED_B", "beta")

	restore := captureChildStdout(t)
	_, execErr := runSecretsCmd(t, "", execShellArgs(t, `printf '%s|%s' "$INJECTED_A" "$INJECTED_B"`)...)
	out := restore()

	if execErr != nil {
		t.Fatalf("secret exec failed: %v", execErr)
	}
	if out != "alpha|beta" {
		t.Errorf("child saw %q, want %q", out, "alpha|beta")
	}
}

func TestSecretExecRefusesParentEnvCollision(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)
	mgr := newTestSecretsManager(t)
	mustSetSecret(t, mgr, "INJECTED_EXISTING", "fromDB")

	t.Setenv("INJECTED_EXISTING", "fromParent")

	_, execErr := runSecretsCmd(t, "", execShellArgs(t, `exit 0`)...)
	if execErr == nil {
		t.Fatal("expected refusal when a secret collides with a parent env var")
	}
	if !strings.Contains(execErr.Error(), "--force") {
		t.Errorf("refusal should mention --force, got: %v", execErr)
	}
}

func TestSecretExecForceOverwritesParentEnv(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)
	mgr := newTestSecretsManager(t)
	mustSetSecret(t, mgr, "INJECTED_EXISTING", "fromDB")

	t.Setenv("INJECTED_EXISTING", "fromParent")

	restore := captureChildStdout(t)
	_, execErr := runSecretsCmd(t, "", "secret", "exec", "--force", "--", "sh", "-c", `printf '%s' "$INJECTED_EXISTING"`)
	out := restore()

	if execErr != nil {
		t.Fatalf("secret exec --force failed: %v", execErr)
	}
	if out != "fromDB" {
		t.Errorf("child saw %q, want the database value", out)
	}
}

func TestSecretExecSkipsInvalidNames(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)
	mgr := newTestSecretsManager(t)
	mustSetSecret(t, mgr, "INJECTED_GOOD", "yes")
	mustSetSecret(t, mgr, "not-env", "no")

	restore := captureChildStdout(t)
	_, execErr := runSecretsCmd(t, "", execShellArgs(t, `printf '%s' "$INJECTED_GOOD"`)...)
	out := restore()

	if execErr != nil {
		t.Fatalf("secret exec failed: %v", execErr)
	}
	if out != "yes" {
		t.Errorf("valid secret not injected: %q", out)
	}
}

func TestSecretExecJSONReportAndExitCode(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)
	mgr := newTestSecretsManager(t)
	mustSetSecret(t, mgr, "INJECTED_JSON", "val")

	restore := captureChildStdout(t)
	_, execErr := runSecretsCmd(t, "", "secret", "exec", "--json", "--", "sh", "-c", "exit 3")
	out := restore()

	var ee *exitError
	if !errors.As(execErr, &ee) || ee.code != 3 {
		t.Fatalf("expected child exit code 3 via exitError, got: %v", execErr)
	}

	var report secretExecJSON
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("report is not JSON: %v\nraw: %q", err, out)
	}
	if len(report.Injected) != 1 || report.Injected[0] != "INJECTED_JSON" {
		t.Errorf("report.Injected = %v", report.Injected)
	}
	if report.ExitCode == nil || *report.ExitCode != 3 {
		t.Errorf("report.ExitCode = %v, want 3", report.ExitCode)
	}
}

func TestSecretExecSignalExitCode(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)
	mgr := newTestSecretsManager(t)
	mustSetSecret(t, mgr, "INJECTED_SIG", "v")

	_, execErr := runSecretsCmd(t, "", execShellArgs(t, `kill -TERM $$`)...)
	var ee *exitError
	if !errors.As(execErr, &ee) {
		t.Fatalf("expected exitError, got: %v", execErr)
	}
	if want := 128 + int(syscall.SIGTERM); ee.code != want {
		t.Errorf("signal exit code = %d, want %d", ee.code, want)
	}
}

func TestSecretExecExplicitDatabaseArg(t *testing.T) {
	configPath := writeIsolatedSecretsConfig(t)
	useTestConfig(t, configPath)
	mgr := newTestSecretsManager(t)
	mustSetSecret(t, mgr, "INJECTED_OTHER", "otherval")

	// A different database must NOT leak the default database's secrets.
	if err := mgr.SetSecret("INJECTED_A", "alpha", "otherdb", "", nil); err != nil {
		t.Fatalf("seed otherdb failed: %v", err)
	}

	restore := captureChildStdout(t)
	_, execErr := runSecretsCmd(t, "", "secret", "exec", "@otherdb", "--", "sh", "-c", `printf '%s:%s' "$INJECTED_A" "$(env | grep -c INJECTED_OTHER)"`)
	out := restore()

	if execErr != nil {
		t.Fatalf("secret exec @otherdb failed: %v", execErr)
	}
	// INJECTED_A comes from @otherdb; INJECTED_OTHER (default db) must NOT leak.
	if out != "alpha:0" {
		t.Errorf("child saw %q, want alpha:0", out)
	}
}
