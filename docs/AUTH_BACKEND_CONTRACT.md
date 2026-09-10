# Seyra Authentication Backend Contract

This is the client-facing authentication contract. It is provider-agnostic.
A future backend may be first-party HTTP, gRPC, or another transport as long as
this contract is preserved. Do not couple the auth domain to Firebase, Auth0,
Supabase, or any other vendor.

The first-party Go server implements this contract under `backend/`.
Development may use HTTP on the local machine. Staging and production must use TLS.


## Transport

- Versioned paths under `/v1/auth` and `/v1/users`.
- JSON request/response bodies unless noted.
- Production and staging **must** use TLS (`https`).
- Development defaults to `http://10.0.2.2:8080` (Android emulator → host).
  Override with `--dart-define=SEYRA_API_BASE_URL=`. Never commit production secrets.
- Flutter production/staging builds (`SEYRA_ENV=staging|production`) refuse HTTP
  base URLs at startup.
- Error bodies:

```json
{ "error": { "code": "invalid_credentials", "message": "Invalid username or password" } }
```

Codes: `invalid_input`, `invalid_credentials`, `username_taken`, `unauthorized`, `session_expired`.
Messages must never include passwords or tokens.


## Endpoints

| Operation | Method | Path | Auth |
|---|---|---|---|
| Register | `POST` | `/v1/auth/register` | None |
| Login | `POST` | `/v1/auth/login` | None |
| Logout / revoke | `POST` | `/v1/auth/logout` | Access credential |
| Current session | `GET` | `/v1/auth/session` | Access credential |
| Refresh | `POST` | `/v1/auth/refresh` | Refresh credential |
| Delete account | `POST` | `/v1/auth/account/delete` | Access credential + password |
| Current profile | `GET` | `/v1/users/me` | Access credential |
| Search usernames | `GET` | `/v1/users/search?q=` | Access credential |

### Register / login request (password is sent only here and on account deletion)

```json
{ "username": "ada", "password": "..." }
```

### Session response

```json
{
  "user": { "id": "usr_...", "username": "ada" },
  "session": { "id": "ses_...", "expires_at": "2026-09-07T12:00:00.000Z" },
  "credentials": {
    "access_token": "...",
    "refresh_token": "...",
    "token_type": "Bearer",
    "expires_in": 3600
  }
}
```

`GET /v1/auth/session` returns `user` and `session` only (no credentials).

### Current profile (`GET /v1/users/me`)

Returns only client-safe fields:

```json
{
  "id": "usr_...",
  "username": "ada",
  "display_name": "Ada",
  "bio": "…",
  "has_avatar": false,
  "created_at": "2026-09-07T12:00:00.000Z"
}
```

Must **not** include password hashes, access tokens, refresh tokens, session
HMAC secrets, avatar object keys, or other internal security fields.

`PATCH /v1/users/me` `{ "display_name", "bio" }` updates the same public profile.

`PATCH /v1/users/me/username` `{ "username" }` returns `409 username_taken` when taken.

`POST /v1/users/me/avatar` multipart `file` (image). `DELETE /v1/users/me/avatar`.
`GET /v1/users/{user_id}/avatar` is authorized; honors `photo_visible`.

`POST /v1/auth/sessions/others` revokes every session except the caller (`204`).
`GET /v1/auth/sessions` includes `current`, `user_agent`, timestamps — never tokens.

`PUT /v1/privacy` also accepts `photo_visible`.

### Username search (`GET /v1/users/search?q=`)

Authenticated prefix search on usernames (case-insensitive, 1–32
`[A-Za-z0-9_]` characters). Returns at most 20 matches. The caller is omitted.
Each item is only `{ "id", "username" }`.


### Logout success

Empty body (`204`).

### Delete account

Request (re-authentication):

```json
{ "password": "..." }
```

Success: empty body (`204`). The server must:

1. Authenticate the access credential.
2. Verify the submitted password against the stored Argon2id hash.
3. In a database transaction, delete all sessions for that user, then delete
   the user row.
4. Never log the password.

After success, the client must wipe locally stored access and refresh tokens
and return to the unauthenticated Welcome/Login flow.

### Refresh request

```json
{ "refresh_token": "..." }
```

Response shape matches the session response. Previous access credentials must
be treated as revoked after a successful refresh.


## Client-visible vs data-layer-only

**Safe for domain and presentation**

- `user.id`, `user.username`
- Profile `id`, `username`, `created_at`
- `session.id` (opaque identifier, not a secret)
- `session.expires_at`

**Data layer only — never widgets, logs, or crash reports**

- `credentials.access_token`
- `credentials.refresh_token`
- Passwords (wire-only, never persisted on device)
- Authorization headers

Passwords **must not** appear on `User` or `AuthSession`.


## Session semantics

1. Login and register return a session plus credentials.
2. Restore uses stored credentials with `GET /v1/auth/session`. Access and
   refresh tokens live in `SecureStorage`, never on `AuthSession` or in UI.
3. Access credentials expire (`expires_in` / `expires_at`). Refresh obtains a
   new pair; the client must not keep using the old access credential.
4. Logout revokes the current server session. After logout, restore is empty.
5. Account deletion revokes **all** sessions for that user (by deleting session
   rows), then deletes the user. The client wipes SecureStorage on success.


## Error categories

| Server / transport | Domain failure |
|---|---|
| Backend not connected / 501 | `AuthUnavailableFailure` |
| 401 invalid username/password | `InvalidCredentialsFailure` |
| 409 username taken | `UsernameTakenFailure` |
| 401 expired access credential | `SessionExpiredFailure` |
| 401/403 missing or invalid credential | `UnauthorizedFailure` |
| Transport failure | `NetworkFailure` |
| Anything else | `UnexpectedFailure` |

Error bodies must never include passwords or tokens.


## Security rules

- Passwords travel only on register, login, and account deletion, over TLS
  (HTTP is allowed only in local development).
- The Flutter client never persists passwords.
- The Flutter client never logs passwords or tokens.
- The server hashes passwords with Argon2id. The client does not hash
  passwords as a substitute for TLS or server hashing.
- Access and refresh credentials are stored in `SecureStorage`, never in
  `LocalStorage`.
- E2E private keys are **not** authentication credentials and must never be
  sent on these endpoints.
- Authentication is independent from future E2E message encryption
  (`EncryptionService`). Replacing the auth backend must not require changing
  the crypto domain.


## Client mapping

```
presentation → domain use cases → AuthRepository / AccountRepository
data: request/response models + AuthRemoteDataSource
app/DI: composition root (swap the remote adapter without changing domain)
```

`HttpAuthRemoteDataSource` implements `AuthRemoteDataSource` and is registered
only in `AppDependencies`. Domain and UI stay unchanged.


## Not in this step

- Session route guards
- E2E encryption
- Chat backend
