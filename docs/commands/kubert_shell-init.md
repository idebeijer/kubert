## kubert shell-init

Print optional shell integration script for the given shell

### Synopsis

Print an optional shell function that wraps the kubert binary. (not required for core functionality)

Optionally source it once in your shell rc file so that env vars like KUBERT_SHELL_CONTEXT
are kept accurate after in-place context switches.

  bash/zsh:  eval "$(kubert shell-init bash)"
  fish:      kubert shell-init fish | source

If no shell is given, kubert attempts to detect it from $SHELL.

```
kubert shell-init [bash|zsh|fish] [flags]
```

### Options

```
  -h, --help   help for shell-init
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml, can be overridden by KUBERT_CONFIG)
      --debug           debug mode
```

### SEE ALSO

* [kubert](kubert.md)	 - kubert is a tool to switch kubernetes contexts and namespaces

