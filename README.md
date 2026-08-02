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
- **Tutors & Students are logins.** Full CRUD directory (name, email, phone,
  profile photo, and a shared normalized **address**), and creating one creates
  the account it signs in with — atomically, in one transaction. `tutors` and
  `students` share their primary key with `dashboard_users`, so a member's id,
  their JWT subject, and every foreign key pointing at them are the same
  integer. A random temporary password is generated, hashed like any other, and
  returned exactly once in the create response; the invite email carries it too.
- **Batches** — a scheduled *offering* of a course:
  - Assign one **tutor per module**, each with a start / expected-end date.
  - Enroll students either by **CSV import** (upserted at the org level, keyed
    on email) or by **picking existing students** from a searchable, paginated
    multiselect drawer.
  - A per-batch **resource drive**: a real folder/file tree (create folders,
    upload files, rename, delete — cascading through subfolders), scoped to
    that batch.
  - **Publishing** a batch notifies every assigned tutor and enrolled student.
    The request returns immediately: the emails are queued as events and sent by
    a separate worker, so publishing to a 200-student roster is one database
    round trip rather than 200 sequential SMTP dials.
- **Lectures & live rooms.** Instructors can create lecture rooms, start/end
  them, generate LiveKit join tokens, and review recordings from the dashboard.
- **File storage.** Course thumbnails/resources and batch drive files are stored
  in **MinIO** (S3-compatible object storage).
- **Frontend authorization mirrors the backend.** The dashboard fetches the
  current user's privileges after login and uses them to hide controls, guard
  whole routes, and render **Not Authorized (403)** / **Not Found (404)** pages
  for direct navigation — matching what the API actually enforces.
- **Event-driven transactional email.** Templates live in the database
  (verification, password reset, member invite, batch tutor/student
  notifications) and are rendered with `{{placeholders}}` — HTML-escaped, in a
  single pass, with a warning logged for any placeholder the caller didn't
  supply. Nothing is sent from the request path: see below.

## Notifications: transactional outbox → Redis Streams → worker

Email is never sent from an HTTP handler. Instead:

```
API process                                    Worker process
┌────────────────────────────┐                 ┌──────────────────────────┐
│ repo tx {                  │                 │ claim → render → SMTP    │
│   business writes          │   XADD          │        → ack             │
│   outbox.Insert(tx, event) │  ──────────►    │                          │
│ } commit                   │  Redis Streams  │ templates + mailer live  │
│        ↓                   │  · notifications│ HERE, not in the API     │
│   outbox_events            │  · auth (own    │        ↓                 │
│        ↓                   │    stream, so   │  processed_events (dedupe)│
│   relay: poll → XADD →     │    bulk mail    │  dead_events (DLQ)       │
│          hard-delete       │    can't delay  │                          │
└────────────────────────────┘    a login OTP) └──────────────────────────┘
```

Why it's built this way:

- **The event is written in the same transaction as the business change.** Either
  a tutor row and the promise to email them both commit, or neither does.
  Publishing to Redis from a handler can't offer that — the process can die
  between `COMMIT` and `PUBLISH`, losing the notification with no record it was
  ever owed.
- **Delivery is at-least-once and unordered**, deliberately. A duplicate email is
  acceptable; a lost one is not. Handlers are idempotent via a claim row that
  doubles as a lock, which is what stops two workers both sending when one stalls
  mid-SMTP past the reclaim window.
- **Failures are triageable.** After N attempts (or immediately, for permanent
  errors like a missing template) an event lands in `dead_events` — a Postgres
  table you can query with `psql`, not a Redis stream you have to spelunk.
  `make dlq` lists them.
- **Retryability is decided by the mailer**, not the worker: the obvious
  4xx/5xx split is wrong often enough to matter (`421` closes the connection,
  `450` is greylisting).
- **The bus is also the seam for a future service split** — admin, auth, and
  notification-worker. Event payloads are therefore self-contained: they carry the
  recipient and every template variable, never an id the worker would have to look
  up in a database it won't have access to.

Two consequences worth knowing before you deploy:

- **The API and worker are a coupled deploy.** With no worker running, mail
  queues up in `outbox_events` and never sends. `make dlq` and the relay's
  lag log are how you'd notice.
- **`POST /auth/send-verification` returns 200 where it used to return 500** on an
  SMTP outage, because delivery now happens after the response. Any alert
  watching 5xx on that route will go quiet; that silence isn't SMTP getting
  healthier.

## Tech stack

| | |
|---|---|
| **Backend** | Go, Gin, PostgreSQL (pgx), Redis (cache + Streams), golang-migrate, MinIO (`minio-go`), LiveKit, JWT |
| **Frontend** | Next.js 15 (App Router), React 19, TypeScript, Redux Toolkit + RTK Query, Tailwind CSS + shadcn/ui, `@uiw/react-md-editor` + `react-markdown`, LiveKit React components |
| **Infra (dev)** | Docker Compose — Postgres, Redis, Mailpit (SMTP + web UI), MinIO, LiveKit |

## Architecture at a glance

**Backend** (`server/`) is a modular monolith with exactly three modules —
**admin** (batches, courses, lecture, students, tutors, webhooks), **auth**, and
**notification** — each split layer-first:

```
modules/admin/
  dto/<domain>/         model/<domain>/       repository/<domain>/
  service/<domain>/     handler/<domain>/     module/<domain>/
```

