# Workflows

Replace example names and IDs with values verified from preceding JSON reads. Mutation examples require the user's intent; reads alone do not authorize writes.

## Select a context and inspect a channel

```sh
rocketchat context list --json
rocketchat context get work --json
rocketchat context use work
rocketchat auth status --json
rocketchat channel list --json
rocketchat channel get general --json
rocketchat channel history general --limit 20 --json
```

For a one-command context override, use `--context work` instead of changing the saved selection.

## Send to a channel or an identified user

Inspect the context and destination first. The colon is essential for channel/group sends:

```sh
rocketchat context current --json
rocketchat channel get general --json
rocketchat message send :general 'Deployment complete' --json
rocketchat message get RETURNED_MESSAGE_ID --json
```

An unprefixed target identifies a user, not a channel or a DM room:

```sh
rocketchat user get alice --json
rocketchat message send alice 'Deployment complete' --json
rocketchat message get RETURNED_MESSAGE_ID --json
```

Sending may create a direct conversation. For generated text, pipe stdin explicitly:

```sh
printf '%s\n' 'Deployment complete' | rocketchat message send :general - --json
```

## Browse an existing DM without creating one

```sh
rocketchat dm list --all --json
rocketchat dm get alice --json
rocketchat dm history alice --limit 20 --since 2h --json
```

Use the inspected room ID in place of `alice` if desired. These commands only discover joined one-to-one conversations. A missing conversation is not permission to create one.

## Read and reply to a thread

```sh
rocketchat message get MESSAGE_ID --json
rocketchat message thread MESSAGE_ID --all --json
printf '%s\n' 'Acknowledged' | rocketchat message reply MESSAGE_ID - --json
rocketchat message thread MESSAGE_ID --all --json
```

Both a root and a reply ID resolve to the root. Add `--also-send` only when the user wants the reply shown in the main channel too.

## Delete only with explicit intent

First inspect the message and confirm its identity, destination, and the user's deletion request:

```sh
rocketchat message get MESSAGE_ID --json
# Only after explicit authorization:
rocketchat message delete MESSAGE_ID --yes --json
rocketchat message get MESSAGE_ID --json
```

The final read should report the deleted message missing. If the mutation outcome is uncertain, inspect state before any retry. Do not expose tokens when reporting results.
