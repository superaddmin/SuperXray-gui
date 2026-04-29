# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**SuperXray** (SuperXray-gui) — web-based control panel for managing Xray-core proxy/VPN servers. Go backend with Gin framework, embedded frontend (Vue.js + Ant Design Vue), SQLite database, and Telegram bot integration.

- **Module**: `github.com/superaddmin/SuperXray-gui/v2`
- **Go version**: 1.26.2
- **License**: GPL V3

## Build & Development Commands

```bash
# Build (output: bin/SuperXray.exe)
go build -o bin/SuperXray.exe ./main.go

# Run with debug logging
XUI_DEBUG=true go run ./main.go

# Run all tests
go test ./...

# Run a single test
go test ./web/service/ -run TestXraySetting

# Vet
go vet ./...
```

CLI flags for admin operations (run the binary directly):
- `-reset` — reset all panel settings to defaults
- `-show` — display current settings (port, paths)

## Architecture

### Dual Server Design
Two HTTP servers run concurrently in one process:
- **Web Server** (`web/`) — main management panel
- **Sub Server** (`sub/`) — subscription link service (separate port)

### Layered Request Flow
```
Controller (web/controller/) → Service (web/service/) → Database (database/model/)
```
- Controllers use `*gin.Context`, translations via `I18nWeb(c, "key")`
- Services contain business logic, inject `xray.XrayAPI` dependency
- Models use GORM auto-migration, seeders tracked via `HistoryOfSeeders`

### Resource Embedding
All frontend assets are embedded at compile time via `//go:embed`:
- `web/assets` → `assetsFS` (CSS, JS, images)
- `web/html` → `htmlFS` (HTML templates)
- `web/translation` → `i18nFS` (TOML translation files for 13 languages)

**Changes to HTML/CSS/JS require recompilation** — no hot reload.

### Xray-core Integration
- Panel generates `config.json` dynamically from inbound/outbound settings
- Communicates with Xray binary via gRPC for real-time traffic stats
- Binary management: platform-specific `xray-{os}-{arch}` in bin folder
- Process lifecycle managed in `xray/process.go`, API in `xray/api.go`

### Key Directories
| Path | Purpose |
|------|---------|
| `config/` | Configuration, version/name constants |
| `database/` | GORM init, models (`database/model/model.go`) |
| `web/controller/` | Gin HTTP handlers |
| `web/service/` | Business logic (InboundService, SettingService, TgBot, etc.) |
| `web/job/` | Cron background jobs (traffic, CPU, IP tracking, LDAP sync) |
| `web/websocket/` | WebSocket hub for real-time client updates |
| `web/html/` | HTML templates |
| `web/translation/` | i18n TOML files |
| `xray/` | Xray-core process management and gRPC API |
| `sub/` | Subscription server |
| `util/` | Utilities (crypto, LDAP, sys helpers) |

### Background Jobs (robfig/cron)
Registered in `web/web.go` during server init:
- Traffic monitoring (`xray_traffic_job.go`)
- CPU alerts (`check_cpu_usage.go`)
- IP tracking (`check_client_ip_job.go`)
- LDAP sync (`ldap_sync_job.go`)

### Telegram Bot
- ~3700 lines in `web/service/tgbot.go`, uses `telego` with long polling
- **Critical**: Always call `service.StopBot()` before server restart to prevent 409 bot conflicts

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `XUI_DEBUG` | Enable debug logging |
| `XUI_LOG_LEVEL` | Override log level |
| `XUI_MAIN_FOLDER` | Override main folder path |

## Critical Patterns & Gotchas

- **Bot restart**: Must stop Telegram bot before any server restart (409 conflict)
- **Embedded assets**: Frontend changes need full recompilation
- **IP limitation**: "Last IP wins" — when LimitIP exceeded, oldest connections disconnected via Xray API
- **Session management**: Uses `gin-contrib/sessions` with cookie store
- **Password migration**: Seeder system tracks bcrypt migration in `HistoryOfSeeders` table

## Testing

Tests are table-driven Go tests in:
- `database/model/model_test.go`
- `web/job/check_client_ip_job_test.go`
- `web/job/check_client_ip_job_integration_test.go`
- `web/service/xray_setting_test.go`
- `web/service/custom_geo_test.go`

## Internationalization

Translation files: `web/translation/translate.*.toml`
- Access in controllers: `I18nWeb(c, "pages.login.loginAgain")`
- Access in bot: via `i18nFS` passed to bot startup
