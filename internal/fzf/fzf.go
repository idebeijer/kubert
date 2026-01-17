package fzf

import (
	"os"
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
	return err == nil
}

// buildFzfArgs constructs fzf arguments based on config.
func buildFzfArgs(multiSelect bool) []string {
	cfg := config.Cfg
	var args []string

	// Check for kubert-specific options from config
	if cfg.Fzf.Opts != "" {
		args = append(args, parseArgs(cfg.Fzf.Opts)...)
	}

	// Always ensure --ansi is present for color support
	hasAnsi := false
	for _, arg := range args {
		if arg == "--ansi" {
			hasAnsi = true
			break
		}
	}
	if !hasAnsi {
		args = append(args, "--ansi")
	}

	// Add --multi flag if multi-select is requested
	if multiSelect {
		hasMulti := false
		for _, arg := range args {
			if arg == "--multi" || arg == "-m" {
				hasMulti = true
				break
			}
		}
		if !hasMulti {
			args = append(args, "--multi")
		}
	}

	return args
}

// parseArgs splits a string of arguments respecting quoted strings
func parseArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range s {
		switch {
		case (r == '"' || r == '\'') && !inQuote:
			inQuote = true
			quoteChar = r
		case r == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// Select presents a list of options to the user using fzf and returns the selected option.
func Select(options []string) (string, error) {
	optionsStr := strings.Join(options, "\n")
	args := buildFzfArgs(false)

	fzfCmd := exec.Command("fzf", args...)
	fzfCmd.Stdin = strings.NewReader(optionsStr)
	fzfCmd.Stderr = os.Stderr
	output, err := fzfCmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// SelectMulti presents a list of options to the user using fzf in multi-select mode
// and returns the selected options. Users can select multiple items with Tab/Shift-Tab.
func SelectMulti(options []string) ([]string, error) {
	optionsStr := strings.Join(options, "\n")
	args := buildFzfArgs(true)

	fzfCmd := exec.Command("fzf", args...)
	fzfCmd.Stdin = strings.NewReader(optionsStr)
	fzfCmd.Stderr = os.Stderr
	output, err := fzfCmd.Output()
	if err != nil {
		return nil, err
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return []string{}, nil
	}

	return strings.Split(result, "\n"), nil
}
