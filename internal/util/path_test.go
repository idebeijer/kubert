package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPathHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	got, err := ExpandPath("~")
	if err != nil {
		t.Fatal(err)
	}
	if got != home {
		t.Fatalf("ExpandPath(~) = %q, want %q", got, home)
	}

	got, err = ExpandPath("~/.kube/config")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".kube", "config")
	if got != want {
		t.Fatalf("ExpandPath(~/.kube/config) = %q, want %q", got, want)
	}
}

func TestExpandPathLeavesOtherUserTilde(t *testing.T) {
	in := "~otheruser/.kube/config"
	got, err := ExpandPath(in)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Fatalf("ExpandPath(%q) = %q, want unchanged", in, got)
	}
}

func TestExpandPathPlain(t *testing.T) {
	got, err := ExpandPath("/etc/kubernetes/admin.conf")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/etc/kubernetes/admin.conf" {
		t.Fatalf("got %q", got)
	}
}
