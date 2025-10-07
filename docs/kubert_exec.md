## kubert exec

Execute a command against multiple contexts

### Synopsis

Execute a command against multiple Kubernetes contexts matching a pattern.

The command will run against all contexts matching the provided pattern.
By default, uses glob-style wildcards (* and ?). Use --regex for regex patterns.

If --contexts is not provided and running in an interactive shell with fzf,
you can select multiple contexts interactively (use Tab/Shift-Tab to select).

Examples:
  # Run kubectl get pods in all production contexts
  kubert exec --contexts "prod*" -- kubectl get pods

  # Use regex to match specific patterns
  kubert exec --contexts "^(test|staging).*" --regex -- kubectl get nodes

  # Run in parallel across contexts
  kubert exec --contexts "staging*" --parallel -- kubectl get deployments

  # Specify namespace for all contexts
  kubert exec --contexts "prod*" --namespace kube-system -- kubectl get pods
  
  # Interactive multi-select (if fzf is available)
  kubert exec -- kubectl get nodes
  
  # Dry run to see which contexts will be used
  kubert exec --contexts "prod*" --dry-run -- kubectl get pods

```
kubert exec [flags] -- command [args...]
```

### Options

```
  -c, --contexts string    Pattern to match context names (omit for interactive multi-select)
      --dry-run            Show which contexts would be used without executing the command
  -h, --help               help for exec
  -n, --namespace string   Namespace to use for all contexts (default "default")
  -p, --parallel           Execute commands in parallel across all contexts
      --regex              Use regex pattern matching instead of glob-style wildcards
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml)
      --debug           debug mode
```

### SEE ALSO

* [kubert](kubert.md)	 - kubert is a tool to switch kubernetes contexts and namespaces

