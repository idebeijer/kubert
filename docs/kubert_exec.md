## kubert exec

Execute a command against multiple contexts

### Synopsis

Execute a command against multiple Kubernetes contexts matching one or more patterns.

The command will run against all contexts matching the provided patterns.
By default, uses glob-style wildcards (* and ?). Use --regex for regex patterns.

If no patterns are provided and running in an interactive shell with fzf,
you can select multiple contexts interactively (use Tab/Shift-Tab to select).

```
kubert exec [pattern...] -- command [args...] [flags]
```

### Examples

```
  # Run kubectl get pods in all production contexts
  kubert exec "prod*" -- kubectl get pods

  # Match multiple patterns
  kubert exec "prod*" "staging*" -- kubectl get nodes

  # Use regex to match specific patterns
  kubert exec --regex "^(test|staging).*" -- kubectl get nodes

  # Run in parallel across contexts
  kubert exec "staging*" --parallel -- kubectl get deployments

  # Specify namespace for all contexts
  kubert exec "prod*" --namespace kube-system -- kubectl get pods
  
  # Interactive multi-select (if fzf is available)
  kubert exec -- kubectl get nodes
  
  # Dry run to see which contexts will be used
  kubert exec "prod*" --dry-run -- kubectl get pods
```

### Options

```
      --dry-run            Show which contexts would be used without executing the command
  -h, --help               help for exec
  -n, --namespace string   Namespace to use for all contexts (default "default")
  -p, --parallel           Execute commands in parallel across all contexts
      --regex              Use regex pattern matching instead of glob-style wildcards
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml, can be overridden by KUBERT_CONFIG)
      --debug           debug mode
```

### SEE ALSO

* [kubert](kubert.md)	 - kubert is a tool to switch kubernetes contexts and namespaces

