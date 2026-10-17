---
name: rocketchat-cli
description: Use rocketchat, the agent CLI for Rocket.Chat contexts, users, channels, direct messages, and messages.
---

# Rocket.Chat CLI

## Quick start

```sh
rocketchat auth status --json
rocketchat context list --json
rocketchat channel list --json
rocketchat channel history general --json
rocketchat dm list --all --json
rocketchat dm history alice --json
rocketchat message get MESSAGE_ID --json
rocketchat message search general 'deployment' --json
rocketchat message thread MESSAGE_ID --all --json
# Only after inspecting the destination and obtaining intent to mutate:
rocketchat message send :general 'Deployment complete' --json
rocketchat message edit MESSAGE_ID 'Corrected deployment status' --json
rocketchat message reply MESSAGE_ID 'Acknowledged' --json
```

## Rules

- Inspect the selected context, destination room/user, and relevant message before mutation. Never expose tokens in output, messages, or logs.
- Use `--json` for machine consumption. `--raw` is the final Rocket.Chat API response, not normalized output; it cannot be combined with `--json`.
- `channel` unifies public channels and private groups.
- `message send :<channel> <text|->` sends to a channel/group. An unprefixed target is a username, user ID, or email for a direct message, not a channel or DM room ID.
- `dm` discovers only joined one-to-one conversations and never creates one. Sending to an identified user can create a DM.
- `message reply` and `message thread` accept a root or reply ID and resolve the thread root. `--also-send` is opt-in when a reply should also appear in the main channel.
- Use `-` to read generated text from stdin for send/edit/reply. Trailing CR/LF is stripped; empty stdin text is rejected.
- Deletion requires explicit user intent. Inspect the message first and use `--yes` non-interactively. Never infer permission to delete from permission to edit.
- Inspect after meaningful mutations. Never blindly retry a mutation whose outcome is uncertain; first inspect server state to avoid duplicate sends or unintended changes.

## Core commands

| Group | Purpose |
| --- | --- |
| `context` | List, inspect, create, select, and delete saved contexts |
| `auth` | Log in, remove saved tokens, and verify authentication |
| `me` | Inspect the authenticated user |
| `user` | Get, list, and search users |
| `channel` | Inspect joined public/private channels, members, and history |
| `dm` | Discover and browse existing one-to-one conversations |
| `message` | Get, search, send, edit, reply, read threads, and delete messages |

See [commands](references/commands.md) for exact syntax and flags, and [workflows](references/workflows.md) for inspect-then-act examples.
