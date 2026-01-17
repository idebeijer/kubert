## kubert protection remove

Remove explicit protection override

### Synopsis

Remove any explicit protection override for the current context.

This clears both the explicit protected/unprotected setting and any active lift,
reverting the context to use the default regex-based protection from config.

```
kubert protection remove [flags]
```

### Options

```
  -h, --help   help for remove
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml, can be overridden by KUBERT_CONFIG)
      --debug           debug mode
```

### SEE ALSO

* [kubert protection](kubert_protection.md)	 - Manage context protection

