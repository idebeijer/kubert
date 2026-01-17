## kubert protection

Manage context protection

### Synopsis

Manage context protection for the current kubert shell.

Protection prevents accidentally running destructive kubectl commands in sensitive contexts.
This only works when using kubectl through "kubert kubectl" (consider aliasing k=kubert kubectl).

### Options

```
  -h, --help   help for protection
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml, can be overridden by KUBERT_CONFIG)
      --debug           debug mode
```

### SEE ALSO

* [kubert](kubert.md)	 - kubert is a tool to switch kubernetes contexts and namespaces
* [kubert protection info](kubert_protection_info.md)	 - Show protection status for current context
* [kubert protection lift](kubert_protection_lift.md)	 - Temporarily lift protection for a duration
* [kubert protection protect](kubert_protection_protect.md)	 - Explicitly protect current context
* [kubert protection remove](kubert_protection_remove.md)	 - Remove explicit protection override
* [kubert protection unprotect](kubert_protection_unprotect.md)	 - Explicitly unprotect current context

