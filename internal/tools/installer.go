package tools

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/google/shlex"
)

// Installer executes the install_cmd for a tool.  The CommandRunner indirection
// keeps the implementation easy to unit-test by providing a fake runner.
type Installer struct {
	Runner CommandRunner
}

// CommandRunner is the seam tests use to capture commands without spawning
// processes.
type CommandRunner interface {
	Run(name string, args []string, stdout, stderr io.Writer) (int, error)
}

// ShellRunner is the default CommandRunner; it executes via /bin/sh -c when the
// install_cmd contains shell metacharacters and via exec.Command otherwise.
type ShellRunner struct{}

// Run executes name with args.  If name is empty, it falls back to shell mode
// using the first arg as the full command.
func (ShellRunner) Run(name string, args []string, stdout, stderr io.Writer) (int, error) {
	if name == "" {
		return 1, errors.New("tools: empty command")
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

// Install runs the install_cmd for the given tool, streaming output through w.
// Empty install_cmd returns nil with a "no install command" sentinel.
var ErrNoInstallCommand = errors.New("tools: tool has no install_cmd")

// Install installs the tool by executing its install_cmd through the shell.
func Install(tool Tool, runner CommandRunner, stdout, stderr io.Writer) (int, error) {
	if runner == nil {
		runner = ShellRunner{}
	}
	if strings.TrimSpace(tool.InstallCmd) == "" {
		return 0, ErrNoInstallCommand
	}
	// Many install_cmd lines pipe to bash, so always run via /bin/sh -c.
	return runner.Run("/bin/sh", []string{"-c", tool.InstallCmd}, stdout, stderr)
}

// Uninstall best-effort removes the tool.  npm-installed tools are removed via
// `npm uninstall -g <pkg>`; for other install styles we fall back to deleting
// the binary discovered on PATH.  The returned message is suitable for stdout.
func Uninstall(tool Tool, runner CommandRunner, stdout, stderr io.Writer) (int, string, error) {
	if runner == nil {
		runner = ShellRunner{}
	}
	pkg, isNPM := npmPackage(tool.InstallCmd)
	if isNPM {
		code, err := runner.Run("/bin/sh", []string{"-c", fmt.Sprintf("npm uninstall -g %s", pkg)}, stdout, stderr)
		return code, fmt.Sprintf("npm uninstall -g %s", pkg), err
	}
	bin := tool.LaunchCommand()
	if bin == "" {
		return 0, "no binary to remove", nil
	}
	path, err := exec.LookPath(bin)
	if err != nil {
		return 0, fmt.Sprintf("%s not found on PATH", bin), nil
	}
	if rmErr := os.Remove(path); rmErr != nil {
		return 1, fmt.Sprintf("rm %s", path), rmErr
	}
	return 0, fmt.Sprintf("removed %s", path), nil
}

// npmPackage extracts the npm package name from a shell install_cmd like
// `npm install -g @scope/name@latest`.  Returns false when the command is not
// an npm install.
func npmPackage(installCmd string) (string, bool) {
	cmd := strings.TrimSpace(installCmd)
	if !strings.HasPrefix(cmd, "npm ") {
		return "", false
	}
	parts, err := shlex.Split(cmd)
	if err != nil {
		return "", false
	}
	skip := map[string]bool{"npm": true, "install": true, "i": true, "-g": true, "--global": true}
	for _, p := range parts {
		if skip[p] {
			continue
		}
		// Strip version suffix after @, preserving @scope/name.
		if strings.HasPrefix(p, "@") {
			idx := strings.Index(p[1:], "@")
			if idx >= 0 {
				return p[:idx+1], true
			}
			return p, true
		}
		if idx := strings.Index(p, "@"); idx > 0 {
			return p[:idx], true
		}
		return p, true
	}
	return "", false
}

// IsInstalled reports whether the tool's CLI binary is reachable via PATH.
func IsInstalled(tool Tool) bool {
	bin := tool.LaunchCommand()
	if bin == "" {
		return false
	}
	_, err := exec.LookPath(bin)
	return err == nil
}

// DetectVersion returns the tool's version string by running `<bin> --version`.
// Returns "unknown" when the binary is missing or every probe fails.
func DetectVersion(tool Tool) string {
	bin := tool.LaunchCommand()
	if bin == "" {
		return "unknown"
	}
	for _, flag := range []string{"--version", "-v", "version"} {
		out, err := exec.Command(bin, flag).CombinedOutput()
		if err != nil {
			continue
		}
		s := strings.TrimSpace(string(out))
		if s == "" {
			continue
		}
		// Take the last non-empty line.
		lines := strings.Split(s, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line != "" {
				return line
			}
		}
	}
	return "unknown"
}
