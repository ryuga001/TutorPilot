# TutorPilot — Web Client

Next.js (App Router) dashboard for the TutorPilot API.

## Stack

- **Next.js 15** + React 19, TypeScript
- **Tailwind CSS** + **shadcn/ui** — square components (`--radius: 0`), blue primary
- **next-themes** — light / dark / system toggle
- **Redux Toolkit + RTK Query** — API layer with automatic refresh-token re-auth

## Setup

```bash
cp .env.local.example .env.local   # point NEXT_PUBLIC_API_URL at the Go API
npm install
npm run dev                        # http://localhost:3000
```

The API base URL defaults to `http://localhost:8080/api/v1`.

## Auth flow → API endpoints

| Screen | Endpoint(s) |
| --- | --- |
| `/login` | `POST /auth/login` |
| `/register` (3-step wizard) | `POST /auth/send-verification` → `POST /auth/verify-email` → `POST /auth/register` |
| `/forgot-password` | `POST /auth/forgot-password` → `POST /auth/reset-password` |
| `/dashboard` (guarded) | `GET /auth/me`, `POST /auth/logout` |
| _transparent_ | `POST /auth/refresh` (on 401, via RTK Query `baseQuery`) |

## Layout

```
app/                  routes (auth group + dashboard)
components/ui/        shadcn primitives
components/           providers, theme toggle, auth guard
lib/store.ts          Redux store (+ localStorage persistence)
lib/api/authApi.ts    RTK Query endpoints
lib/api/baseQuery.ts  fetchBaseQuery + refresh-token re-auth
lib/features/         auth slice
```

## Theming

`next-themes` drives the `class` strategy; palette lives in CSS variables in
`app/globals.css`. Swap the `--primary` / `--radius` values there to re-skin —
e.g. set `--radius: 0.5rem` for rounded components.
