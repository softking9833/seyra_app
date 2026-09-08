# Seyra messaging API contract

1-to-1 messages **may** be Signal-encrypted (`e2e: true`, ciphertext in `body`).
Historical plaintext messages remain plaintext.

**Groups and channels are not E2E.** Public channels stay server-readable for
discovery, moderation, and search. Group `e2e=true` bodies, if ever stored, are
opaque and excluded from plaintext search; the Flutter client does not send them.

**1-to-1 attachments** may be AES-256-GCM ciphertext uploaded with form field
`e2e=true`. File keys travel only inside the Signal payload. Group/channel
`e2e` uploads are rejected. Legacy plaintext attachments remain `e2e=false`.

Bot tokens use `Authorization: Bearer bot_...` (shown once at creation; never
returned again). Built-in commands in a granted room: `/ping`, `/help`, `/whoami`.

## REST

All routes require `Authorization: Bearer <access_token>` (or `bot_…` on bot-capable routes). Sender identity is taken from the session/bot token, never from the client body.

| Operation | Method | Path |
|---|---|---|
| Create or fetch 1-to-1 chat | `POST` | `/v1/chats` |
| Create a group | `POST` | `/v1/chats/groups` |
| Create a channel | `POST` | `/v1/chats/channels` |
| List my chats | `GET` | `/v1/chats` |
| List members | `GET` | `/v1/chats/{chat_id}/members` |
| Add members | `POST` | `/v1/chats/{chat_id}/members` |
| Remove member | `DELETE` | `/v1/chats/{chat_id}/members/{user_id}` |
| Set member role | `POST` | `/v1/chats/{chat_id}/members/{user_id}/role` |
| Leave group/channel | `POST` | `/v1/chats/{chat_id}/leave` |
| List messages (paginated) | `GET` | `/v1/chats/{chat_id}/messages` |
| Send message | `POST` | `/v1/chats/{chat_id}/messages` |
| Soft-delete message | `DELETE` | `/v1/chats/{chat_id}/messages/{message_id}` |

`POST /v1/chats` body: `{ "username": "lin" }` — forbidden if either user blocked the other.

Create a group: `POST /v1/chats/groups` body `{ "title": "Design Team", "usernames": ["lin","maya"] }`.
Create a channel: `POST /v1/chats/channels` body `{ "title": "Seyra News", "usernames": ["lin"], "visibility": "public" }`.
Private is the default. Public channels appear in `GET /v1/channels/discover?q=` and can be joined with `POST /v1/chats/{id}/join`.

