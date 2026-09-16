# Rocket.Chat CLI skill

Use `rocketchat` to read and interact with Rocket.Chat. Prefer `--json` when consuming output programmatically.

## Context and identity

```bash
rocketchat context current
rocketchat context list --json
rocketchat auth status --json
rocketchat me --json
```

Do not print or expose authentication tokens. Select another workspace with `--context <name>` without changing the current context.

## Users

```bash
rocketchat user get <username|id|email> --json
rocketchat user search <text> --json
rocketchat user list --all --json
```

## Channels

`channel` covers both public channels and private groups. The CLI resolves the underlying Rocket.Chat room type automatically.

```bash
rocketchat channel list --json
rocketchat channel get <name|id> --json
rocketchat channel members <name|id> --all --json
rocketchat channel history <name|id> --limit 50 --json
rocketchat channel history <name|id> --since 2h --json
```

History/search output includes message IDs. Use those IDs for `message get`, `reply`, and `thread`.

## Messages

```bash
rocketchat message get <message-id> --json
rocketchat message search <channel> <text> --json
rocketchat message thread <message-id> --all --json
rocketchat message send :<channel> "text" --json
rocketchat message send <username|email> "text" --json
rocketchat message edit <message-id> "replacement text" --json
rocketchat message reply <message-id> "text" --json
```

Prefix every public-channel or private-group target with `:`. An unprefixed `message send` target is a direct-message user; it resolves usernames, user IDs, and email addresses and must not be used for channels.

`message reply` accepts either the thread root or any reply in the thread. The CLI resolves the actual thread root automatically.

For generated or multiline text, use stdin:

```bash
some-command | rocketchat message send :<channel> - --json
some-command | rocketchat message send <username> - --json
some-command | rocketchat message edit <message-id> - --json
some-command | rocketchat message reply <message-id> - --json
```

Use `--also-send` only when the thread reply must also appear in the main channel.

Do not delete a message unless the user explicitly requested deletion. Non-interactive deletion requires `--yes`:

```bash
rocketchat message delete <message-id> --yes
```
