<div align="center">

  <h1>kubert</h1>

**A kubectx/kubens alternative with isolated shells, context protection, and more**

[![License](https://img.shields.io/github/license/idebeijer/kubert)](https://github.com/idebeijer/kubert/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/idebeijer/kubert)](https://github.com/idebeijer/kubert/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/idebeijer/kubert)](https://goreportcard.com/report/github.com/idebeijer/kubert)
[![CI](https://img.shields.io/github/actions/workflow/status/idebeijer/kubert/build-test.yml?branch=main)](https://github.com/idebeijer/kubert/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/idebeijer/kubert)](https://go.dev/)

</div>

## Overview

`kubert` lets you hop between Kubernetes contexts and namespaces inside dedicated subshells. Each shell gets its own kubeconfig copy so you can keep production, staging, and local sessions open side-by-side without collisions. On top of that, kubert offers optional guard rails (context protection), multi-context execution, and shell hooks to give you a contextual workflow similar to `kubectx`, `kubens`, and [kubie](https://github.com/sbstp/kubie) in one tool.

**Jump to:** [Installation](#installation) | [Quick Start](#quick-start) | [Configuration](#configuration)

## Features

- **Context isolation**: spawn shells with temporary kubeconfig files so contexts never bleed into other terminals.
- **Namespace management**: switch namespaces within an active kubert shell without touching other sessions.
- **Context protection**: block (or confirm) risky `kubectl` commands in sensitive contexts. (optional, not enabled by default)
- **Multi-context fan-out**: run a command across many contexts with glob or regex selection.
- **Interactive selection**: opt into fuzzy selection with `fzf` or list contexts/namespaces in non-interactive environments.
- **Shell hooks**: run pre/post shell commands to e.g. set tab titles, or log usage.

## Installation

### Homebrew

```sh
brew install idebeijer/tap/kubert --cask
```

### Bash script (Linux & macOS)

Installs the latest binary from GitHub Releases.

```sh
curl -fsSL https://raw.githubusercontent.com/idebeijer/kubert/main/scripts/install.sh | bash
```

Install to a custom directory (e.g., `~/.local/bin`):

```sh
curl -fsSL https://raw.githubusercontent.com/idebeijer/kubert/main/scripts/install.sh | bash -s -- ~/.local/bin
```

### Arch Linux (AUR)

```sh
yay -S kubert-bin
```

### Linux Packages (.deb, .rpm, .apk)

Pre-built packages are available on the [GitHub Releases](https://github.com/idebeijer/kubert/releases/latest) page.

### From source with Go

```sh
go install github.com/idebeijer/kubert@latest
```

### Shell Completion

If you installed via Homebrew, completions are usually installed automatically. See [Homebrew's Shell Completion docs](https://docs.brew.sh/Shell-Completion).

For manual installation, add the following to your shell config (e.g., `~/.bashrc`, `~/.zshrc`):

```sh
# Bash
source <(kubert completion bash)

# Zsh
source <(kubert completion zsh)

# Fish
kubert completion fish | source
```

## Quick Start

By default, `kubert` searches for kubeconfigs in these paths (`~/.kube/config`, `~/.kube/*.yml`, `~/.kube/*.yaml`). If your config files are in other locations or you need additional configuration, please check out the [Configuration](#configuration) section.

> Tip: install [`fzf`](https://github.com/junegunn/fzf) to pick contexts and namespaces interactively. Without it, kubert prints the available options so you can copy/paste.

```sh
# Start an isolated shell; choose a context interactively (fzf) or name it directly
kubert ctx
kubert ctx my-cluster
kubert ctx -             # jump back to the previously used context

# Switch namespaces inside the current kubert shell
kubert ns kube-system

# Run a command across several contexts (glob, regex, or interactive multi-select)
kubert exec "prod-*" "staging-?" -- kubectl get nodes
kubert exec --regex "^(dev|qa)-.*" -- kubectl get pods
kubert exec --parallel --dry-run "prod-*" -- kubectl rollout status

# Open the used kubert config file, editor is determined by $EDITOR or $VISUAL, falls back to 'vim'
kubert config edit

# Wrap kubectl to enforce context protection rules
kubert kubectl get pods

# Manage context protection (optional, no protection by default)
kubert protection info      # show current protection status
kubert protection protect   # explicitly protect current context (overrides default regex)
kubert protection unprotect # explicitly unprotect current context (overrides default regex)
kubert protection lift 5m   # temporarily lift protection for 5 minutes
kubert protection remove    # remove explicit override, fall back to regex

# Inspect what kubert is using right now
kubert which ctx
kubert which ns
kubert kubeconfig list
kubert kubeconfig lint  # check kubeconfig files for errors and issues
```

## Command Reference

For more information on all commands, see the [docs](docs/commands/README.md).

## Configuration

kubert reads from `~/.config/kubert/config.yaml`. You can override this location using the `KUBERT_CONFIG` environment variable or the `--config <path>` flag.

```yaml
# All settings shown with their defaults.

# Paths to kubeconfig files. Supports glob patterns.
kubeconfigs:
  include:
    - "~/.kube/config"
    - "~/.kube/*.yml"
    - "~/.kube/*.yaml"

  # Exclude these patterns. (takes precedence over include)
  exclude: []

# Use `fzf` for interactive context/namespace selection when available.
# If `fzf` is not found, kubert falls back to a non-interactive list.
interactive: true

# Context switch mode when already inside a kubert shell (i.e., when KUBERT_SHELL_ACTIVE=1):
# - false (default): switch context in-place, stay in the same shell
# - true: spawn a new nested sub-shell with the new context (nested mode)
nested: false

# Protect contexts against accidental destructive commands. See "Context Protection" below for details. (not configured by default)
protection:
  regex: null # regex pattern to auto-protect matching contexts (e.g., "(prod|prd)")
  commands: # kubectl commands to block in protected contexts
    - delete
    - edit
    - exec
    - drain
    - scale
    - autoscale
    - replace
    - apply
    - patch
    - set
  prompt: true # ask for confirmation (false = exit immediately)

hooks:
  preShell: "" # run before spawning shell or switching context in-place
  postShell: "" # run after exiting shell or switching to another context in-place

fzf:
  opts: "" # additional fzf options
```

> Tip: run `kubert kubeconfig list` to confirm which kubeconfig files kubert will process.

### Environment Variables

All config settings can be overridden via environment variables prefixed with `KUBERT_`. Dots (`.`) become underscores (`_`).

Examples:

- `protection.regex` → `KUBERT_PROTECTION_REGEX`
- `protection.prompt` → `KUBERT_PROTECTION_PROMPT`
- `fzf.opts` → `KUBERT_FZF_OPTS`

### FZF Customization

Customize fzf appearance via the `fzf.opts` config setting:

```yaml
fzf:
  opts: "--height=50% --border --layout=reverse"
```

This can also be overridden via environment variable: `KUBERT_FZF_OPTS`.

`FZF_DEFAULT_OPTS` is inherited natively by fzf and applies as a fallback.

### Shell Hooks

Hooks let you run shell commands before and after kubert spawns the subshell (and on every in-place context switch). They run in a child shell process attached to the same TTY, so terminal-side effects like updating the tab title, sending notifications, or logging actions work, but environment changes such as `export` or prompt-variable updates do not persist in the caller's shell.

Name shell tab after selected Kubernetes context:

```yaml
hooks:
  preShell: 'printf "\033]0;k8s: $KUBERT_SHELL_CONTEXT\007"'
  postShell: 'printf "\033]0;\007"'
```

> **Note:** `$KUBERT_SHELL_CONTEXT` in hooks is only available and reliable when shell-init is configured. See [Shell Init](#shell-init-optional) below.

### Shell Init (Optional)

`kubert shell-init` prints a shell function that wraps the `kubert` binary. When
sourced, it enables kubert to keep `KUBERT_SHELL_CONTEXT` and
`KUBERT_SHELL_ORIGINAL_KUBECONFIG` up to date across in-place context switches —
something kubert cannot do on its own because a child process cannot modify its
parent shell's environment.

Add one of the following to your shell config:

```sh
# Bash (~/.bashrc)
eval "$(kubert shell-init bash)"

# Zsh (~/.zshrc)
eval "$(kubert shell-init zsh)"

# Fish (~/.config/fish/config.fish)
kubert shell-init fish | source
```

Without shell-init, `KUBERT_SHELL_CONTEXT` and `KUBERT_SHELL_ORIGINAL_KUBECONFIG`
are intentionally not set (a stale value is worse than no value). `KUBECONFIG`
always works correctly regardless because kubert rewrites the file contents
in-place rather than changing the path.

### Starship Prompt Integration

You can show the current context protection status in your prompt using a custom [starship.rs](https://starship.rs) module.
Add the following to your `starship.toml`:

<details>
<summary>Click to expand configuration</summary>

```toml
[custom.kubert]
command = '''
  BINARY="kubert"
  prot=$($BINARY protection info -o short)

  red="\033[31m"
  yellow="\033[33m"
  green="\033[32m"
  reset="\033[0m"

  case "$prot" in
    "lifted")      color=$yellow ;;
    "unprotected") color=$green ;;
    "protected")   color=$red ;;
    *)             color=$reset ;;
  esac

  printf "${color}${prot}${reset}"
'''
format = '\([ctx-protection](dimmed white):$output(dimmed white)\) '
when = 'test -n "$KUBERT_SHELL_ACTIVE"'
```

Looks like:

```bash
(orbstack:default) (ctx-protection:protected) ~
❯ k get pods
```

</details>

For the context and namespace display, the standard `[kubernetes]` module works out of the box because kubert manages the standard `KUBECONFIG` environment variable:

```toml
[kubernetes]
format = '\([($cluster)](red):[$namespace](cyan)\) '
disabled = false
```

## Context Protection

> [!WARNING]  
> Context protection works only when you run `kubectl` through `kubert kubectl`. It does not modify your existing `kubectl` binary or configuration.
> You might want to alias it for convenience, e.g., `alias k=kubert kubectl`.

Context protection ensures destructive commands can’t hit sensitive clusters by accident. Protection is only enforced when you run `kubectl` through `kubert kubectl` (consider aliasing `k=kubert kubectl`).

### Pattern-based defaults

Add a regular expression to your config to protect any context whose name matches:

```yaml
protection:
  regex: "(prd|prod)" # protect contexts matching this pattern
  commands:
    - delete
    - apply
    - scale
  prompt: true # ask for confirmation (false = exit immediately)
```

- Omit or set `regex: null` to disable automatic protection.
- Use `regex: ""` to protect every context by default.

### Explicit overrides

Use the CLI to manage protection for the current context:

```sh
kubert protection protect   # explicitly protect this context
kubert protection unprotect # explicitly unprotect this context
kubert protection lift 5m   # temporarily lift protection for 5 minutes
kubert protection remove    # remove explicit override, fall back to default regex
```

When a protected context sees a protected command, kubert will prompt for confirmation (`prompt: true`) or exit immediately (`prompt: false`).

## Limitations

### `KUBERT_SHELL_CONTEXT` requires shell-init to stay accurate

When you switch contexts inside an active kubert shell (in-place switch), kubert
runs as a child process of your shell. A child process cannot modify environment
variables in its parent — so `KUBERT_SHELL_CONTEXT` would go stale the moment
you switch context for the first time.

`KUBECONFIG` avoids this problem because kubert rewrites the _file contents_
rather than changing the variable value — the path stays the same, kubectl picks
up the new context automatically.

For `KUBERT_SHELL_CONTEXT` and `KUBERT_SHELL_ORIGINAL_KUBECONFIG` to stay
accurate, kubert needs cooperation from the shell itself. `kubert shell-init`
provides this: the generated shell function sources a small env-update file after
every `kubert ctx` call so these vars are always current.

Without shell-init, both vars are intentionally not set to avoid misleading stale
values. Configure shell-init if you rely on `$KUBERT_SHELL_CONTEXT` in hooks or
your prompt. See [Shell Init](#shell-init-optional) for setup.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for information on how to contribute to `kubert`.
