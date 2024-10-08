## kubert context-protection delete

Delete protection setting for the current context

### Synopsis

Delete protection setting for the current context.

This will delete the explicit protect/unprotect setting for the current context. So if either "protect" or "unprotect" was set, it will be removed and the default will be used.

```
kubert context-protection delete [flags]
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.kubert/config.yaml)
      --debug           debug mode
```

### SEE ALSO

* [kubert context-protection](kubert_context-protection.md)	 - Protect and unprotect contexts

