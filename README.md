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

`kubert` lets you hop between Kubernetes contexts and namespaces inside dedicated subshells. Each shell gets its own kubeconfig copy so you can keep production, staging, and local sessions open side-by-side without collisions. On top of that, kubert offers guard rails (context protection), multi-context execution, and shell hooks to give you a contextual workflow similar to `kubectx`, `kubens`, and [kubie](https://github.com/sbstp/kubie) in one tool.

## Features

- **Context isolation**: spawn shells with temporary kubeconfig files so contexts never bleed into other terminals.
- **Namespace management**: switch namespaces within an active kubert shell without touching other sessions.
- **Context protection**: block (or confirm) risky `kubectl` commands in sensitive contexts.
- **Multi-context fan-out**: run a command across many contexts with glob or regex selection.
- **Interactive selection**: opt into fuzzy selection with `fzf` or list contexts/namespaces in non-interactive environments.
- **Shell hooks**: run pre/post shell commands to tweak prompts, set tab titles, or log usage.

## Installation

### Homebrew

```sh
brew install idebeijer/tap/kubert --cask
```

### From source

```sh
go install github.com/idebeijer/kubert@latest
```

### Shell Completion

If you installed via Homebrew, completions are usually installed automatically. To configure your shell to load them, see [Homebrew's Shell Completion docs](https://docs.brew.sh/Shell-Completion). Completions are also available through `kubert completion <shell>`.

## Quick Start

By default, `kubert` searches for kubeconfigs in:

- `~/.kube/config`
- `~/.kube/*.yml`
- `~/.kube/*.yaml`

If your config files are in these locations, you don't need any additional configuration. To scan other directories, create `~/.config/kubert/config.yaml`:

```yaml
kubeconfigs:
  include:
    - "~/custom/k8s/configs/*"
```

> Tip: install [`fzf`](https://github.com/junegunn/fzf) to pick contexts and namespaces interactively. Without it, kubert prints the available options so you can copy/paste.

```sh
# Start an isolated shell; choose a context interactively (fzf) or name it directly
kubert ctx
kubert ctx my-cluster
kubert ctx -             # jump back to the previously used context

# Switch namespaces inside the current kubert shell
kubert ns kube-system

# Wrap kubectl to enforce context protection rules
kubert kubectl get pods

# Run a command across several contexts (glob, regex, or interactive multi-select)
kubert exec "prod-*" "staging-?" -- kubectl get nodes
kubert exec --regex "^(dev|qa)-.*" -- kubectl get pods
kubert exec --parallel --dry-run "prod-*" -- kubectl rollout status

# Manage context protection
kubert protection           # show current protection status
kubert protection protect   # explicitly protect current context
kubert protection unprotect # explicitly unprotect current context
kubert protection lift 5m   # temporarily lift protection for 5 minutes
kubert protection remove    # remove explicit override, fall back to regex

# Inspect what kubert is using right now
kubert which ctx
kubert which ns
kubert kubeconfig list
```

## Command Reference

| Command                                 | Purpose                                             | Highlights                                                                        |
| --------------------------------------- | --------------------------------------------------- | --------------------------------------------------------------------------------- |
| `kubert ctx [<context>\|-]`             | Launch a shell pinned to a context                  | Supports interactive selection, remembers the previous context with `-`           |
| `kubert ns [<namespace>]`               | Switch namespace inside the current kubert shell    | Lists namespaces when `fzf` is unavailable                                        |
| `kubert exec [pattern...] -- <command>` | Execute one command across multiple contexts        | Supports glob (`*`, `?`), `--regex`, `--parallel`, `--namespace`, and `--dry-run` |
| `kubert kubectl <args...>`              | Run `kubectl` with protection checks                | Blocks/asks confirmation for commands marked as protected                         |
| `kubert protection <subcommand>`        | Manage context protection                           | `protect`, `unprotect`, `lift`, `remove`, and `info`                              |
| `kubert which <ctx\|ns\|config>`        | Print the active context, namespace, or config path | Handy for scripts/prompts                                                         |
| `kubert kubeconfig list`                | List kubeconfig files kubert will scan              | Respects include/exclude settings                                                 |

## Configuration

kubert reads from `~/.config/kubert/config.yaml`. You can override this location using the `KUBERT_CONFIG` environment variable or the `--config <path>` flag.

```yaml
# All settings shown with their defaults
kubeconfigs:
  include:
    - "~/.kube/config"
    - "~/.kube/*.yml"
    - "~/.kube/*.yaml"
  exclude: []

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
  preShell: "" # run before spawning shell
  postShell: "" # run after exiting shell

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

Hooks let you run shell commands before and after kubert spawns the subshell. They execute in the parent shell, so you can adjust prompts, send notifications, or log actions.

Name shell tab after selected Kubernetes context:

```yaml
hooks:
  preShell: 'echo "\033]0;k8s: $KUBERT_CONTEXT\007"'
  postShell: 'echo "\033]0;\007"'
```

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
