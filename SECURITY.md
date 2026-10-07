# Security Policy

## Supported versions

The latest release is supported with security fixes.

## Reporting a vulnerability

Please report security vulnerabilities using GitHub's private vulnerability
reporting:

https://github.com/roboalchemist/tagents/security/advisories/new

Do **not** open a public issue for security vulnerabilities. We aim to
acknowledge reports within a few days.

## Scope notes

tagents shells out to `tmux` and `ssh` from your PATH and reads `~/.ssh/config`.
With `sshpass` configured, passwords are read from a file you specify — tagents
never stores or logs them. Review your SSH config and password-file permissions
(`chmod 600`) accordingly.
