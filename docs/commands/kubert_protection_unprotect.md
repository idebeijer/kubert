## kubert protection unprotect

Explicitly unprotect current context

### Synopsis

Explicitly unprotect the current context.

This sets an explicit unprotected override for the current context.
To revert to the default regex-based protection, use "kubert protection remove".

```
kubert protection unprotect [flags]
```

### Options

```
  -h, --help   help for unprotect
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml, can be overridden by KUBERT_CONFIG)
      --debug           debug mode
```

### SEE ALSO

* [kubert protection](kubert_protection.md)	 - Manage context protection

