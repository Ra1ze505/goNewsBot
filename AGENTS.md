# AGENTS.md

## Cursor Cloud specific instructions

### Product

Single Go binary (`go run ./src/main.go`): Telegram bot (long polling) plus background workers for CBR exchange rates, OpenWeatherMap, Telegram channel ingestion (MTProto), and Yandex Cloud AI summaries. There is no HTTP server and no exposed ports.

### Prerequisites (not in `go mod download`)

- **Go 1.23+** — see `go.mod` / `toolchain` (CI uses 1.23).
- **PostgreSQL** — not bundled in `docker-compose.yml` (compose only builds/runs `app`). Schema: `db/migrations/0001_init.sql`.
- **`.env`** at repo root (gitignored). Required for a full process start:
  - `DATABASE_URL` — Postgres connection string
  - `BOT_TOKEN` — Telegram Bot API
  - `YANDEX_SERVICE_ACCOUNT_KEY_PATH`, `YANDEX_FOLDER_ID`, `YANDEX_ASSISTANT_ID` — ML summarization (`NewMLRepository()` fails without these)
  - `API_ID`, `API_HASH` — Telegram user client for channel fetch (`scripts/auth`)
  - `WEATHER_API_KEY` — optional for weather flows
  - `ADMIN_ID` — optional for `/admin`
- **MTProto session** — `go run ./scripts/auth/main.go --phone <number>` writes `session/telegram-session/` (needed for news pipeline).

### Local PostgreSQL (typical Cloud VM)

If Postgres is not running:

```bash
sudo pg_ctlcluster 16 main start   # or: sudo service postgresql start
```

Example dev DB (adjust credentials as needed):

```bash
export DATABASE_URL="postgres://gonewsbot:gonewsbot_dev@localhost:5432/gonewsbot?sslmode=disable"
psql "$DATABASE_URL" -f db/migrations/0001_init.sql
```

### Commands (see `.github/workflows/go.yml`)

| Task | Command |
|------|---------|
| Download deps | `go mod download` |
| Build | `go build -v ./...` |
| Test (no external services) | `go test -v ./...` |
| Run bot | `go run ./src/main.go` (with `.env` and Postgres up) |
| Docker image | `docker compose up --build` (needs Docker + `.env`; still no Postgres in compose) |

There is **no** golangci-lint or separate lint job in CI; validation is `go build` + `go test`.

### Gotchas

- **`go test ./...` does not need Postgres, Telegram, or Yandex** — repositories use sqlmock; services use gomock.
- **Full `go run ./src/main.go` needs all Yandex env vars** before the bot polls; missing `BOT_TOKEN` yields Telegram API errors immediately.
- **Channel fetcher** needs a valid on-disk session under `session/`; without it, message/summary workers error but the bot may still respond to some commands if DB + token work.
- **Reinstalling Go modules** does not require restarting Postgres; the app has no hot-reload — restart the process after code or `.env` changes.
