## kubert protection lift

Temporarily lift protection for a duration

### Synopsis

Temporarily lift protection for the current context.

The duration argument is required and specifies how long protection should be lifted.
Examples: 5m (5 minutes), 1h (1 hour), 30s (30 seconds)

After the duration expires, protection will automatically be restored.

```
kubert protection lift <duration> [flags]
```

### Examples

```
  # Lift protection for 5 minutes
  kubert protection lift 5m
```

### Options

```
  -h, --help   help for lift
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.config/kubert/config.yaml, can be overridden by KUBERT_CONFIG)
      --debug           debug mode
```

### SEE ALSO

* [kubert protection](kubert_protection.md)	 - Manage context protection

