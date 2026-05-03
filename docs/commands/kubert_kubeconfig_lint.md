## kubert kubeconfig lint

Lint kubeconfig files for errors and issues

### Synopsis

Lint kubeconfig files to check for errors, warnings, and potential issues.

If no files are provided, all kubeconfig files from the configured include patterns will be linted.
If file paths are provided as arguments (including glob patterns), only those files will be linted.

```
kubert kubeconfig lint [file...] [flags]
```

### Options

```
  -h, --help   help for lint
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml, can be overridden by KUBERT_CONFIG)
      --debug           debug mode
```

### SEE ALSO

* [kubert kubeconfig](kubert_kubeconfig.md)	 - Manage and inspect kubeconfig files

