# Command reference

All examples use the `rocketchat` executable. Angle brackets denote required positional arguments; brackets denote optional arguments.

## Global flags

| Flag | Meaning |
| --- | --- |
| `--config <path>` | Configuration file |
| `--context <name>` | Context override |
| `--url <url>` | Rocket.Chat URL override |
| `--user-id <id>` | Authentication user ID override |
| `--token <token>` | Authentication token override; keep secret |
| `--timeout <duration>` | HTTP timeout, default `30s` |
| `--json` | Normalized JSON for machine consumption |
| `--raw` | Final Rocket.Chat response for API-backed commands, including only the final response of multi-request operations |

`--json` and `--raw` are mutually exclusive. Local context and skill operations do not have a Rocket.Chat response. Cobra provides `--help` (`-h`); the root also provides `--version`.

## Context and authentication

| Syntax | Flags and behavior |
| --- | --- |
| `context list` | List saved contexts without exposing tokens |
| `context current` | Show selected context; errors if none |
| `context get <name>` | Inspect URL, user ID, and token presence |
| `context set <name>` | `--url`, `--user-id`, `--token`; only supplied fields change; URL required for a new context; first context becomes current |
| `context use <name>` | Select an existing context |
| `context delete <name>` | Requires `--yes` (`-y`); deleting the current context clears selection |
| `auth login [context]` | `--url`, `--user-id`, `--token`; uses saved values or prompts for missing values; verifies credentials and saves/selects the context; defaults to current context or `default` |
| `auth logout [context]` | Remove saved token from specified/current context |
| `auth status` | Verify current credentials |
| `me` | Get authenticated user |

## Users

| Syntax | Flags |
| --- | --- |
| `user get <username|id|email>` | No local flags |
| `user list` | `--limit 50`, `--offset 0`, `--all` |
| `user search <text>` | `--limit 50`, `--offset 0`, `--all` |

`--limit` must be greater than zero; `--offset` skips items; `--all` fetches all pages with the requested page size.

## Channels and direct messages

`channel` covers public channels and private groups. Channel arguments are names or IDs. `dm` only discovers joined one-to-one conversations; it never creates a room and does not support group DMs.

| Syntax | Flags |
| --- | --- |
| `channel get <name|id>` | No local flags |
| `channel list` | `--public`, `--private`, `--unread`; public/private are mutually exclusive; lists joined channels |
| `channel members <channel>` | `--limit 50`, `--offset 0`, `--all` |
| `channel history <channel>` | `--limit 50`, `--offset 0`, `--since <time>`, `--before <time>` |
| `dm list` | `--all` fetches all pages |
| `dm get <user|room-id>` | Existing conversation by user or room ID |
| `dm history <user|room-id>` | `--limit 50`, `--offset 0`, `--since <time>`, `--before <time>` |

Limits must be greater than zero. Offsets skip members/messages. History time bounds accept RFC3339 timestamps or Go durations such as `2h` or `30m`, interpreted as now minus the duration. `--since` is the oldest bound; `--before` is the latest. History has no `--all` flag.

## Messages

| Syntax | Flags and behavior |
| --- | --- |
| `message get <message-id>` | Inspect one message |
| `message send <target> <text|->` | `:<channel>` targets a channel/group; an unprefixed username, user ID, or email targets a DM user and may create a conversation; bare `:` is invalid |
| `message edit <message-id> <text|->` | Replace message text |
| `message reply <message-id> <text|->` | Resolve root from a root/reply ID; `--also-send` also shows the reply in the main channel (default false) |
| `message thread <message-id>` | Resolve root from a root/reply ID; `--limit 50`, `--offset 0` (replies to skip), `--all` |
| `message search <channel> <text>` | `--limit 50`, `--offset 0` (results to skip) |
| `message delete <message-id>` | `--yes` (`-y`) skips confirmation; otherwise prompts `[y/N]`, accepting only `y` or `yes`, case-insensitively |

Thread/search limits must be greater than zero. Send/edit/reply take exactly two positional arguments. Quote literal text as one shell argument, or pass `-` to read all stdin. Stdin trailing CR/LF is stripped and an empty result is rejected. Search text is a literal positional argument, not the stdin convention.

Inspect before mutation, require explicit deletion intent, and use `--yes` for authorized non-interactive deletion. Inspect after mutations rather than blindly retrying uncertain results.
