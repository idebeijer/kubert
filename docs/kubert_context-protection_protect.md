## kubert context-protection protect

Protect current context

### Synopsis

Protect current context.

This will set an explicit "protect" for the current context. That means it wil override the default setting. If the current context should use the default again, use "kubert context-protection delete".

```
kubert context-protection protect [flags]
```

### Options

```
  -h, --help   help for protect
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.kubert/config.yaml)
      --debug           debug mode
```

### SEE ALSO

* [kubert context-protection](kubert_context-protection.md)	 - Protect and unprotect contexts

