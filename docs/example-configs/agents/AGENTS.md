# Cryptic Stash — Agent Instructions

## Project Overview

Cryptic Stash is an "inverted 2FA" web app for securely storing 2FA recovery codes. Users upload encrypted stashes that can be downloaded only after a waiting period following authentication. If an attacker attempts a download, the real user is notified via multiple messengers and can block it.

- **Backend**: Go + SQLite + Gin (see [`backend/AGENTS.md`](backend/AGENTS.md))
- **Frontend**: Svelte 5 + SvelteKit SPA + Tailwind v4 (see [`frontend/AGENTS.md`](frontend/AGENTS.md))
- **Deployment**: Single Docker binary (no CGo, frontend embedded via `embed.FS`)
- **Status**: Pre-release — breaking schema changes still expected

## Monorepo Structure

```
backend/    — Go module (github.com/NicoClack/cryptic-stash/backend)
frontend/   — SvelteKit SPA (static adapter, embedded by backend at build time or hosted by Vite dev server, communicating with backend via CORS)
Dockerfile  — Multi-stage: builds frontend, then Go binary with embedded static files
```

## Cross-Cutting Rules

- **Don't set up the project** — assume the environment is already configured. Only install packages or run setup when explicitly asked.
- **Run Go commands from `backend/`**, run Node commands from `frontend/`.
- **Minimise CLI commands** — prefer tools (read_file, search, runTests, etc.) over `cat <path> | tail 20`, `rg`, `go test <path> -v 2>&1 | grep -E "^(--- PASS|--- FAIL|FAIL|ok)" | head -20"` etc. Each CLI command requires human approval (~30s). Only use CLI for bulk actions tools can't handle (e.g., pipes).
- **Comments explain WHY, not WHAT** — if you need to explain what code does, extract it into a well-named function instead.
- **Check for errors before completing** — use the language server to validate changes. Skip only if the change is intentionally incomplete.
- **Read existing code before writing** — the codebase has strong conventions (especially error handling). Follow them.
- **Check repo memory** (`/memories/repo/`) for known pitfalls before debugging related areas.