Every layer is its own Go package, so the dependency direction is enforced by the
compiler rather than by convention. The graph is acyclic:

```
module → handler → service → repository → model
              ↘        ↘         ↘
                  dto  ────────────
```

Domain errors live in `model/` so a handler can map them to status codes without
importing the repository. Because `dto/batches` and `model/batches` are both
`package batches`, importers alias them (`dto "…/dto/batches"`).

`internal/pkg/` holds only what crosses module boundaries: `events`, `outbox`,
`pg`, `httpx`, `jwtutil`, `security`, `database`, `redisclient`. Anything used by
a single module lives inside it — `address`, `livekit`, `scope` and `storage`
under `admin/`; `mailer` and `worker` under `notification/`.

There are two binaries: `cmd/api` (HTTP + the outbox relay) and `cmd/worker`
(the only process that talks to SMTP).

Every table that holds tenant data carries a `customer_id`, and every query is
scoped to the caller's tenant — there is no cross-tenant data access path.
Access control is two-layered: privileges are checked server-side on every
request (the source of truth), and mirrored client-side purely for UX (hiding
buttons/routes the user couldn't use anyway).

**Frontend** (`client/`) is a Next.js App Router dashboard: RTK Query slices
(`authApi`, `coursesApi`, `tutorsApi`, `studentsApi`, `batchesApi`,
`lecturesApi`) talk to the API with automatic access-token refresh, and a
`DashboardProvider` fetches the current user + privileges once and shares them
via context.

Layout is centralized rather than reassembled per page: `AppShell` owns the
navbar + collapsible sidebar (with a `Sheet` drawer on mobile), and `PageHeader`
owns titles, breadcrumbs and action slots. `DataTable` renders a table above
`sm` and stacked cards below it, with skeleton rows while loading and an
`EmptyState` when there's nothing — so a list page gets responsive, accessible
behaviour without opting in. Design tokens (a two-hue brand palette, radius,
elevation) live in `globals.css` and drive both light and dark mode. Multi-field
forms use a Sheet; quick actions use a Dialog; destructive actions go through
`useConfirm()` rather than `window.confirm`. Lecture pages lazy-load the LiveKit
bundle, so it costs nothing until someone actually joins a room.

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

**5. Start the notification worker** (second terminal). Without it, emails queue
in `outbox_events` and are never delivered:
```sh
make run-worker   # readiness probe on http://localhost:8081/ready
```

**6. Start the web client** (third terminal):
```sh
cd client
cp .env.local.example .env.local   # set NEXT_PUBLIC_API_URL if needed
npm install
npm run dev        # http://localhost:3000
```

Open the client, register an organization (this creates your tenant and an
Admin account after email verification — check Mailpit at
`http://localhost:8025` for the OTP), and sign in.

### Tests

```sh
make test                                        # unit tests, no infrastructure
TEST_DATABASE_URL=postgres://tutorpilot:tutorpilot@localhost:5432/tutorpilot?sslmode=disable \
  go test ./...                                  # + integration tests
```

The integration tests **skip silently** without `TEST_DATABASE_URL`, so a plain
`make test` looks green while only exercising pure logic. The skipped ones cover
outbox atomicity and the concurrent-claim race — the parts most worth running.

### Inspecting the queue

```sh
make dlq        # unreplayed dead events
```

`dead_events` is currently read-only: there is no replay tool. One-time-code
payloads are stored redacted (the code expires within minutes), while invite
payloads are stored whole, because the temporary password exists nowhere else in
the system.

## Repository layout

```
server/
  cmd/
    api/                 HTTP server + the outbox relay
    worker/              notification consumer -- the only process that sends email
    migrate/             migration runner
  internal/
    config/              env-driven configuration (validated at startup)
    middleware/          auth + RBAC gating for Gin routes
    modules/
      admin/             batches, courses, lecture, students, tutors, webhooks
        dto/<domain>/         request/response shapes
        model/<domain>/       domain types + domain errors
        repository/<domain>/  Postgres access
        service/<domain>/     business logic
        handler/<domain>/     HTTP
        module/<domain>/      wiring + route registration
        address/ livekit/ scope/ storage/    admin-only infrastructure
      auth/              same six layers: tenants, users, roles, JWT, OTP
      notification/      templates + the invite contract (imported by producers)
        mailer/ repository/ service/ worker/
    pkg/                 shared across modules only
      database/ events/ httpx/ jwtutil/ outbox/ pg/ redisclient/ security/
  migrations/            golang-migrate SQL migrations (schema + seed data)
  docker-compose.yml     local dev infrastructure

client/
  app/
    (auth)/              split-screen brand panel + login/register/reset
    dashboard/           courses, tutors, students, batches, lectures + loading states
  components/
    ui/                  shadcn/ui primitives (dialog, sheet, avatar, skeleton, ...)
    layout/              AppShell, PageHeader, StatCard, EmptyState, PageLoader
    brand/               the logo mark
    providers/           ConfirmProvider — replaces window.confirm app-wide
    dashboard/           feature components (courses, people forms, batches, lectures)
  lib/
    api/                 RTK Query slices (authApi, coursesApi, tutorsApi, ...)
    features/            Redux slices (auth state incl. privileges)
    hooks/               useCan() privilege-check hook, typed Redux hooks
```
