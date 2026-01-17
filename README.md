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

## Quick Start

Configure `kubert` to find your kubeconfigs by creating `~/.config/kubert/config.yaml` with the following content:

```yaml
kubeconfigs:
  include:
    - "~/.kube/**"
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

# Mark the current context as (un)protected explicitly (overrides regex/defaults)
kubert context-protection protect
kubert context-protection unprotect
kubert context-protection delete     # remove explicit override

# Inspect what kubert is using right now
kubert which ctx
kubert which ns
kubert kubeconfig list
```

## Command Reference

| Command                                  | Purpose                                             | Highlights                                                                        |
| ---------------------------------------- | --------------------------------------------------- | --------------------------------------------------------------------------------- |
| `kubert ctx [<context>\|-]`              | Launch a shell pinned to a context                  | Supports interactive selection, remembers the previous context with `-`           |
| `kubert ns [<namespace>]`                | Switch namespace inside the current kubert shell    | Lists namespaces when `fzf` is unavailable                                        |
| `kubert exec [pattern...] -- <command>`  | Execute one command across multiple contexts        | Supports glob (`*`, `?`), `--regex`, `--parallel`, `--namespace`, and `--dry-run` |
| `kubert kubectl <args...>`               | Run `kubectl` with protection checks                | Blocks/asks confirmation for commands marked as protected                         |
| `kubert context-protection <subcommand>` | Manage protection status                            | `protect`, `unprotect`, `delete`, and `info` operate on the active context        |
| `kubert which <ctx\|ns\|config>`         | Print the active context, namespace, or config path | Handy for scripts/prompts                                                         |
| `kubert kubeconfig list`                 | List kubeconfig files kubert will scan              | Respects include/exclude settings                                                 |

## Context Protection

> [!WARNING]  
> Context protection works only when you run `kubectl` through `kubert kubectl`. It does not modify your existing `kubectl` binary or configuration.
> You might want to alias it for convenience, e.g., `alias k=kubert kubectl`.

Context protection ensures destructive commands can’t hit sensitive clusters by accident. Protection is only enforced when you run `kubectl` through `kubert kubectl` (consider aliasing `k=kubert kubectl`).

### Pattern-based defaults

Add a regular expression to your config to protect any context whose name matches:

```yaml
contexts:
  protectedByDefaultRegexp: "(prd|prod)"
  protectedKubectlCommands:
    - delete
    - apply
    - scale
```

- Omit or set the value to `null` to disable automatic protection.
- Use an empty string (`""`) to protect every context by default.

### Explicit overrides

Use the CLI when you want to mark a specific context as protected or unprotected regardless of the regex:

```sh
kubert context-protection protect   # enforce protection
kubert context-protection unprotect # lift protection
kubert context-protection delete    # remove explicit override and fall back to regex/default
kubert context-protection info      # show the current protection status
```

When a protected context sees a protected command, kubert will either exit immediately (`contexts.exitOnProtectedKubectlCmd: true`) or prompt for confirmation.

## Configuration

kubert reads from `~/.config/kubert/config.yaml` (override with `kubert --config <path>`). A minimal example:

```yaml
kubeconfigs:
  # Paths to include where kubert will look for kubeconfigs (supports globs)
  # Defaults:
  include:
    - "~/.kube/*"
    - "~/.kube/*.yml"
    - "~/.kube/*.yaml"
  # Paths to ignore
  exclude: []
contexts:
  protectedByDefaultRegexp: "(prod|prd)"
  protectedKubectlCommands:
    - delete
    - apply
  exitOnProtectedKubectlCmd: false
hooks:
  preShell: 'echo "Entering $KUBERT_CONTEXT ($KUBERT_NAMESPACE)"'
  postShell: 'echo "Exited $KUBERT_CONTEXT"'
```

- Run `kubert kubeconfig list` to confirm which kubeconfig files kubert will process.

### Environment Variables

All configuration settings can be overridden using environment variables prefixed with `KUBERT_`. The config structure is flattened, and dots (`.`) are replaced with underscores (`_`).

Examples:

- `fzf.opts` → `KUBERT_FZF_OPTS`
- `contexts.protectedByDefaultRegexp` → `KUBERT_CONTEXTS_PROTECTEDBYDEFAULTREGEXP`
- `hooks.preShell` → `KUBERT_HOOKS_PRESHELL`

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
