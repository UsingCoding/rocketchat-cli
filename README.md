# rocketchat-cli

A shell- and agent-friendly CLI for Rocket.Chat.

The CLI intentionally presents a small normalized model instead of mirroring every Rocket.Chat REST endpoint. In particular, `channel` covers both public channels and private groups, while the client resolves Rocket.Chat's separate `channels.*` and `groups.*` APIs internally.

## MVP1

```text
rocketchat
├── context
│   ├── list
│   ├── current
│   ├── get
│   ├── set
│   ├── use
│   └── delete
├── auth
│   ├── login
│   ├── logout
│   └── status
├── me
├── user
│   ├── get
│   ├── list
│   └── search
├── channel
│   ├── get
│   ├── list
│   ├── members
│   └── history
├── message
│   ├── get
│   ├── send
│   ├── reply
│   ├── thread
│   ├── search
│   └── delete
└── skill
```

## Build

This repository uses `mise`:

```bash
mise install
mise run test
mise run build
```

The binary is written to `bin/rocketchat`.

You can also use plain Go:

```bash
go build -o rocketchat .
```

## Contexts

Contexts are Rocket.Chat server/auth combinations, similar to kubeconfig contexts.

Create one:

```bash
rocketchat context set company \
  --url https://chat.company.local \
  --user-id "$ROCKETCHAT_USER_ID" \
  --token "$ROCKETCHAT_TOKEN"
```

Or use interactive login, which verifies the credentials before saving them:

```bash
rocketchat auth login company
```

List/switch contexts:

```bash
rocketchat context list
rocketchat context use company
rocketchat context current
```

Use another context for one invocation:

```bash
rocketchat --context test channel list
```

Config defaults to:

```text
~/.config/rocketchat-cli/config.toml
```

or `$XDG_CONFIG_HOME/rocketchat-cli/config.toml` when `XDG_CONFIG_HOME` is set.

Example:

```toml
current_context = "company"

[contexts.company]
url = "https://chat.company.local"
user_id = "abc123"
token = "..."

[contexts.test]
url = "https://chat-test.company.local"
user_id = "def456"
token = "..."
```

The file is written with mode `0600`. For CI/temporary credentials, environment variables are preferred:

```bash
export ROCKETCHAT_CONTEXT=company
export ROCKETCHAT_URL=https://chat.company.local
export ROCKETCHAT_USER_ID=abc123
export ROCKETCHAT_TOKEN=secret
```

Precedence is:

```text
CLI flags > environment > selected context
```

## Users

```bash
rocketchat me
rocketchat user get vadim
rocketchat user get user-id-here
rocketchat user get vadim@example.com
rocketchat user search vadim
rocketchat user list --all
```

`user search` uses Rocket.Chat's Directory API rather than the deprecated unsafe Mongo-style `users.list?query=...` parameter.

## Channels

Both public channels and private groups use the same CLI resource:

```bash
rocketchat channel list
rocketchat channel list --private
rocketchat channel list --unread
rocketchat channel get platform
rocketchat channel members platform --all
```

History supports RFC3339 timestamps or relative durations:

```bash
rocketchat channel history platform --limit 50
rocketchat channel history platform --since 2h
rocketchat channel history platform --since 2026-09-14T10:00:00Z
rocketchat channel history platform --before 30m
```

Message IDs are always shown in tabular history/search output so the result can immediately be used by message commands.

## Messages

Send to a public channel or private group by prefixing the target with `:`:

```bash
rocketchat message send :platform "Deployment completed"
```

An unprefixed target is a direct message recipient. Usernames and email addresses are supported:

```bash
rocketchat message send vadim "Deployment completed"
rocketchat message send vadim@example.com "Deployment completed"
```

For generated direct-message text:

```bash
cat report.txt | rocketchat message send vadim -
```

Edit a message with literal or stdin replacement text:

```bash
rocketchat message edit CeXwh5eBbdrtvnqG6 "Deployment completed successfully"
cat corrected-message.txt | rocketchat message edit CeXwh5eBbdrtvnqG6 -
```

Get/search:

```bash
rocketchat message get CeXwh5eBbdrtvnqG6
rocketchat message search platform "redis OOM"
```

### Replying

Reply to a root message:

```bash
rocketchat message reply CeXwh5eBbdrtvnqG6 "Checked; looks good"
```

The same command can receive the ID of an existing reply. The CLI fetches that message, reads its `tmid`, and sends the new reply to the real thread root. Callers never need to resolve thread roots themselves.

To additionally show the reply in the main channel:

```bash
rocketchat message reply CeXwh5eBbdrtvnqG6 \
  "Important update" \
  --also-send
```

Multiline/generated reply:

```bash
some-command | rocketchat message reply CeXwh5eBbdrtvnqG6 -
```

Show a thread using either its root ID or any reply ID:

```bash
rocketchat message thread CeXwh5eBbdrtvnqG6
rocketchat message thread CeXwh5eBbdrtvnqG6 --all
```

Delete:

```bash
rocketchat message delete CeXwh5eBbdrtvnqG6
rocketchat message delete CeXwh5eBbdrtvnqG6 --yes
```

## Structured output

Human-readable tables/details are the default.

Normalized JSON:

```bash
rocketchat channel history platform --json
```

The JSON schema belongs to `rocketchat-cli`, not Rocket.Chat, so scripts do not depend on fields such as `_id`, `rid`, or `t`.

For debugging the server API response:

```bash
rocketchat channel history platform --raw
```

`--raw` is the body of the final Rocket.Chat API request made by the command. For commands that auto-page with `--all`, normalized `--json` should be preferred because raw output represents only the final API page.

## Agent skill

The binary embeds a concise skill describing the safe/agent-friendly command surface:

```bash
rocketchat skill
```

For example:

```bash
mkdir -p ~/.config/opencode/skills/rocketchat-cli
rocketchat skill > ~/.config/opencode/skills/rocketchat-cli/SKILL.md
```

Agents should normally use `--json`.

## Exit codes

```text
0  success
1  generic error
2  invalid CLI usage
3  authentication error
4  not found
5  permission denied
6  conflict
7  network/server connection error
```

## Release

GoReleaser is configured:

```bash
mise run release
```

After GitHub releases are published, the repository can be consumed through mise's GitHub backend, for example:

```bash
mise use -g github:UsingCoding/rocketchat-cli
```

## API choices

The implementation intentionally uses a small direct HTTP client rather than a large Rocket.Chat Go SDK.

- `rooms.info` resolves human channel names/IDs and tells the service whether to use public-channel or private-group endpoints.
- `rooms.get` lists joined public/private rooms.
- `channels.history` / `groups.history` are hidden behind `channel history`.
- `channels.members` / `groups.members` are hidden behind `channel members`.
- `chat.getMessage` + `chat.sendMessage` implement smart replies.
- `dm.create` + `chat.sendMessage` implement direct messages.
- `chat.getMessage` + `chat.update` implement message edits.
- `chat.getThreadMessages` implements threads.
- `chat.search` implements per-channel message search.
- `directory?type=users&text=...` implements user search without deprecated unsafe query syntax.

The next logical versions are files/reactions, then administrative mutation commands.
