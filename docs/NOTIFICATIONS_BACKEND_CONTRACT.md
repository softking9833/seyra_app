# Notifications backend contract

Seyra does not embed a push vendor in domain code. The API stores device tokens
and preferences, then dispatches through a `Sender` interface.

Production OS push (FCM / APNs) is enabled by pointing `SEYRA_PUSH_WEBHOOK_URL`
at a small gateway you operate. Tokens and webhook secrets must not be committed.

While the app process is running, Flutter also presents a local OS notification
from realtime `message.created` when that conversation is not open.

## Endpoints

| Operation | Method | Path | Auth |
|---|---|---|---|
| Register device | POST | `/v1/notifications/devices` | Access token |
| Unregister device | DELETE | `/v1/notifications/devices/{device_id}` | Access token |
| Get preferences | GET | `/v1/notifications/preferences` | Access token |
| Update preferences | PUT | `/v1/notifications/preferences` | Access token |

### Register device

```json
{ "platform": "android", "token": "…" }
```

`platform`: `android` | `ios` | `web` | `dev`

Response `201`:

```json
{ "id": "dev_…", "platform": "android" }
```

The token is never returned.

### Preferences

```json
{
  "messages_enabled": true,
  "calls_enabled": true,
  "show_preview": true
}
```

When `show_preview` is false, dispatched alerts use a generic body (`New message`).

## Dispatch

On `POST /v1/chats/{id}/messages`, other members with `messages_enabled` receive:

1. Existing realtime `message.created` (if connected)
2. A push `Sender.Send` call per registered device (webhook if configured)

Account deletion cascades device rows.

## Errors

Standard `{ "error": { "code", "message" } }`. Codes: `unauthorized`, `invalid_input`, `not_found`.
