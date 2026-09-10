package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"

	"github.com/indietool/cli/indietool/secrets"
	"github.com/spf13/cobra"
)

// exitError carries a child process exit status through RunE so Execute() can
// exit with the same code (os.Exit inside RunE would skip cobra's error path).
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("child process exited with status %d", e.code)
}

var secretsExecCmd = &cobra.Command{
	Use:   "exec [@database] -- <command> [args...]",
	Short: "Run a command with a database's secrets injected as environment variables",
	Long: `Run a command with all secrets from a secrets database injected as
environment variables (variable name = secret name).

Use "--" to separate the command from indietool flags. Without an @database
argument the configured default database is used.

Collision policy: a secret whose name already exists in the parent environment
is refused by default; pass --force to let database values win. Names that are
not valid environment variables are skipped with a warning. Secret values are
passed to the child via its environment only — they never appear in argv.

Examples:
  indietool secret exec @hermes -- ./deploy.sh
  indietool secret exec -- env
  indietool secret exec @production --force -- python3 sync.py`,
	Args: cobra.MinimumNArgs(1),
	RunE: execSecrets,
}

type secretExecJSON struct {
	Status       string   `json:"status"`
	Database     string   `json:"database"`
	Injected     []string `json:"injected"`
	Skipped      []string `json:"skipped,omitempty"`
	Conflicts    []string `json:"conflicts,omitempty"`
	ForceApplied []string `json:"force_applied,omitempty"`
	Command      []string `json:"command"`
	ExitCode     *int     `json:"exit_code,omitempty"`
	Signal       string   `json:"signal,omitempty"`
}

func init() {
	secretsCmd.AddCommand(secretsExecCmd)
	secretsExecCmd.Flags().Bool("force", false, "Let database values overwrite variables already present in the parent environment")
}

func execSecrets(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	var database string
	// A leading "@db" argument selects the database; the rest is the command.
	if strings.HasPrefix(args[0], "@") {
		database = strings.TrimPrefix(args[0], "@")
		args = args[1:]
	}
	if len(args) == 0 {
		return fmt.Errorf("no command given: use 'secret exec [@database] -- <command> [args...]'")
	}

	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration available")
	}
	secretsConfig := cfg.GetSecretsConfig()
	if database == "" {
		database = secretsConfig.GetDefaultDatabase()
	}

	manager, err := secrets.NewManager(secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}

	items, err := manager.ListSecrets(database)
	if err != nil {
		return fmt.Errorf("failed to list secrets in database '%s': %w", database, err)
	}

	// Deterministic injection order; collisions detected against the PARENT
	// environment (os.Environ), not against earlier injections.
	names := make([]string, 0, len(items))
	for _, it := range items {
		names = append(names, it.Name)
	}
	sort.Strings(names)

	var injected, conflicts, forceApplied []string
	var skipped []string
	// ListSecrets carries no values by design; decrypt each secret.
	secretList := make([]*secrets.Secret, 0, len(names))
	for _, name := range names {
		sc, err := manager.GetSecret(name, database)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ skipping %q: %v\n", name, err)
			skipped = append(skipped, name)
			continue
		}
		secretList = append(secretList, sc)
	}

	childEnv := os.Environ()
	parentVars := make(map[string]struct{}, len(childEnv))
	for _, kv := range childEnv {
		if i := strings.IndexByte(kv, '='); i > 0 {
			parentVars[kv[:i]] = struct{}{}
		}
	}

	for _, s := range secretList {
		name := s.Name
		if !isValidEnvVarName(name) {
			fmt.Fprintf(os.Stderr, "⚠ skipping %q: not a valid environment variable name\n", name)
			skipped = append(skipped, name)
			continue
		}
		if _, exists := parentVars[name]; exists {
			if !force {
				conflicts = append(conflicts, name)
				continue
			}
			forceApplied = append(forceApplied, name)
		}
		childEnv = append(childEnv, name+"="+s.Value)
		injected = append(injected, name)
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("refusing to overwrite %d environment variable(s) already set: %s (use --force to override)",
			len(conflicts), strings.Join(conflicts, ", "))
	}

	// Names go to stderr in human mode: the child owns stdout.
	if !jsonOutput && len(injected) > 0 {
		fmt.Fprintf(os.Stderr, "✓ Injected %d secret(s) from @%s: %s\n",
			len(injected), database, strings.Join(injected, ", "))
	}

	child := exec.Command(args[0], args[1:]...) // #nosec G204 — deliberate exec of user-provided command
	child.Env = childEnv
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	exitCode := 0
	signalName := ""
	if err := child.Wait(); err != nil {
		var ws *exec.ExitError
		if errors.As(err, &ws) {
			if status, ok := ws.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				signalName = status.Signal().String()
				exitCode = 128 + int(status.Signal())
			} else {
				exitCode = ws.ExitCode()
			}
		} else {
			return fmt.Errorf("failed to run command: %w", err)
		}
	}

	if jsonOutput {
		report := secretExecJSON{
			Status:       "success",
			Database:     database,
			Injected:     injected,
			Skipped:      skipped,
			Conflicts:    conflicts,
			ForceApplied: forceApplied,
			Command:      args,
		}
		if exitCode != 0 {
			report.ExitCode = &exitCode
			report.Signal = signalName
		}
		if err := printJSON(report); err != nil {
			return err
		}
	}

	if exitCode != 0 {
		return &exitError{code: exitCode}
	}
	return nil
}

// isValidEnvVarName mirrors POSIX: [A-Za-z_][A-Za-z0-9_]*
func isValidEnvVarName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r == '_':
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
