# Seyra API (auth)

Local development:

```bash
cd backend
docker compose up -d
copy .env.example .env   # or: cp .env.example .env
go run ./cmd/server
```

Health check: `GET http://127.0.0.1:8080/health`

The Android emulator reaches this host at `http://10.0.2.2:8080`.
