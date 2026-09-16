# MVP1 specification

## Product boundary

`rocketchat` is a shell- and agent-friendly CLI over Rocket.Chat REST APIs. It exposes a small, normalized resource model rather than mirroring every server endpoint. Human-readable output is the default; `--json` emits the CLI-owned normalized model and `--raw` emits the final Rocket.Chat response. These output modes remain mutually exclusive.

MVP1 covers authenticated workspace use, user and channel discovery, channel messaging, one-to-one direct-message sending, and editing an existing message. It does not add file uploads, reactions, group DMs, DM browsing/history, room administration, or other administrative mutations.

## Cross-epic contracts

- Commands use the selected context and existing global `--context`, `--url`, `--user-id`, `--token`, `--timeout`, `--json`, and `--raw` flags without new authentication or configuration paths.
- Mutations return the normalized `Message` result. Human output names the operation and prints the message ID; `--json` returns the full normalized message; `--raw` returns the response from the final mutation request.
- `<text|->` accepts one literal argument or reads stdin for `-`. Stdin text has its final CR/LF removed and is rejected when empty, matching existing `message send` and `message reply` behavior.
- Existing error classification is preserved: invalid command input exits 2, authentication 3, not found 4, permission denied 5, conflict 6, and network/server connection errors 7.
- No confirmation is required for send or edit. `message delete` remains the only destructive command requiring confirmation unless `--yes` is passed.

## Epic 1 — Core workspace and channel messaging

### Outcome

A configured user can authenticate, inspect the workspace, find people and joined public/private channels, read and search channel messages, and send, reply to, inspect, and delete channel messages.

### Delivered command surface

```text
rocketchat
├── context list|current|get|set|use|delete
├── auth login|logout|status
├── me
├── user get|list|search
├── channel get|list|members|history
└── message get|send|reply|thread|search|delete
```

### Delivered behavior

- Context credentials resolve with precedence `CLI flags > environment > selected context`; persisted config is written with mode `0600`.
- `user get` resolves username, then ID when the username lookup is not found; emails resolve by email. `user search` uses the Directory API rather than deprecated Mongo-style user query input.
- The `channel` resource normalizes Rocket.Chat public channels (`c`) and private groups (`p`). Name-or-ID lookup and type-specific history/member endpoints are hidden behind one command surface.
- Channel history accepts paging plus RFC3339 or relative-duration `--since` and `--before` bounds. Table history/search output includes message IDs.
- `message send` resolves a public/private channel before sending. `message reply` accepts a root or any existing reply and resolves the true thread root before sending. `message thread` likewise accepts either identifier.
- JSON models expose CLI-owned names such as `id`, `room_id`, `thread_id`, `text`, `author`, and `created_at`; callers do not depend on Rocket.Chat fields such as `_id`, `rid`, or `t`.

### Verification already present

Focused service tests prove private-room history routing and that a reply to an existing reply uses the actual thread root and preserves `--also-send`. API client tests prove authentication headers and normalized not-found detection.

## Epic 2 — Send a direct message to one user

### User outcome

A user can send a one-to-one direct message without knowing, listing, or creating a DM room manually.

### Command contract

```text
rocketchat message send <target> <text|->
```

`target` selects its kind by prefix:

- `:<channel>` sends to a public channel or private group. The CLI removes the one leading `:` and applies the existing channel name-or-ID resolution.
- Any target without `:` is a user identifier: username, user ID, or email.

The leading `:` is shell-safe and does not require quoting:

```bash
rocketchat message send :platform "Deployment completed"
printf 'Build failed\nSee CI.' | rocketchat message send alice@example.com -
rocketchat message send user-id-here "Can you review this?" --json
```

The command must:

1. Accept exactly `<target> <text|->`.
2. If `target` starts with `:`, reject `:` alone with usage exit code 2; otherwise remove the prefix, resolve the remainder as an existing public/private channel, and send using the existing channel path. It must not perform user lookup in this branch.
3. If `target` has no `:` prefix, resolve it with the same lookup rules as `user get` (email; otherwise username then ID).
4. If that user lookup is not found, return a not-found error that says the identifier was not found as a user and advises: `If this is a channel, prefix it with ':', for example: rocketchat message send :<channel> <text>`. It retains exit code 4.
5. For a resolved user, create or retrieve the one-to-one DM using the recipient's **username** via `POST /api/v1/dm.create` with `{"username":"<username>"}`. This API is idempotent for the participant pair: its room response may be newly created or an existing room.
6. Read the returned DM room ID from `room.rid`, then send with the existing `POST /api/v1/chat.sendMessage` flow using that room ID and the supplied text.
7. Return the sent normalized `Message`, including its channel or DM `room_id`.

