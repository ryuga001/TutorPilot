# TutorPilot

TutorPilot is a multi-tenant SaaS platform for running an online tutoring/coaching
business: each signed-up organization (a **tenant**) gets its own isolated space to
manage staff accounts, roles and permissions, build courses, and run them as
scheduled **batches** — with tutor assignment, student enrollment, and a shared
file drive per batch.

The repo is a monorepo with two halves:

```
TutorPlatform/
├── server/   Go API — auth, multi-tenancy, RBAC, courses, batches, people
└── client/   Next.js dashboard — the web UI
```

## What it does today

- **Multi-tenant accounts.** Signing up creates an *organization* (tenant), not
  just a user — the signer becomes that organization's Admin. Every tenant's data
  (users, roles, courses, batches, tutors, students, files) is isolated from every
  other tenant's.
- **Auth.** Email OTP verification → registration, login, refresh-token rotation,
  forgot/reset password, logout — all backed by per-tenant JWT signing secrets.
- **Role-based access control (RBAC).** Each tenant has `Super Admin` / `Admin` /
  `User` roles. Permissions are granular *privileges* (e.g. `course.create`,
  `batch.edit`, `tutor.delete`) mapped to roles per tenant, not hardcoded to role
  names — so what a role can do is entirely data-driven and adjustable per tenant.
  New privileges added to the catalog automatically flow to every tenant's Admin.
- **Courses.** Admins create courses with a title, summary, and Markdown
  description; build a curriculum of **Modules → Lessons** (each lesson is
  Markdown content); attach file/image **resources** (shown as a folder/file-style
  icon grid); and **publish/unpublish** a course when it's ready.
- **Tutors & Students.** Full CRUD directory of tutors and students (name, email,
  phone, profile photo, and a shared normalized **address**). These are
  admin-managed records — no login of their own (yet).
- **Batches** — a scheduled *offering* of a course:
  - Assign one **tutor per module**, each with a start / expected-end date.
  - Enroll students either by **CSV import** (upserted at the org level, keyed
    on email) or by **picking existing students** from a searchable, paginated
    multiselect drawer.
  - A per-batch **resource drive**: a real folder/file tree (create folders,
    upload files, rename, delete — cascading through subfolders), scoped to
    that batch.
  - **Publishing** a batch emails every assigned tutor and every enrolled
    student their details, via the same templated-email system used for OTPs.
- **Lectures & live rooms.** Instructors can create lecture rooms, start/end
  them, generate LiveKit join tokens, and review recordings from the dashboard.
- **File storage.** Course thumbnails/resources and batch drive files are stored
  in **MinIO** (S3-compatible object storage).
- **Frontend authorization mirrors the backend.** The dashboard fetches the
  current user's privileges after login and uses them to hide controls, guard
  whole routes, and render **Not Authorized (403)** / **Not Found (404)** pages
  for direct navigation — matching what the API actually enforces.
- **Transactional email**, driven by admin-editable templates (welcome, password
  reset, email verification, batch tutor/student notifications) rendered with
  `{{placeholders}}`, sent through SMTP (Mailpit in dev).

## Tech stack

| | |
|---|---|
| **Backend** | Go, Gin, PostgreSQL (pgx), Redis, golang-migrate, MinIO (`minio-go`), LiveKit, JWT |
| **Frontend** | Next.js 15 (App Router), React 19, TypeScript, Redux Toolkit + RTK Query, Tailwind CSS + shadcn/ui, `@uiw/react-md-editor` + `react-markdown`, LiveKit React components |
| **Infra (dev)** | Docker Compose — Postgres, Redis, Mailpit (SMTP + web UI), MinIO, LiveKit |

## Architecture at a glance

**Backend** (`server/`) is organized into small, self-contained modules under
`internal/modules/` (`auth`, `courses`, `tutors`, `students`, `batches`,
`lecture`, `notification`), each following the same shape: `handler.go` (HTTP) →
`service.go` (business logic) → `repository.go` (Postgres access via pgx), plus
`model.go`/`dto.go` for data shapes. Shared infrastructure lives under
`internal/pkg/` (JWT, password hashing, mailer, MinIO storage client, a shared
`address` store used by both tutors and students, HTTP response envelope +
pagination helper) and `internal/middleware/` (auth + privilege-gating for Gin
routes). Live lecture rooms and join-token generation are handled by
`internal/livekit/`.

Every table that holds tenant data carries a `customer_id`, and every query is
scoped to the caller's tenant — there is no cross-tenant data access path.
Access control is two-layered: privileges are checked server-side on every
request (the source of truth), and mirrored client-side purely for UX (hiding
buttons/routes the user couldn't use anyway).

**Frontend** (`client/`) is a standard Next.js App Router dashboard: RTK Query
slices (`authApi`, `coursesApi`, `tutorsApi`, `studentsApi`, `batchesApi`,
`lecturesApi`) talk to the API with automatic access-token refresh; a
`DashboardProvider` fetches the current user + privileges once and shares them
via context; generic building blocks (`DataTable`, `ContentCard`, `PageTheme`,
`DetailPageHeader`) keep pages visually consistent across entities. Multi-field
forms use a Sheet (drawer); quick single-purpose actions use a Dialog. Lecture
pages render a LiveKit room experience for active sessions and recording players
for completed ones.

See [`client/README.md`](client/README.md) for frontend stack details.

## Running it locally

**1. Start infrastructure** (Postgres, Redis, Mailpit, MinIO):
```sh
cd server
docker compose up -d
```

**2. Configure environment.** Copy `server/.env.example` to `server/.env` and
fill in `DATABASE_URL`, `PASSWORD_PEPPER`, etc. (sane defaults are provided for
local Postgres/Redis/MinIO/Mailpit). The LiveKit values in the example are
already wired for the local Docker stack; if you want the client to connect to a
custom LiveKit host, set `NEXT_PUBLIC_LIVEKIT_URL` in `client/.env.local` (for
example `ws://127.0.0.1:7880`).

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
      tutors/            tutor directory CRUD + profile image
      students/          student directory CRUD + profile image
      batches/           batches, module↔tutor assignment, enrollment, resource drive
      lecture/           live lecture rooms, join tokens, recording workflow
      notification/      email templates + sending
    pkg/                 shared: jwtutil, security, mailer, storage (MinIO), address, httpx, pg
  migrations/            golang-migrate SQL migrations (schema + seed data)
  docker-compose.yml     local dev infrastructure

client/
  app/dashboard/         courses, tutors, students, batches, lectures routes + shell
  components/
    ui/                  shadcn/ui primitives (dialog, sheet, select, table, ...)
    dashboard/           feature components (courses, tutors/students forms, batches)
  lib/
    api/                 RTK Query slices (authApi, coursesApi, tutorsApi, studentsApi, batchesApi)
    features/            Redux slices (auth state incl. privileges)
    hooks/               useCan() privilege-check hook, typed Redux hooks
```
