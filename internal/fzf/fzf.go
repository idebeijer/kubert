package fzf

import (
	"os/exec"
	"strings"

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

// Select presents a list of options to the user using fzf and returns the selected option.
func Select(options []string) (string, error) {
	optionsStr := strings.Join(options, "\n")
	fzfCmd := exec.Command("fzf", "--ansi")
	fzfCmd.Stdin = strings.NewReader(optionsStr)
	output, err := fzfCmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
