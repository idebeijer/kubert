## kubert context-protection unprotect

Unprotect current context

### Synopsis

Unprotect current context. 

This will set an explicit "unprotect" for the current context. That means it wil override the default setting. If the current context should use the default again, use "kubert context-protection delete".

```
kubert context-protection unprotect [flags]
```

### Options

```
  -h, --help   help for unprotect
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml)
      --debug           debug mode
```

### SEE ALSO

* [kubert context-protection](kubert_context-protection.md)	 - Protect and unprotect contexts

