# Frontend Agent Instructions

## Architecture (High-Level)

- **Svelte 5 + SvelteKit** (v2), **static adapter** with SPA fallback — NO SSR
- **Tailwind CSS v4** + **shadcn-svelte** (`bits-ui`) for UI components
- **Dark theme only** — `class="dark"` forced in root layout
- **`sessionStorage`** for auth tokens (no cookies/JWT)

## Auth

Two separate systems:

- **User auth**: `UserAuth.svelte.ts` — WebAuthn/passkey login, `Bearer <token>` header
- **Admin auth**: `AdminAuth.svelte.ts` — password + TOTP, `AdminCode <code>` header

Both follow the same pattern: hydrate from `sessionStorage` on load → `requireAuth()` guards → redirect with `?redirectTo=`

## API Layer

`src/lib/api.ts` — `fetchJson()`, `fetchUserJson()`, `fetchAdminJson()` all prepend `PUBLIC_API_DOMAIN`. The `JsonResponse` wrapper handles error codes, auto-redirects on 401/403/404.

## Key Patterns

- SvelteKit file-based routing under `src/routes/`
- shadcn-svelte UI components imported from `$lib/components/ui/`
- JSON Schema forms via `@sjsf/form` (`$lib/form/JsonForm.svelte`)
- Setup-first flow: `+page.ts` load function probes `/api/v1/setup/`, redirects if incomplete

## Note

The frontend is incomplete — many backend features lack UI. Significant rework is planned. Focus backend-side changes on API surface compatibility.
