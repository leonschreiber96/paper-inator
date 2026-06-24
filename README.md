# paper-inator

Self-hosted academic-publication aggregator. It polls RSS/Atom feeds, normalizes
and deduplicates publications into a single SQLite database, and serves a web UI
and REST API to browse and manage them.

Everything ships as **one static binary**: the background ingestion worker, the
REST API, and the web frontend all run from the same executable. Deployment is
"copy the binary to a server and run it."

## Status

This is the **foundation milestone**: project structure, database + migrations,
feeds CRUD, publication ingestion (fetch → parse → map → dedup → store), a web
shell, and a REST API. Email summaries are configurable and persisted, but
delivery/scheduling is not yet implemented (see `src/serviceWorker/summary.go`).

## Build & run

Requires Go 1.26+. No C compiler needed (pure-Go SQLite).

```sh
make build          # produces ./paper-inator
./paper-inator      # starts on :8080, DB at ./paper-inator.db

# or run from source
make run

make test           # run the test suite
make cross          # build Linux amd64 + arm64 binaries
```

Then open <http://localhost:8080>.

## Configuration

Flags (with environment-variable fallbacks):

| Flag               | Env var                      | Default              | Purpose                          |
| ------------------ | ---------------------------- | -------------------- | -------------------------------- |
| `--db`             | `PAPERINATOR_DB`             | `./paper-inator.db`  | SQLite database file path        |
| `--addr`           | `PAPERINATOR_ADDR`           | `:8080`              | HTTP listen address              |
| `--fetch-interval` | `PAPERINATOR_FETCH_INTERVAL` | `15m`                | default feed poll interval       |

SMTP settings (for the upcoming email-summary feature) are read from the
environment only: `PAPERINATOR_SMTP_HOST`, `PAPERINATOR_SMTP_PORT`,
`PAPERINATOR_SMTP_USER`, `PAPERINATOR_SMTP_PASS`, `PAPERINATOR_SMTP_FROM`.

## REST API

| Method & path                     | Description                                  |
| --------------------------------- | -------------------------------------------- |
| `GET /api/health`                 | Liveness check                               |
| `GET /api/feeds`                  | List feeds                                   |
| `POST /api/feeds`                 | Create a feed                                |
| `GET /api/feeds/{id}`             | Get one feed                                 |
| `PUT /api/feeds/{id}`             | Update a feed                                |
| `DELETE /api/feeds/{id}`          | Delete a feed (cascades mappings + pubs)     |
| `GET /api/feeds/{id}/mappings`    | List a feed's field mappings                 |
| `PUT /api/feeds/{id}/mappings`    | Replace a feed's field mappings              |
| `GET /api/publications`           | List publications (`feed_id`,`q`,`sort`,`order`,`limit`,`offset`) |
| `GET/POST /api/summaries`         | List / create email summaries                |
| `PUT/DELETE /api/summaries/{id}`  | Update / delete a summary                    |
| `GET/PUT /api/settings/{key}`     | Read / write a key/value setting             |

## Project layout

```
main.go                  unified entrypoint (worker + server)
src/serviceWorker/       feed fetch, parse, field mapping, dedup, summaries
src/api/                 REST handlers + static-asset serving
src/frontend/            embedded HTML/CSS/JS web UI
src/shared/              models, validation, config, and the SQLite store
```

## Deploying behind nginx

Run the binary (e.g. via systemd) on a private port and reverse-proxy to it:

```nginx
server {
    listen 80;
    server_name papers.example.org;

    location / {
        proxy_pass         http://127.0.0.1:8080;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
    }
}
```

A minimal systemd unit:

```ini
[Unit]
Description=paper-inator
After=network.target

[Service]
ExecStart=/opt/paper-inator/paper-inator --db /var/lib/paper-inator/paper-inator.db
Restart=on-failure

[Install]
WantedBy=multi-user.target
```
