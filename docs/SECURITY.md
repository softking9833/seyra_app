# Seyra security model (Phase 1–5)

This document describes the **actual** security properties of the current
code, not a marketing claim. **Seyra is not fully end-to-end encrypted.**

## Layers

| Layer | What it protects | Status |
|---|---|---|
| Transport (TLS) | Credentials and payloads in transit | TLS **required** when `SEYRA_ENV=production` (`SEYRA_TLS_CERT` / `SEYRA_TLS_KEY`). Development may use HTTP to `10.0.2.2`. |
| Authentication | Who is acting | Access/refresh sessions. Tokens are HMAC'd with `SEYRA_SESSION_PEPPER`. Login is rate-limited (8 failures / 15 minutes per username). Passwords use Argon2id. |
| Authorization | Who may read/write a conversation | **Server-side** membership, roles (`owner` / `admin` / `member`), member send restrictions, blocks, and bot grants. The client is not trusted. |
| Storage (device) | Access/refresh tokens, Signal identity + protocol state | Flutter `SecureStorage`. |
| Server-readable plaintext | Direct, group, and channel bodies when `e2e` is false | The API and PostgreSQL can read these bodies. Historical plaintext is **not** retroactively encrypted. |
| Signal E2E (1-to-1 only) | New direct-message bodies against the server | `libsignal_protocol_dart` (X3DH + Double Ratchet) when both devices have published prekeys. Ciphertext is stored in `messages.body` with `e2e=true`. |
| Attachment AEAD (1-to-1) | File bytes for new direct uploads | AES-256-GCM (`cryptography` package) on the sender device. Ciphertext is uploaded as `application/octet-stream` with `attachments.e2e=true`. The 32-byte file key and nonce travel **inside** the Signal-protected message payload, not as server metadata. |
| Groups / channels | Room bodies and room files | **Not** E2E. Groups and channels remain server-readable plaintext. Encrypted attachment uploads are rejected on non-direct rooms. Public channels are not E2E by product design (discovery, moderation, search). |

## Authentication and sessions

- REST and WebSocket use `Authorization: Bearer` **headers only**. Access tokens must not appear in URLs.
- Revoked or expired sessions are rejected on REST. The realtime socket re-checks the session on inbound frames and on the ping interval and disconnects if the session is gone.
- Account deletion verifies the password, deletes owned bot records and bot user rows, then deletes the user and sessions. Message `sender_id` is set NULL so history can remain for other members. Bot tokens are stored as SHA-256 hashes and are shown **once** at creation.

## Authorization

