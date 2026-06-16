package tools_test

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/tools"
)

type fakeRunner struct {
	gotName string
	gotArgs []string
	code    int
	err     error
}

func (f *fakeRunner) Run(name string, args []string, stdout, stderr io.Writer) (int, error) {
	f.gotName = name
	f.gotArgs = args
	return f.code, f.err
}

func TestInstallRunsInstallCmdThroughShell(t *testing.T) {
	tool := tools.Tool{Name: "x", InstallCmd: "npm install -g @foo/bar@latest"}
	r := &fakeRunner{code: 0}
	code, err := tools.Install(tool, r, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Install err = %v", err)
	}
	if code != 0 {
		t.Fatalf("Install code = %d", code)
	}
	wantShell, wantShellFlag := "/bin/sh", "-c"
	if runtime.GOOS == "windows" {
		wantShell, wantShellFlag = "cmd", "/C"
	}
	if r.gotName != wantShell || len(r.gotArgs) != 2 || r.gotArgs[0] != wantShellFlag {
		t.Fatalf("invocation = %s %v", r.gotName, r.gotArgs)
	}
	if r.gotArgs[1] != "npm install -g @foo/bar@latest" {
		t.Fatalf("install cmd = %q", r.gotArgs[1])
	}
}

func TestInstallNoCommandSignalsSentinel(t *testing.T) {
	if _, err := tools.Install(tools.Tool{}, &fakeRunner{}, io.Discard, io.Discard); !errors.Is(err, tools.ErrNoInstallCommand) {
		t.Fatalf("err = %v, want ErrNoInstallCommand", err)
	}
}

func TestUninstallNpmInvokesNpmUninstall(t *testing.T) {
	r := &fakeRunner{}
	_, msg, err := tools.Uninstall(tools.Tool{InstallCmd: "npm install -g @foo/bar@latest"}, r, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Uninstall err = %v", err)
	}
	if msg != "npm uninstall -g @foo/bar" {
		t.Fatalf("msg = %q", msg)
	}
	wantShell, _ := "/bin/sh", "-c"
	if runtime.GOOS == "windows" {
		wantShell = "cmd"
	}
	if r.gotName != wantShell || r.gotArgs[1] != "npm uninstall -g @foo/bar" {
		t.Fatalf("invocation = %s %v", r.gotName, r.gotArgs)
	}
}

func TestUninstallNonNpmBinNotFoundIsBenign(t *testing.T) {
	tool := tools.Tool{Name: "ghost", CLICommand: "definitely-not-real-binary-xyz", InstallCmd: "curl ..."}
	_, msg, err := tools.Uninstall(tool, &fakeRunner{}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if msg == "" {
		t.Fatal("expected message")
	}
}

func testShell(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "/bin/sh", []string{"-c", command}
}

func TestShellRunnerStreamsTrueExit(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	successCommand := "true"
	if runtime.GOOS == "windows" {
		successCommand = "exit 0"
	}
	name, args := testShell(successCommand)
	code, err := tools.ShellRunner{}.Run(name, args, stdout, stderr)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
}

func TestShellRunnerCapturesExitCode(t *testing.T) {
	name, args := testShell("exit 7")
	code, err := tools.ShellRunner{}.Run(name, args, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if code != 7 {
		t.Fatalf("code = %d", code)
	}
}
