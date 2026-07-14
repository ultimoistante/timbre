# Timbre

A self-hosted, single-binary music streaming server. Index your music library,
browse it by album/artist/genre, manage playlists, save web radio stations, and
stream everything from a built-in web player — or from any open-standard player
app via the **OpenSubsonic API**.

Timbre is multi-user: every account owns a fully isolated media library, with an
admin role for user management.

---

## Features

- **Library indexing** — recursive scan with metadata extraction (title, album,
  artist, genre, track/disc number, year, duration, bitrate, …) and incremental
  re-scans (unchanged files are skipped).
- **Derived browsing** — albums and artists are computed from track tags on the
  fly; no separate catalog to maintain.
- **On-the-fly transcoding** — stream the original file (with HTTP Range / seek)
  or transcode to `mp3`/`aac`/`ogg`/`opus`/`flac` at a chosen bitrate via
  `ffmpeg`. No temp files: ffmpeg output is piped straight to the response.
- **Built-in web player** — a SvelteKit SPA embedded in the binary (home,
  library, playlists, search, files, settings, admin).
- **Playlists** — create, edit, pin, reorder, add/remove tracks.
- **Web radio** — save internet radio stations, probe stream metadata, proxy and
  optionally re-encode the upstream to MP3, with ICY "now playing" via SSE.
- **Tag editing** — edit a single track's tags or bulk-edit a whole album;
  changes are written back to the files.
- **Album art** — extracted from embedded pictures and cached on disk; an iTunes
  art search/save flow for missing covers.
- **File manager** — browse/upload/download/rename/move/copy/delete within your
  isolated media root.
