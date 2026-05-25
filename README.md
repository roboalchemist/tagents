# tagents

Manage AI agent fleet across tmux sessions and SSH machines.

See `tagents docs` for full documentation.

## Install

```bash
brew tap roboalchemist/tap ssh://git@github.com:2222/roboalchemist/homebrew-tap.git
brew install tagents
```

## Quick Start

```bash
tagents list              # list all agent sessions
tagents status            # fleet summary
tagents read myagent      # read last 50 lines of agent output
tagents send myagent "please continue"
```

## Report Bugs

Report bugs to: https://github.com/roboalchemist/tagents/issues
