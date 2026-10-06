package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("KUBERT_EXPAND_TEST", "expanded-value")

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "lone tilde", in: "~", want: home},
		{name: "tilde slash", in: "~/.kube/config", want: filepath.Join(home, ".kube", "config")},
		{name: "other user tilde", in: "~otheruser/.kube/config", want: "~otheruser/.kube/config"},
		{name: "double tilde", in: "~~", want: "~~"},
		{name: "absolute", in: "/etc/kubernetes/admin.conf", want: "/etc/kubernetes/admin.conf"},
		{name: "env var", in: "$KUBERT_EXPAND_TEST/config", want: "expanded-value/config"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpandPath(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("ExpandPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
