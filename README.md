# TutorPilot

TutorPilot is a multi-tenant SaaS platform for running an online tutoring/coaching
business: each signed-up organization (a **tenant**) gets its own isolated space to
manage staff accounts, roles and permissions, and build courses — complete with a
curriculum editor, Markdown lesson content, and file/image storage.

The repo is a monorepo with two halves:

```
TutorPlatform/
├── server/   Go API — auth, multi-tenancy, RBAC, courses
└── client/   Next.js dashboard — the web UI
```

## What it does today

- **Multi-tenant accounts.** Signing up creates an *organization* (tenant), not
  just a user — the signer becomes that organization's Admin. Every tenant's data
  (users, roles, courses, files) is isolated from every other tenant's.
- **Auth.** Email OTP verification → registration, login, refresh-token rotation,
  forgot/reset password, logout — all backed by per-tenant JWT signing secrets.
- **Role-based access control (RBAC).** Each tenant has `Super Admin` / `Admin` /
  `User` roles. Permissions are granular *privileges* (e.g. `course.create`,
  `course.edit`) mapped to roles per tenant, not hardcoded to role names — so
  what a role can do is entirely data-driven and adjustable per tenant.
- **Courses.** Admins create courses with a title, summary, and Markdown
  description; build a curriculum of **Modules → Lessons** (each lesson is
  Markdown content); attach file/image **resources**; and **publish/unpublish**
  a course when it's ready. Course files and images are stored in **MinIO**
  (S3-compatible object storage).
- **Frontend authorization mirrors the backend.** The dashboard fetches the
  current user's privileges after login and uses them to hide controls, guard
  whole routes, and render **Not Authorized (403)** / **Not Found (404)** pages
  for direct navigation — matching what the API actually enforces.
- **Transactional email**, driven by admin-editable templates (welcome, password
  reset, email verification) rendered with `{{placeholders}}`, sent through SMTP
  (Mailpit in dev).

## Tech stack

| | |
|---|---|
| **Backend** | Go, Gin, PostgreSQL (pgx), Redis, golang-migrate, MinIO (`minio-go`), JWT |
| **Frontend** | Next.js 15 (App Router), React 19, TypeScript, Redux Toolkit + RTK Query, Tailwind CSS + shadcn/ui, `@uiw/react-md-editor` + `react-markdown` |
| **Infra (dev)** | Docker Compose — Postgres, Redis, Mailpit (SMTP + web UI), MinIO |

## Architecture at a glance

**Backend** (`server/`) is organized into small, self-contained modules under
`internal/modules/` (currently `auth` and `courses`), each following the same
shape: `handler.go` (HTTP) → `service.go` (business logic) → `repository.go`
(Postgres access via pgx), plus `model.go`/`dto.go` for data shapes. Shared
infrastructure lives under `internal/pkg/` (JWT, password hashing, mailer,
MinIO storage client, HTTP response envelope + pagination helper) and
`internal/middleware/` (auth + privilege-gating for Gin routes).

Every table that holds tenant data carries a `customer_id`, and every query is
scoped to the caller's tenant — there is no cross-tenant data access path.
Access control is two-layered: privileges are checked server-side on every
request (the source of truth), and mirrored client-side purely for UX (hiding
buttons/routes the user couldn't use anyway).

**Frontend** (`client/`) is a standard Next.js App Router dashboard: RTK Query
slices (`authApi`, `coursesApi`) talk to the API with automatic access-token
refresh; a `DashboardProvider` fetches the current user + privileges once and
shares them via context; generic building blocks (`DataTable`, `ContentCard`,
`PageTheme`) keep pages visually consistent.

See [`server/README.md`](server/README.md) *(if present)* and
[`client/README.md`](client/README.md) for stack details specific to each half.

## Running it locally

**1. Start infrastructure** (Postgres, Redis, Mailpit, MinIO):
```sh
cd server
docker compose up -d
```

**2. Configure environment.** Copy `server/.env.example` to `server/.env` and
fill in `DATABASE_URL`, `PASSWORD_PEPPER`, etc. (sane defaults are provided for
local Postgres/Redis/MinIO/Mailpit).

**3. Run database migrations:**
```sh
make migrate-up
```

**4. Start the API:**
```sh
make run          # http://localhost:8080, Swagger at /swagger/index.html (non-prod)
```

**5. Start the web client** (in a second terminal):
```sh
cd client
cp .env.local.example .env.local   # set NEXT_PUBLIC_API_URL if needed
npm install
npm run dev        # http://localhost:3000
```

Open the client, register an organization (this creates your tenant and an
Admin account after email verification — check Mailpit at
`http://localhost:8025` for the OTP), and sign in.

## Repository layout

```
server/
  cmd/                  entrypoints: api server, migration runner
  internal/
    config/             env-driven configuration
    middleware/          auth + RBAC gating for Gin routes
    modules/
      auth/              tenants, users, roles, privileges, JWT, OTP flows
      courses/           courses, modules, lessons, resources
      notification/      email templates + sending
    pkg/                 shared: jwtutil, security, mailer, storage (MinIO), httpx, pg
  migrations/            golang-migrate SQL migrations (schema + seed data)
  docker-compose.yml     local dev infrastructure

client/
  app/                   Next.js App Router routes (auth pages, dashboard, courses)
  components/             shadcn/ui primitives + feature components (courses, dashboard shell)
  lib/
    api/                  RTK Query slices (authApi, coursesApi)
    features/             Redux slices (auth state incl. privileges)
    hooks/                 useCan() privilege-check hook, typed Redux hooks
```
