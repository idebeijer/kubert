<!-- This file is a generated copy of kubert.md for GitHub display purposes -->
> [!NOTE]
> This file is a generated copy of `kubert.md`.

## kubert

kubert is a tool to switch kubernetes contexts and namespaces

### Synopsis

kubert is a CLI tool to switch kubernetes contexts and namespaces within an isolated shell so you can have multiple shells with different contexts and namespaces.

It also includes a wrapper around kubectl to provide the ability to protect contexts by setting a regex pattern to match the context name. This can be used to prevent accidentally running certain kubectl commands in an unwanted context.
Keep in mind, this will only work when using kubectl through the "kubert kubectl" command. Direct commands using just "kubectl" will not be blocked. (If you use this feature, you could set an alias e.g. "k" for "kubert kubectl".)


### Options

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml, can be overridden by KUBERT_CONFIG)
      --debug           debug mode
  -h, --help            help for kubert
```

### SEE ALSO

* [kubert ctx](kubert_ctx.md)	 - Spawn a shell with the selected context
* [kubert exec](kubert_exec.md)	 - Execute a command against multiple contexts
* [kubert kubeconfig](kubert_kubeconfig.md)	 - Kubeconfig command
* [kubert kubectl](kubert_kubectl.md)	 - Wrapper for kubectl
* [kubert ns](kubert_ns.md)	 - Switch to a different namespace
* [kubert protection](kubert_protection.md)	 - Manage context protection
* [kubert version](kubert_version.md)	 - Display version and build information
* [kubert which](kubert_which.md)	 - Display information about current context, cluster, namespace, or config