### Implementation boundary

- Add the API client operation for `dm.create` and a DTO sufficient to read `room.rid`.
- Change `message send` to dispatch by the target's leading `:`: `:<channel>` takes the existing channel path; an unprefixed target takes the direct-message path.
- Keep exactly two positional arguments in either branch. Reject `:` as an empty channel target.
- Add a not-found error wrapper for failed unprefixed user resolution that preserves the existing not-found exit classification and adds the `:` channel-target hint.
- Reuse `readText`, `render`, and error classification. This is an intentional breaking syntax change: existing channel callers must add the `:` prefix.
- Do not implement multiple-recipient DMs, DM list/history/member commands, DM replies/threads, or an explicit open/create-DM command in MVP1.

### Acceptance criteria

- `:platform` resolves `platform` only as a public/private channel and never looks it up as a user; `:` alone returns usage error before requests.
- An unprefixed username, ID, or email identifies the recipient and results in `dm.create` with that recipient's resolved username followed by `chat.sendMessage` to `room.rid`.
- An unprefixed target that is not a user returns exit code 4 with the prescribed `:` channel-target advice, without attempting `dm.create` or `chat.sendMessage`.
- Literal and stdin message bodies follow the existing text contract; empty stdin returns usage error before any request.
- A failure from channel resolution, user lookup, DM creation, or message sending reaches the existing classifier unchanged except for the user-not-found channel hint; no message is attempted after a failed prior step.
- Human, JSON, and raw output follow the cross-epic contract. `--raw` is the response from the final request (`chat.sendMessage`).
- Focused HTTP service tests assert prefix dispatch, request order, request bodies, user-identifier variants, the not-found hint, and failure short-circuiting.

## Epic 3 — Edit a message

### User outcome

A user can replace the text of an existing message by its message ID without supplying a room ID.

### Command contract

```text
rocketchat message edit <message-id> <text|->
```

Examples:

```bash
rocketchat message edit CeXwh5eBbdrtvnqG6 "Corrected deployment window: 16:00 UTC"
cat final-update.txt | rocketchat message edit CeXwh5eBbdrtvnqG6 - --json
```

The command must:

1. Read the replacement text using the existing `<text|->` rule.
2. Fetch the target with `GET /api/v1/chat.getMessage?msgId=<message-id>` to obtain the canonical message ID and room ID.
3. Submit `POST /api/v1/chat.update` with exactly the MVP1 request fields:

   ```json
   {"roomId":"<target.room_id>","msgId":"<target.id>","text":"<replacement text>"}
   ```

4. Return the normalized updated `Message` from the endpoint response.

Server-side authorization determines whether the caller may edit the message; MVP1 does not preemptively check authorship, timestamps, room type, or workspace edit-window settings. The resulting server permission/validation error must remain visible and receive normal exit-code classification.

### Implementation boundary

- Add the API client operation for `chat.update` with room ID, message ID, and replacement text; reuse the existing message DTO.
- Add a service operation that fetches the message and then calls the update operation with its canonical identifiers.
- Add `message edit` to the existing `message` command; reuse `readText`, `render`, and `classify`.
- Do not add interactive editing, patches/diffs, revision history, bulk editing, URL-preview options, or `customFields`. Those are server API capabilities outside MVP1.
- Editing is deliberately allowed for messages in any room type returned by `chat.getMessage`, including a DM created by Epic 2. The command has no channel-name or DM-recipient argument.

### Acceptance criteria

- The CLI performs `chat.getMessage` before `chat.update`, and the update body contains the fetched room ID and canonical message ID, never a caller-supplied room ID.
- Literal and stdin replacement bodies follow the existing text contract; empty stdin returns usage error before any request.
- The updated message returned by the server is rendered as the normalized message in JSON and human output; `--raw` emits the `chat.update` response.
- Lookup failure stops before update; update failures propagate through existing error classification unchanged.
- The service does not reject server-authorized edits based on local authorship or message age.
- Focused HTTP service tests assert lookup/update order, exact update payload, normalized return value, empty-stdin handling, and lookup failure short-circuiting.

## Delivery order

Implement Epic 2 before Epic 3. Epic 3 itself is independent of direct-message creation, but this order proves that the existing normalized `Message` and shared sending path support direct-message IDs before edit support is added.

## MVP1 exit criteria

MVP1 is complete when all three epics meet their acceptance criteria, command help and the embedded agent skill document `message send :<channel>`, unprefixed user targets, and `message edit`, and the focused test suite passes.

## Epic status

- [x] Epic 1 — Core workspace and channel messaging
- [ ] Epic 2 — Send a direct message to one user
- [ ] Epic 3 — Edit a message