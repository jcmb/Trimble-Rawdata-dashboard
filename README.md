# Trimble-Rawdata-dashboard

Live web dashboard for Trimble DCOL **0x57 RAWDATA**: sky plot and SNR from record subtype 6 (RT27), position from subtype 7.

Both **0x40 GSOF** and **0x57 RAWDATA** are Trimble DCOL message types on the same wire — this dashboard consumes **0x57** only (not 0x40).

Uses [geoffrey-kirk-go-dcol](https://github.com/gkirk/geoffrey-kirk-go-dcol) (or a local checkout via `replace` in `go.mod`).

## Quick start

```bash
# Hosted UI — user connects from the browser (public hosts only by default)
go run ./cmd/trimble-rawdata-dashboard

# Allow web UI connections to localhost / private IPs (e.g. tcp://192.168.1.10:5017)
go run ./cmd/trimble-rawdata-dashboard -allow-local-hosts

# UI only, synthetic data at startup (no connect form)
go run ./cmd/trimble-rawdata-dashboard -demo

# Fixed receiver at startup (CLI — not restricted by -allow-local-hosts)
go run ./cmd/trimble-rawdata-dashboard -port 'tcp://sps855.com:28005' -verbose info
go run ./cmd/trimble-rawdata-dashboard -port 'serial:///dev/ttyACM0?baud=115200'
go run ./cmd/trimble-rawdata-dashboard -port 'tcp://192.168.1.10:5017'

# Verbose logging (stderr): off | info | debug | trace
go run ./cmd/trimble-rawdata-dashboard -port 'tcp://sps855.com:28005' -verbose debug

# Developer options in the web UI (track mode numbers); also enabled with -verbose debug or trace
go run ./cmd/trimble-rawdata-dashboard -dev
```

Open http://localhost:8080 (direct/local access).

The web UI includes a **Theme** control (System / Light / Dark). System follows the OS preference; the choice is remembered in the browser.

### Multiple browser users

Each browser tab gets its own session (cookie). Users can connect to **different** receivers at the same time. Multiple users on the **same** host:port share one receiver link and the same live data; the link is closed only after the **last** user disconnects (Disconnect button or closing the tab).

### Reverse proxy

Use `-base-path` when the dashboard is served under a URL prefix (local direct access keeps the default root `/`):

```bash
go run ./cmd/trimble-rawdata-dashboard -base-path /trimble-dashboard
```

Open http://localhost:8080/trimble-dashboard (or your proxy’s public URL with the same path).

Example **nginx** (SSE needs buffering off):

```nginx
location /trimble-dashboard/ {
    proxy_pass http://127.0.0.1:8080/trimble-dashboard/;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_buffering off;
}
```

Start the app with `-base-path /trimble-dashboard` so routes, static assets, API calls, and session cookies align with the proxy path.

### Verbose levels

| Level | What you get |
|-------|----------------|
| `off` | Errors and connection events only (default) |
| `info` | Link stats every 10s (bytes, frames, DCOL 0x57 / 0x40 counts, assembled RT27/position) |
| `debug` | Each DCOL packet by type (0x57, 0x40, …), GSOF records, RAWDATA pages, reassembly errors |
| `trace` | Raw RX bytes, frame hex, stream-decoder state messages |

With `-verbose debug` or `-verbose trace`, developer-only table options are shown (same as `-dev`).

The UI needs **DCOL 0x57 RAWDATA** record subtypes **6** (RT27 survey) and **7** (enhanced position). Other 0x57 subtypes (e.g. 12 = receiver info) are decoded but not shown. A stream may carry **0x40 GSOF**, **0x55 RETSVDATA**, and **0x57 RAWDATA** together — all are DCOL.

## Build

Cross-compile for Windows, macOS (ARM64), and Linux (32- and 64-bit):

```bash
chmod +x scripts/build.sh   # once
./scripts/build.sh
```

Binaries are written to `dist/`:

| File | Platform |
|------|----------|
| `trimble-rawdata-dashboard-windows-amd64.exe` | Windows 64-bit |
| `trimble-rawdata-dashboard-darwin-arm64` | macOS Apple Silicon |
| `trimble-rawdata-dashboard-linux-amd64` | Linux 64-bit |
| `trimble-rawdata-dashboard-linux-arm32` | Linux ARM32 (armv7) |

Override output directory: `OUT_DIR=/tmp/out ./scripts/build.sh`

### Build on every commit (local)

Install the pre-commit hook once after cloning:

```bash
chmod +x scripts/install-hooks.sh scripts/build.sh
./scripts/install-hooks.sh
```

Each commit then cross-compiles all four targets into `dist/` (gitignored). The commit is blocked if any build fails.

Skip for a single commit: `SKIP_BUILD=1 git commit …`

Local build for the current machine:

```bash
go build -o trimble-rawdata-dashboard ./cmd/trimble-rawdata-dashboard
```

## Local development

`go.mod` includes a `replace` for the DCOL library at `../../BitBucket/geoffrey-kirk-go-dcol`. Adjust the path or remove it once the library is published where `go get` can reach it.

## Architecture

```
Receiver (serial/tcp)
    → session.Client (geoffrey-kirk-go-dcol)
    → store (latest snapshot)
    → hub (SSE broadcast)
    → browser (sky plot, position, SNR table)
```

- **GET /** — static UI (embedded)
- **GET /api/config** — hosted mode and connection policy
- **POST /api/connect** — `{ "host", "port" }` or `{ "uri": "tcp://…" }` (hosted mode)
- **POST /api/disconnect** — stop ingest (hosted mode)
- **POST /api/demo** — synthetic data (hosted mode)
- **GET /api/snapshot** — current JSON state
- **GET /api/events** — Server-Sent Events stream

## Related

- **SNR compare app** — planned separate repo; diffs `RT27Survey.SNREntries()` across receivers or models.
