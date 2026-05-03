package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNewShellInitCommand_Bash(t *testing.T) {
	var buf bytes.Buffer
	cmd := NewShellInitCommand()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"bash"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	assertContains(t, out, "KUBERT_SHELL_INIT=1")
	assertContains(t, out, "KUBERT_SHELL_INIT_SHELL=bash")
	assertContains(t, out, "command kubert")
	assertContains(t, out, "eval \"$(kubert shell-init bash)\"")
}

func TestNewShellInitCommand_Zsh(t *testing.T) {
	var buf bytes.Buffer
	cmd := NewShellInitCommand()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"zsh"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	assertContains(t, out, "KUBERT_SHELL_INIT=1")
	assertContains(t, out, "KUBERT_SHELL_INIT_SHELL=zsh")
	assertContains(t, out, "eval \"$(kubert shell-init zsh)\"")
}

func TestNewShellInitCommand_Fish(t *testing.T) {
	var buf bytes.Buffer
	cmd := NewShellInitCommand()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"fish"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	assertContains(t, out, "set -gx KUBERT_SHELL_INIT 1")
	assertContains(t, out, "set -gx KUBERT_SHELL_INIT_SHELL fish")
	assertContains(t, out, "kubert shell-init fish | source")
	assertContains(t, out, "fish_pid")
}

func TestNewShellInitCommand_AutoDetect(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	var buf bytes.Buffer
	cmd := NewShellInitCommand()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, buf.String(), "KUBERT_SHELL_INIT_SHELL=zsh")
}

func TestNewShellInitCommand_InvalidShell(t *testing.T) {
	cmd := NewShellInitCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"powershell"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unsupported shell, got nil")
	}
}

func TestWriteEnvUpdateFile_Bash(t *testing.T) {
	t.Setenv("KUBERT_SHELL_INIT_SHELL", "bash")

	if err := writeEnvUpdateFile("my-cluster", "/home/user/.kube/config"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := envUpdateFilePath(os.Getppid())
	t.Cleanup(func() { _ = os.Remove(path) })

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("env update file not created: %v", err)
	}
	content := string(data)
	assertContains(t, content, "export KUBERT_SHELL_CONTEXT='my-cluster'")
	assertContains(t, content, "export KUBERT_SHELL_ORIGINAL_KUBECONFIG='/home/user/.kube/config'")
}

func TestWriteEnvUpdateFile_Fish(t *testing.T) {
	t.Setenv("KUBERT_SHELL_INIT_SHELL", "fish")

	if err := writeEnvUpdateFile("my-cluster", "/home/user/.kube/config"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := envUpdateFilePath(os.Getppid())
	t.Cleanup(func() { _ = os.Remove(path) })

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("env update file not created: %v", err)
	}
	content := string(data)
	assertContains(t, content, "set -gx KUBERT_SHELL_CONTEXT")
	assertContains(t, content, "set -gx KUBERT_SHELL_ORIGINAL_KUBECONFIG")
}

func TestShellSingleQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{"simple", "'simple'"},
		{"it's", `'it'\''s'`},
		{"", "''"},
	}
	for _, c := range cases {
		if got := shellSingleQuote(c.in); got != c.want {
			t.Errorf("shellSingleQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\ngot:\n%s", needle, haystack)
	}
}
