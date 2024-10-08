## kubert context-protection

Protect and unprotect contexts

### Synopsis

Protect and unprotect contexts.

This will allow you to protect and unprotect contexts for the "kubert kubectl" command. This can be useful if you want to prevent accidentally running certain kubectl commands to a cluster.
Keep in mind, this will only work when using kubectl through the "kubert kubectl" command. Direct commands using just "kubectl" will not be blocked. (If you use this feature, you could set an alias e.g. "k" for "kubert kubectl".)

Both "protect" and "unprotect" will set an explicit setting for the given context. That means if either of those has been set, kubert will ignore the default setting. If you want to use the default setting again, use "kubert context-protection delete <context>".

What kubectl commands should be blocked can be configured in the kubert configuration file.

### Options

```
  -h, --help   help for context-protection
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.kubert/config.yaml)
      --debug           debug mode
```

### SEE ALSO

* [kubert](kubert.md)	 - kubert is a tool to switch kubernetes contexts and namespaces
* [kubert context-protection delete](kubert_context-protection_delete.md)	 - Delete protection setting for the current context
* [kubert context-protection info](kubert_context-protection_info.md)	 - Show protection status for the current context
* [kubert context-protection protect](kubert_context-protection_protect.md)	 - Protect current context
* [kubert context-protection unprotect](kubert_context-protection_unprotect.md)	 - Unprotect current context

