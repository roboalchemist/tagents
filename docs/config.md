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

## SSH Password Auth (sshpass)

For hosts that disable public-key auth (`PubkeyAuthentication no`) and only accept a
password, tagents wraps `ssh` with `sshpass` (which must be on your PATH). It resolves
the password file for a host in this order:

1. An explicit `# tagents-sshpass-file <path>` comment in the host block:
   ```ssh-config
   Host example-host
     HostName Mac.local
     User user
     PubkeyAuthentication no
     # tagents-sshpass-file ~/.ssh/example-host-pw
   ```
   A comment is used deliberately: `ssh` rejects unknown keywords with a fatal
   "Bad configuration option" error, but ignores comments.
2. Auto-detection: when the host sets `PubkeyAuthentication no` and `~/.ssh/<host>-pw`
   exists, that file is used.

Password mode forces `PreferredAuthentications=password,keyboard-interactive` and omits
`BatchMode=yes` (which would suppress the password prompt sshpass answers). Hosts that
use keys are unaffected. If a password is configured but `sshpass` is missing, the
command fails with a clear error. `tagents machines` reports the resolved method in the
`AUTH` column and the `auth` JSON field.

Note: `~` in a password-file path is expanded, so `~/.ssh/example-host-pw` works as written.
