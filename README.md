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
```

Open http://localhost:8080

### Verbose levels

| Level | What you get |
|-------|----------------|
| `off` | Errors and connection events only (default) |
| `info` | Link stats every 10s (bytes, frames, DCOL 0x57 / 0x40 counts, assembled RT27/position) |
| `debug` | Each DCOL packet by type (0x57, 0x40, …), GSOF records, RAWDATA pages, reassembly errors |
| `trace` | Raw RX bytes, frame hex, stream-decoder state messages |

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
| `trimble-rawdata-dashboard-linux-386` | Linux 32-bit |

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