- **OpenSubsonic API** — connect Symfonium, substreamer, Feishin, Amperfy, DSub,
  Tempo and other open-standard clients (see [below](#opensubsonic-api)).
- **Multi-user + admin** — per-user libraries, quotas, and an admin panel.
- **Single binary** — frontend is embedded; SQLite by default, PostgreSQL
  optional.

---

## Tech stack

| Layer      | Choice                                                         |
|------------|----------------------------------------------------------------|
| Language   | Go 1.26                                                        |
| HTTP       | [Echo v4](https://echo.labstack.com/)                          |
| ORM / DB   | [GORM](https://gorm.io/) over SQLite (default) or PostgreSQL   |
| Tags       | [`go.senan.xyz/taglib`](https://go.senan.xyz/taglib) (read/write), [`dhowden/tag`](https://github.com/dhowden/tag) (cover art) |
| Transcode  | `ffmpeg` (external, invoked at runtime)                        |
| Auth       | JWT access/refresh tokens (header or HttpOnly cookie)          |
| Frontend   | [SvelteKit](https://kit.svelte.dev/) (Svelte 5), static adapter, embedded via `go:embed` |

---

## Requirements

- **Go** ≥ 1.26 (to build the backend)
- **Node.js** + npm (to build the frontend)
- **ffmpeg** on `PATH` (required for transcoding and radio re-encoding;
  original-file streaming works without it)

No system `taglib` is required — tag reading/writing is handled by a pure-Go
library.

---

## Quick start

```bash
# 1. Build everything (frontend → embed → backend)
make build

# 2. Run
TIMBRE_DATA_DIR=./data ./bin/timbre-server
# → timbre listening on 0.0.0.0:8080 (data dir: ./data, db: sqlite)
```

Open <http://localhost:8080>. On first launch you'll be guided through
**onboarding** to create the first user (an admin).

Then:

1. Upload or place your audio files under the user's media root
   (`<TIMBRE_DATA_DIR>/users/<id>/`), or use the **Files** page to upload.
2. Trigger a **scan** (the app does this from the UI; scan progress streams over
   SSE).
3. Browse your **Library**, build **Playlists**, add **Streams**, and play.

---

## Configuration

All configuration is via environment variables.

| Variable              | Default     | Description                                                        |
|-----------------------|-------------|--------------------------------------------------------------------|
| `TIMBRE_HOST`             | `0.0.0.0`   | Bind address.                                                      |
| `TIMBRE_PORT`             | `8080`      | Listen port.                                                       |
| `TIMBRE_DATA_DIR`         | `./data`    | Root for DB, per-user media (`<dir>/users/<id>`), JWT secret, art cache. |
| `TIMBRE_DB_DRIVER`        | `sqlite`    | `sqlite` or `postgres`.                                            |
| `TIMBRE_DB_DSN`           | *(empty)*   | DB connection string. For SQLite, empty ⇒ `<TIMBRE_DATA_DIR>/timbre.db`. |
| `TIMBRE_ACCESS_TTL_MIN`   | `30`        | Access-token lifetime (minutes).                                  |
| `TIMBRE_REFRESH_TTL_DAYS` | `30`        | Refresh-token lifetime (days).                                    |

The JWT signing secret is auto-generated at `<TIMBRE_DATA_DIR>/jwt.secret` on first
run (32 random bytes, hex-encoded).

**PostgreSQL example:**

```bash
TIMBRE_DB_DRIVER=postgres \
TIMBRE_DB_DSN="host=localhost user=timbre password=secret dbname=timbre sslmode=disable" \
./bin/timbre-server
```

---

## Deployment (Proxmox guest)

Two ways to run Timbre on a Proxmox guest (LXC container or VM): **Docker**
(isolated, easiest updates) or **native** (no container runtime, runs as a
system service). Pick your guest OS below — both are covered.

> **LXC + Docker note:** Docker needs an unprivileged LXC container with
> nesting enabled, or a VM. On the Proxmox host:
> `pct set <vmid> --features nesting=1,keyctl=1`, then reboot the container.
> If you'd rather avoid this entirely, use the native install below — an LXC
> container is fine for it as-is.

### Alpine Linux

#### Option A — Docker

**Install**

```bash
# On the Alpine guest, as root:
apk update
apk add docker docker-cli-compose git
rc-update add docker boot
service docker start

git clone <your-repo-url> /opt/timbre
cd /opt/timbre
docker compose up -d --build
```

Timbre listens on `:8080` (mapped in `docker-compose.yml`); data (DB, media,
JWT secret) lives in the `music_data` named volume. Edit the `environment:`
block in `docker-compose.yml` first if you want Postgres or a different port —
see [Configuration](#configuration).

**Update**

```bash
cd /opt/timbre
git pull
docker compose up -d --build   # rebuilds the image, recreates the container; the volume is untouched
docker image prune -f          # optional: drop the old image layers
```

#### Option B — Native (OpenRC service)

**Install**

```bash
# On the Alpine guest, as root:
apk update
apk add ffmpeg ca-certificates tzdata go nodejs npm git

# Check the Go version matches go.mod's requirement (go 1.26+); if the apk
# package is older, install the toolchain from https://go.dev/dl instead.
go version

git clone <your-repo-url> /opt/timbre-src
cd /opt/timbre-src
make build   # frontend build → embed → CGO_ENABLED=0 go build

# Install the binary + create a dedicated user and data dir.
addgroup -S timbre 2>/dev/null; adduser -S -D -H -G timbre timbre 2>/dev/null
install -d -o timbre -g timbre /var/lib/timbre
install -d -o timbre -g timbre /var/log/timbre
mkdir -p /opt/timbre
cp bin/timbre-server /opt/timbre/timbre-server

# OpenRC service + config (shipped in this repo under contrib/alpine/).
cp contrib/alpine/timbre.initd /etc/init.d/timbre
chmod +x /etc/init.d/timbre
cp contrib/alpine/timbre.conf.d /etc/conf.d/timbre

rc-update add timbre default
rc-service timbre start
rc-service timbre status
```

Edit `/etc/conf.d/timbre` to change data dir, port, or switch to Postgres — see
[Configuration](#configuration) for all variables. Logs land in
`/var/log/timbre/timbre.log` / `timbre.err`.

**Update**

```bash
cd /opt/timbre-src
git pull
make build
rc-service timbre stop
cp bin/timbre-server /opt/timbre/timbre-server
rc-service timbre start
```

The data directory (`/var/lib/timbre` by default) is untouched by an update —
only the binary is replaced.

### Debian

#### Option A — Docker

**Install**

```bash
# On the Debian guest, as root. Debian's own `docker.io` package is often
# quite old; the official install script below tracks current Docker releases.
apt update
apt install -y ca-certificates curl git
curl -fsSL https://get.docker.com | sh   # installs docker-ce + the compose plugin

git clone <your-repo-url> /opt/timbre
cd /opt/timbre
docker compose up -d --build
```

Timbre listens on `:8080` (mapped in `docker-compose.yml`); data (DB, media,
JWT secret) lives in the `music_data` named volume. Edit the `environment:`
block in `docker-compose.yml` first if you want Postgres or a different port —
see [Configuration](#configuration).

**Update**

```bash
cd /opt/timbre
git pull
docker compose up -d --build   # rebuilds the image, recreates the container; the volume is untouched
docker image prune -f          # optional: drop the old image layers
```

#### Option B — Native (systemd service)

**Install**

```bash
# On the Debian guest, as root:
apt update
apt install -y ffmpeg ca-certificates tzdata git

# Debian stable's `golang-go`/`nodejs` packages usually lag behind go.mod's
# requirement (go 1.26+) and the frontend's needs. Prefer the upstream
# toolchains: https://go.dev/dl and https://github.com/nodesource/distributions
go version
node --version

git clone <your-repo-url> /opt/timbre-src
cd /opt/timbre-src
make build   # frontend build → embed → CGO_ENABLED=0 go build

# Install the binary + create a dedicated system user and data dir.
useradd --system --home-dir /var/lib/timbre --shell /usr/sbin/nologin timbre
install -d -o timbre -g timbre /var/lib/timbre
mkdir -p /opt/timbre /etc/timbre
cp bin/timbre-server /opt/timbre/timbre-server

# systemd unit + env file (shipped in this repo under contrib/debian/).
cp contrib/debian/timbre.service /etc/systemd/system/timbre.service
cp contrib/debian/timbre.env /etc/timbre/timbre.env

systemctl daemon-reload
systemctl enable --now timbre
systemctl status timbre
```

Edit `/etc/timbre/timbre.env` to change data dir, port, or switch to Postgres —
see [Configuration](#configuration) for all variables. Logs go to the journal:
`journalctl -u timbre -f`.

**Update**

```bash
cd /opt/timbre-src
git pull
make build
systemctl stop timbre
cp bin/timbre-server /opt/timbre/timbre-server
systemctl start timbre
```

The data directory (`/var/lib/timbre` by default) is untouched by an update —
only the binary is replaced.

---

## OpenSubsonic API

Timbre exposes an [OpenSubsonic](https://opensubsonic.netlify.app/)-compatible
API under `/rest`, so third-party player apps can connect to your library.

### Connecting a client

1. Sign in to the web UI and open **Settings → External player apps**.
2. Click **Generate token**. You'll get:
   - **Server URL** — e.g. `http://your-host:8080`
   - **Username** — your account username
   - **Token** — a long random secret
3. In your player app, enter the server URL and username, and use the **token as
   the password**. Most clients append `/rest` automatically.

Tested against substreamer, Feishin, Symfonium, DSub and Tempo.

**Easier entry on mobile.** The random token is long to type. The Settings page
offers two shortcuts:

- **Show QR** — renders the token as a QR code; scan it with your phone to copy
  the token to the clipboard, then paste it into the app's password field.
- **Set a custom password** — replace the token with your own memorable
  passphrase (min 8 characters). It works exactly like the token across all auth
  schemes — handy for clients without copy/paste.

### Authentication

The Subsonic token is a **per-user, revocable secret** stored separately from
your account password (which is bcrypt-hashed and never used here). It supports
all three Subsonic auth schemes, so both old and new clients work:

| Scheme                 | Request parameters                          |
|------------------------|---------------------------------------------|
| OpenSubsonic API key   | `apiKey=<token>`                            |
| Token + salt (legacy)  | `u=<user>&t=md5(<token>+<salt>)&s=<salt>`   |
| Password (legacy)      | `u=<user>&p=<token>` or `p=enc:<hex>`       |

Regenerating the token invalidates the old one immediately; revoking disables
external access until you generate a new one. Manage this any time from the
Settings page, or via the native API:

```
GET    /api/me/subsonic-token   # show current (404 if none)
POST   /api/me/subsonic-token   # generate / rotate a random token
PUT    /api/me/subsonic-token   # set a custom password  {"token":"..."}
DELETE /api/me/subsonic-token   # revoke
```

### Implemented endpoints

Responses are XML by default, or JSON/JSONP with `f=json` / `f=jsonp`. Each
method is reachable both as `<name>` and `<name>.view`, over GET or POST.

- **System** — `ping`, `getLicense`, `getOpenSubsonicExtensions`, `getUser`,
  `getScanStatus`, `startScan`
- **Browsing** — `getMusicFolders`, `getArtists`, `getIndexes`, `getArtist`,
  `getAlbum`, `getSong`, `getGenres`, `getMusicDirectory`
- **Lists** — `getAlbumList`, `getAlbumList2`, `getRandomSongs`, `getNowPlaying`
- **Search** — `search2`, `search3`
- **Streaming** — `stream` (with `maxBitRate` / `format` transcoding),
  `download`, `getCoverArt`
- **Playlists** — `getPlaylists`, `getPlaylist`, `createPlaylist`,
  `updatePlaylist`, `deletePlaylist`
- **Annotation** — `scrobble`, `star`, `unstar`, `setRating`, `getStarred`,
  `getStarred2`

Quick check with `curl`:

```bash
TOKEN=...   # from Settings → External player apps
curl "http://localhost:8080/rest/ping.view?apiKey=$TOKEN&v=1.16.1&c=test&f=json"
# → {"subsonic-response":{"status":"ok","openSubsonic":true,...}}
```

---

## Native API overview

The web UI talks to a JSON API under `/api` (JWT-authenticated unless noted).

- **Auth** — `POST /api/onboarding`, `POST /api/auth/login`,
  `POST /api/auth/refresh`, `POST /api/auth/logout`, `GET /api/me`,
  `GET /api/onboarding` (public), `GET /api/healthz` (public)
- **Library** — `GET /api/tracks`, `GET /api/albums`, `GET /api/albums/:hash`,
  `GET /api/artists`, `GET /api/search`, `GET /api/recently-added`
- **Metadata** — `PATCH /api/tracks/:id`, `PATCH /api/albums/:hash`
- **Album art** — `GET /api/albums/:hash/art`, `GET /api/albums/:hash/art/search`,
  `PUT /api/albums/:hash/art`
- **Playlists** — CRUD under `/api/playlists` (+ `/tracks` sub-routes)
- **Streaming** — `GET /api/stream/:id?quality=<bitrate>&container=<fmt>`
- **Web radio** — `/api/streams*` (+ `probe`, `:id/play`)
- **Scan / events** — `POST /api/scan`, `GET /api/events` (SSE)
- **Files** — `/api/fs/*`, `POST /api/upload`, `GET /api/download`
- **Admin** — `/api/admin/users*` (admin role required)

---

## Development

Run the backend and the Vite dev server separately for hot-reload:

```bash
# Terminal 1 — backend (serves /api on :8080)
TIMBRE_DATA_DIR=./data go run ./cmd/server

# Terminal 2 — frontend dev server (proxies /api to :8080)
cd web && npm run dev   # http://localhost:5173
```

Run tests:

```bash
make test   # go test ./...
```

### Project layout

```
cmd/server/            # entry point
internal/
  api/                 # Echo server, route registry, native /api handlers
    frontend/          # embedded SvelteKit build (generated by `make frontend`)
  subsonic/            # OpenSubsonic /rest adapter (auth, id codec, handlers)
  auth/                # JWT manager, middleware, password hashing
  models/              # GORM models (User, MediaFile, Playlist, RadioStation, Star)
  scanner/             # library scan + tag read/write
  stream/              # audio serving + ffmpeg transcoding
  storage/             # per-user media-root path resolution (traversal-safe)
  config/ db/ events/ fsops/
web/                   # SvelteKit frontend source
```

### Build targets

| Target          | Action                                                    |
|-----------------|-----------------------------------------------------------|
| `make build`    | Build frontend, copy into the embed dir, build backend.   |
| `make frontend` | `npm run build` then copy `web/build` → `internal/api/frontend`. |
| `make backend`  | `go build -o bin/timbre-server ./cmd/server`.             |
| `make run`      | Build backend and run with `TIMBRE_DATA_DIR=./data`.          |
| `make test`     | Run Go tests.                                             |
| `make clean`    | Remove build artifacts, embed dir, and `data/`.           |

---

## Security notes

- **Per-user isolation** — every library row is scoped by user ID, and file
  access is resolved against the user's media root with path-traversal and
  symlink-escape protection.
- **Passwords** — bcrypt-hashed; never stored or transmitted in plaintext.
- **Subsonic token** — stored in plaintext **by design**: it's a revocable
  secret, not your account password, and the Subsonic token-auth scheme requires
  the server to know it verbatim. Treat it like an app password; rotate or revoke
  it from Settings at any time.
- Put Timbre behind a TLS-terminating reverse proxy for any non-local
  deployment — Subsonic legacy auth can carry the token in the URL.

---

## License

No license file is currently included. Add one before distributing.
