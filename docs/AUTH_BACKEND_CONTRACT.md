# Seyra Authentication Backend Contract

This is the client-facing authentication contract. It is provider-agnostic.
A future backend may be first-party HTTP, gRPC, or another transport as long as
this contract is preserved. Do not couple the auth domain to Firebase, Auth0,
Supabase, or any other vendor.

The first-party Go server implements this contract under `backend/`.
Development may use HTTP on the local machine. Staging and production must use TLS.



## Transport

- Versioned paths under `/v1/auth`.
- JSON request/response bodies unless noted.
- Production and staging **must** use TLS (`https`).
- Development defaults to `http://10.0.2.2:8080` (Android emulator → host).
  Override with `--dart-define=SEYRA_API_BASE_URL=`. Never commit production secrets.
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

### Logout / delete success

Empty body. Delete account also requires `{ "password": "..." }` so the server
can re-authenticate the destructive action.

### Refresh request

```json
{ "refresh_token": "..." }
```

Response shape matches the session response. Previous access credentials must
be treated as revoked after a successful refresh.


## Client-visible vs data-layer-only

**Safe for domain and presentation**

- `user.id`, `user.username`
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
2. Restore uses `GET /v1/auth/session` (or locally stored credentials later).
   Until SecureStorage is wired, restore returns no session.
3. Access credentials expire (`expires_in` / `expires_at`). Refresh obtains a
   new pair; the client must not keep using the old access credential.
4. Logout revokes the current server session. After logout, restore is empty.
5. Account deletion revokes **all** sessions for that user, then deletes server
   account data. The client must later wipe local/secure storage as well.


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

- Passwords travel only on register, login, and account deletion, over TLS.
- The Flutter client never persists passwords.
- The Flutter client never logs passwords or tokens.
- The server **must** hash passwords with a slow password hash. The client
  does not hash passwords as a substitute for TLS or server hashing.
- Access and refresh credentials must eventually be stored in `SecureStorage`,
  never in `LocalStorage`. Storage is **not** implemented in this step.
- E2E private keys are **not** authentication credentials and must never be
  sent on these endpoints.
- Authentication is independent from future E2E message encryption
  (`EncryptionService`). Replacing the auth backend must not require changing
  the crypto domain.
- Account deletion must revoke sessions and remove corresponding server-side
  and local data (local wipe is a later step).


## Client mapping

```
presentation → domain use cases → AuthRepository
data: request/response models + AuthRemoteDataSource
app/DI: composition root (swap the remote adapter without changing domain)
```

A future HTTP/gRPC adapter implements `AuthRemoteDataSource` and is registered
only in `AppDependencies`. Domain and UI stay unchanged.


## Not in this step

- Account-deletion API
- Session route guards
- E2E encryption