| Upload file | `POST` | `/v1/chats/{chat_id}/attachments` (multipart `file`; optional `e2e=true` for 1-to-1 ciphertext) |
| List media | `GET` | `/v1/chats/{chat_id}/attachments` (no `object_key`) |
| Download file | `GET` | `/v1/attachments/{id}` |
| Delete file | `DELETE` | `/v1/attachments/{id}` |
| Stickers | `GET` | `/v1/stickers` (built-in pack) |
| Last seen | `GET` | `/v1/users/{user_id}/last-seen` (1-to-1 + privacy) |
| Read receipts | `GET` | `/v1/chats/{chat_id}/receipts` (1-to-1 + peer privacy) |
| Edit own **plaintext** message | `PATCH` | `/v1/chats/{chat_id}/messages/{message_id}` (`e2e` messages cannot be edited) |
| Toggle reaction | `PUT` | `/v1/chats/{chat_id}/messages/{message_id}/reactions` |
| Search | `GET` | `/v1/search?q=&type=` (`type` optional: users, chats, groups, channels, messages). Message search matches **plaintext only** (`e2e=false`) in rooms the caller can read. User hits honor `profile_visible`. |
| Discover public channels | `GET` | `/v1/channels/discover?q=` |
| User search | `GET` | `/v1/users/search?q=` (same `profile_visible` filter) |
| Privacy | `GET/PUT` | `/v1/privacy` |
| Blocks | `GET/POST /v1/blocks`, `DELETE /v1/blocks/{user_id}` |
| Reports | `POST` | `/v1/reports` |
| Sessions | `GET /v1/auth/sessions`, `DELETE /v1/auth/sessions/{id}` |
| Mute/archive/draft | `PUT .../mute`, `PUT .../archive`, `GET/PUT .../draft` |
| Pins | `GET .../pins`, `POST/DELETE .../messages/{id}/pin` |
| Invites | `POST .../invites`, `POST /v1/invites/join` (blocked vs invite creator → 403) |
| Room meta | `PATCH /v1/chats/{id}` |
| Transfer owner | `POST .../transfer` |
| Restrict member | `POST .../members/{user_id}/restrict` |
| Forward | `POST .../messages/{id}/forward` body `{ "destination_id": "..." }` — plaintext text only; `forwarded_from_id` is stored; E2E forbidden |
| E2E keys | `POST /v1/e2e/keys`, `GET /v1/e2e/bundle/{user_id}` (requires an existing 1-to-1 chat unless fetching self; consumes a one-time prekey only for others), `GET/DELETE /v1/e2e/devices` |
| Calls | `GET /v1/calls/ice`, `POST /v1/calls`, `POST /v1/calls/{id}/signal`, `GET /v1/calls` |
| Bots | `POST/GET /v1/bots`, `DELETE /v1/bots/{id}`, `POST /v1/bots/{id}/grants` |

Send body: `{ "body": "Hello from Seyra", "reply_to_id": "", "attachment_id": "", "e2e": false }`. Max attachment 25 MiB. Allowed types: jpeg, png, webp, gif, pdf, text/plain, mp3, mp4, application/octet-stream. Upload/download JSON does **not** include object keys, disk paths, or file encryption keys.

Development stores blobs under `SEYRA_MEDIA_DIR` (local disk). Production should point the same `media.Store` interface at S3-compatible storage.

The creator of a group/channel is **owner**. Groups need at least one other member. Roles are `owner`, `admin`, and `member`.

- Owner cannot be removed, demoted, or leave (transfer first).
- Owner or admin may add members. A bot with `can_manage_members` may add/remove **members** only.
- Owner may promote/demote admins (`{"role":"admin"|"member"}`).
- Admin may remove members only (not owner/admin).
- Duplicate membership is ignored.
- Channel posts: humans must be admin/owner; bots need `can_send` plus a grant in that channel.

Chat list items include: `id`, `kind` (`direct` | `group` | `channel`), `title`, `visibility`, `member_count`, `peer.id`, `peer.username` (empty for rooms), `last_message_preview` (ciphertext 1-to-1 previews are `"Encrypted message"`), `last_message_at`, `unread_count`.

Message list query: `?before=<message_id>&limit=50` (newest page, returned oldest-first). Bots with `can_read=false` receive 403.

Errors: `401` unauthorized / revoked session, `403` forbidden (including blocks and missing grants), `404` missing resource, `400` invalid input, `413`/invalid type for uploads.

## Realtime

`GET /v1/realtime` — WebSocket upgrade. Bearer token required (header only; tokens must not appear in the URL). First event: `realtime.connected`. Reconnect creates a new hub subscription; the previous connection's unsubscribe runs on close (no duplicate subs on one socket). After leave/kick, further room events are not published to that user because publish uses current member IDs.

Inbound JSON may include `call.signal`. A new call is created only for `offer` (or empty action) with `kind` `voice`|`video` and `conversation_id`. ICE/answer without `call_id` is rejected. ICE/answer on ended calls is forbidden.

Additional events: `message.created`, `message.deleted`, `message.updated`, `message.pinned`, `message.unpinned`, `member.joined`, `member.left`, `member.updated`, `call.signal`, `typing` (respects sender `typing_visible`).