- Non-members receive a generic forbidden/not-found style failure; they must not receive private message bodies, attachments, pins, or E2E prekeys.
- Public channel discovery lists **public channels only**.
- Profile search (`GET /v1/users/search` and `GET /v1/search?type=users`) omits users with `profile_visible=false`.
- Direct chats and direct messages are blocked when either party has blocked the other. Invite join is refused if the actor is blocked with the invite creator.
- Group/channel membership is not the same as a 1-to-1 block: admins may still add members to rooms.
- Bot grants `can_read`, `can_send`, `can_manage_messages`, and `can_manage_members` are enforced on the server (list/search/download, send/react/upload, deleting others' messages, add/remove members respectively).

## Signal / E2E (1-to-1)

- Private identity keys never leave the device. The server stores identity **public** key, signed prekey public+signature, and one-time prekey **public** values.
- Fetching another user's bundle requires an existing 1-to-1 conversation, is denied if blocked, and consumes one one-time prekey. Fetching **your own** bundle does not consume a prekey.
- If session setup fails, the client may send a normal plaintext **text** message (`e2e` omitted/false). File envelopes that contain attachment keys are **not** posted in plaintext; send fails closed instead.
- Encrypted messages are excluded from server-side body search. The server cannot decrypt E2E bodies.
- Forwarding encrypted messages is forbidden. Forwarding plaintext copies **text only** (not attachment bytes) and records `forwarded_from_id`. Messages that have an `attachment_id` cannot be forwarded.
- Editing an `e2e=true` message is forbidden so decrypted plaintext is not written back to the server.
- Decrypt failure in the client displays the placeholder **"Encrypted message"**. Attachment decrypt failure does not write plaintext files.
- Protocol state (sessions, one-time prekeys, signed prekeys, trusted peer identity publics) is persisted in SecureStorage. Identity keys were already persisted. After app/process/device restart, ratchet state is restored from that blob. If the SecureStorage blob is missing or corrupt, decrypt of existing ciphertext fails and the UI shows "Encrypted message".
- Device id is **`1` only**. This is not a multi-device E2E product. Additional devices are not fully implemented.
- Groups and channels are **not** Signal-encrypted. `libsignal_protocol_dart` has no maintained group/MLS API in this project. Seyra does not invent a group ratchet. The backend may store opaque `e2e=true` **group** bodies (excluded from plaintext search) for a future protocol; the Flutter client never sets `e2e` on group or channel messages. Channel `e2e=true` sends are rejected.
- Stickers are a built-in emoji pack (`GET /v1/stickers`), not a marketplace. 1-to-1 sticker JSON is Signal-wrapped when E2E is available; groups/channels send the emoji as plaintext.

## Attachments

- Max size **25 MiB**. Allowed types: jpeg, png, webp, gif, pdf, text/plain, mp3, mp4, and `application/octet-stream` (used for E2E ciphertext).
- Object keys are generated server-side; path traversal is rejected. HTTP responses do not include filesystem paths, storage credentials, or file encryption keys.
- Download requires conversation membership (and `can_read` for bots). Gallery `GET /v1/chats/{id}/attachments` is membership-gated and omits `object_key`.
- **1-to-1 encrypted attachments:** the client AES-256-GCM-encrypts bytes before upload. The server stores only ciphertext (`e2e=true`, filename `encrypted.bin`). Keys are not in attachment metadata. Recipients decrypt locally after Signal-decrypting the message envelope. Legacy plaintext attachments remain distinguishable (`e2e=false`).
- Encrypted attachments cannot be decrypted from the gallery list alone (no keys there). Open from the chat message on a device that can decrypt the Signal payload.
- Group/channel attachment uploads with `e2e=true` are rejected.

## Calls (WebRTC)

- Signaling uses authenticated REST `/v1/calls` and inbound WebSocket `call.signal`. ICE/TURN credentials are not placed in URLs.
- STUN defaults to a public Google STUN URL. TURN is environment-driven (`SEYRA_TURN_*`) and empty until configured. This pass does not invent TURN credentials.
- Call signaling requires membership. ICE/answer on an already ended/rejected/missed call is rejected.
- Incoming calls prompt accept/decline; media is not captured until the call screen opens after accept.
- Media uses DTLS-SRTP from the WebRTC stack. There is no additional application-layer call E2E.

## Privacy controls (server-enforced where implemented)

| Setting | Enforcement |
|---|---|
| `profile_visible` | Username search |
| `notification_preview` | Push/local preview body (with notification `show_preview`) |
| Blocking | Direct chat create + direct send; invite join vs creator |
| `read_receipts` | `GET /v1/chats/{id}/receipts` returns the peer's `last_read_at` for a 1-to-1 chat only when the **peer** has `read_receipts` enabled. |
| `last_seen_visible` | `GET /v1/users/{id}/last-seen` for a 1-to-1 peer. Hidden when the peer disabled the setting. Not a live online indicator. Presence is updated on authenticated REST (`user_presence`). |
| `photo_visible` | `GET /v1/users/{id}/avatar`. Hidden from others when disabled. Owners can always fetch their own photo. There is no contacts graph; UI exposes Everyone / Nobody only. |
| `typing_visible` | WebSocket inbound `typing` events. If the sender disabled the setting, the server does not broadcast. The chat UI shows a real typing indicator from those events. |

## Logging

HTTP logs record method, path, status, and duration only. Passwords, tokens, private keys, and message bodies are not logged.

## Residual / not claimed

- Compromised client device
- Metadata (membership, timestamps, sizes, who messaged whom)
- Killed-app FCM/APNs until a production push gateway is configured
- Production TURN and S3 (or equivalent) until operators configure them
- Windows plugin symlinks if Developer Mode is off (environment, not an app bug)
- Bot platform: `/ping`, `/help`, `/whoami` in rooms where a granted bot has `can_send`. Not a hosted marketplace. Tokens are never listed.
