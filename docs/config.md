# tagents Configuration

tagents requires no configuration files. It uses:

## Machine Discovery

Reads `~/.ssh/config` automatically. All `Host` entries (non-wildcard) are candidates for
`--machine` and `--all-machines`. `Include` directives are supported — tagents follows them
to read additional config files.

## Environment Variables

No environment variables required or used.

## SSH Config Example

```ssh-config
Host gateway
  HostName gateway.tailnet.example.com
  User ubuntu

Host mini
  HostName 192.168.1.10
  User mini

Include ~/.ssh/config.d/*
```

Both `gateway` and `mini` will appear in `tagents machines`. Hosts defined in included
files are also discovered.

## tmux

tagents uses the `tmux` binary from your PATH. No configuration required.
It reads pane contents and sends keys to existing sessions — it does not create sessions.

## SSH

tagents uses the `ssh` binary from your PATH for remote machine operations.
It inherits your existing SSH config (keys, jump hosts, ControlMaster, etc.).
No separate auth is needed beyond your existing SSH setup.
