# kubert

A `kubectx`/`kubens` alternative inspired by [kubie](https://github.com/sbstp/kubie).
`kubert` is a tool that allows you to switch between Kubernetes contexts and namespaces within an isolated shell.
That way, you can have multiple shells open, each with a different context and namespace.

kubert also has a wrapper for `kubectl` (`kubert kubectl`) to enable you to protect contexts to prevent accidentally running
certain kubectl commands in the wrong context. Checkout the [Protecting contexts](#protecting-contexts) section for more information.

## Installation

### Homebrew

```sh
brew install idebeijer/tap/kubert --cask
```

## Usage

### Switching contexts

To switch to a different context, run:

```sh
kubert ctx <context-name>
```

You can also print out a list of available contexts by running, or if `fzf` is installed, you can select a context from a list:

```sh
kubert ctx
```

### Switching namespaces

To switch to a different namespace, run:

```sh
kubert ns <namespace-name>
```

You can also print out a list of available namespaces by running, or if `fzf` is installed, you can select a namespace from a list:

```sh
kubert ns
```

### Protecting contexts

Kubert can be configured to protect certain contexts, to prevent you from accidentally running certain kubectl commands in the wrong context.
This will only work when using kubectl through the `kubert kubectl` command, it will **not** work when using `kubectl` directly.

To protect a context, you can either set a regex pattern in the Kubert config file, or you can explicitly protect a context.
When a context is protected, you will be prompted to confirm that you want to run a protected kubectl command in that context.

#### Context protection using a regex pattern

To protect a context using a regex pattern, you need to set the `protectedByDefaultRegexp` in the Kubert config file.
The following example will protect all contexts that contain `prd` or `prod` in their name:

```yaml
contexts:
  protectedByDefaultRegexp: "(prd|prod)"
```

By default, kubert has set this setting to `null`, which means that no contexts are protected by default.
If you provide an empty string as pattern `""`, all contexts will be protected by default.

The default regex pattern will be ignored for contexts that have an explicit protection set.

#### Setting an explicit protect/unprotect

Instead of using a regex pattern, you can also explicitly protect the current context.
When using an explicit protect or unprotect, Kubert will save this in a state file, and the default regex pattern will be ignored for this context.

To tell Kubert to protect the current context, run:

```sh
kubert context-protection protect
```

To tell Kubert to explicitly unprotect the current context, run:

```sh
kubert context-protection unprotect
```

The above commands will set an explicit protection for the current context. This means that the context will be (un)protected even if it matches the regex pattern or not.
To delete the explicit protection, run:

```sh
kubert context-protection delete
```

### Shell hooks

Kubert supports pre and post shell hooks that can be configured to run custom commands before and after spawning a shell with the selected context.
This is useful for customizing your shell environment based on the selected context.

#### Configuration

You can configure hooks in the Kubert config file (`~/.config/kubert/config.yaml`):

```yaml
hooks:
  # Command to run before spawning the shell
  preShell: 'echo "Entering context: $KUBERT_CONTEXT"'

  # Command to run after exiting the shell
  postShell: 'echo "Exited context: $KUBERT_CONTEXT"'
```

#### Examples

**Set terminal tab title to the context name and reset it on exit:**

```yaml
hooks:
  preShell: 'echo "\033]0;k8s: $KUBERT_CONTEXT\007"'
  postShell: 'echo "\033]0;\007"'
```

**Send a notification when entering/exiting a production context:**

```yaml
hooks:
  preShell: |
    if [[ "$KUBERT_CONTEXT" == *"prod"* ]]; then
      osascript -e 'display notification "Entering production context!" with title "Kubert"'
    fi
  postShell: |
    if [[ "$KUBERT_CONTEXT" == *"prod"* ]]; then
      osascript -e 'display notification "Exited production context" with title "Kubert"'
    fi
```

**Log context usage:**

```yaml
hooks:
  preShell: 'echo "$(date): Entered context $KUBERT_CONTEXT" >> ~/.kubert_usage.log'
  postShell: 'echo "$(date): Exited context $KUBERT_CONTEXT" >> ~/.kubert_usage.log'
```
