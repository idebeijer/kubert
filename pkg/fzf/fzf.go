package fzf

import (
	"os/exec"

	"github.com/idebeijer/kubert/internal/config"
)

func IsInteractiveShell() bool {
	cfg := config.Cfg
	if !cfg.InteractiveShellMode {
		return false
	}
	_, err := exec.LookPath("fzf")
	if err != nil {
		return false
	}

	return true
}
