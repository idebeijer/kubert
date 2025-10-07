## kubert ctx

Spawn a shell with the selected context

### Synopsis

Start a shell with the KUBECONFIG environment variable set to the selected context.
Kubert will issue a temporary kubeconfig file with the selected context, so that multiple shells can be spawned with different contexts.

```
kubert ctx [flags]
```

### Options

```
  -h, --help   help for ctx
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml)
      --debug           debug mode
```

### SEE ALSO

* [kubert](kubert.md)	 - kubert is a tool to switch kubernetes contexts and namespaces

