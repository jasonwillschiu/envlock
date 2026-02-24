# envlock Server Migration Plan (Single-Team, E2E, CLI-First)

## Goal

Migrate `envlock` from a Tigris-centric metadata workflow (and repo-coupled setup) to a server-backed sharing model that:

- removes repo clone/pull as a requirement for machine onboarding
- keeps encryption/decryption in the CLI (end-to-end with `age`)
- uses Google login for server access (team-only)
- uses a web UI for invite approval, device visibility, and status
- keeps secret upload/download in the CLI

This document is the implementation/migration plan for pre-MVP.

## Product Decisions (Locked In)

- Team scope: single team only (not multi-tenant)
- Auth: browser Google login
- CLI login UX: localhost callback, with copy/paste code fallback
- Invite approval: manual via web UI (default)
- Web UI: metadata only (no plaintext secret editing/viewing)
- Secret pull behavior: default write to local filename, optional `--out` override, `--force` for overwrite
- Multiple secret files supported from day one (`.env`, `.env.production`, `worker.env`, etc.)
- Device revoke flow: revoke now, then rekey later with visible `rekey required` status

## Security Model (MVP)

### Protects

- Secret plaintext from server/storage compromise (ciphertext only on server)
- Secret plaintext from unauthorized team members/devices that lack a valid recipient private key
- Team workflows without sharing private keys

### Does Not Protect

- Compromised client machine with access to plaintext and private key
- Plaintext once written locally after `pull`
- Secrets copied elsewhere outside `envlock`

### Why manual web approval is the right default

Manual approval is stronger than auto-approve because an admin can verify:

- authenticated user email
- requested device name
- recipient fingerprint
- request time / invite source

Auto-approve can be added later, but it should not be the default.

## Target Architecture

## Components

### 1) `envlock` CLI (data plane + local crypto)

Responsibilities:

- key generation and local key storage
- local encryption/decryption using `age`
- authenticated calls to envlock server APIs
- upload/download ciphertext
- rekeying on active devices

### 2) Envlock Server (control plane + ciphertext store)

Responsibilities:

- Google-authenticated user access
- project membership and team authorization
- devices/recipients registry
- invites and enrollment requests
- secret manifest/version metadata
- ciphertext blob storage
- audit events
- minimal admin web UI

The server must not decrypt secrets.

### 3) Web UI (admin and visibility)

Responsibilities:

- create invites
- review/approve/reject enrollment requests
- view devices and revoke
- view secrets metadata (name/version/size/rekey status)

The web UI should not handle plaintext secret bytes in MVP.

## Data Model (Server)

Minimal schema for MVP:

- `users`
  - Google identity
  - email, display name
- `projects`
  - team-owned project metadata
- `project_members`
  - user membership and role (`admin`, `member`)
- `devices`
  - user_id, project_id, device_name, public_key, fingerprint, status
- `invites`
  - project_id, token hash, expires_at, single-use state, created_by
- `enroll_requests`
  - project_id, invite_id, user_id, device details, status, decision metadata
- `secrets`
  - project_id, secret_name, latest_version, rekey_required flag
- `secret_versions`
  - secret_id, version, blob_path, sha256, size_bytes, recipient_set_hash, created_by_device_id
- `audit_events`
  - actor, action, project_id, entity refs, timestamp, metadata

## Storage Layout

MVP recommendation:

- DB: SQLite (fast setup, enough for single-team)
- Ciphertext blobs: local filesystem on server (`data/blobs/...`)

Future:

- swap blob storage backend to S3/Tigris without changing CLI/API contracts

## CLI Command Surface (New)

The CLI should move toward cleaner nouns while preserving old commands during migration.

### Auth / Setup

- `envlock init`
- `envlock login`
- `envlock whoami`
- `envlock project create <name>`
- `envlock project use <name>`

### Secrets

- `envlock secrets push <path>`
- `envlock secrets pull <name> [--out <path>] [--force]`
- `envlock secrets ls`
- `envlock secrets status`
- `envlock secrets rekey <name>`
- `envlock secrets rekey --all`

### Sharing / Devices

- `envlock invite create [--ttl 10m]`
- `envlock invite join <token-or-url>`
- `envlock devices ls`
- `envlock devices revoke <device>`

### Admin fallback (CLI)

- `envlock requests ls`
- `envlock requests approve <id>`
- `envlock requests reject <id>`

## Main Scenarios

## Scenario 1: Machine1 starts project and uploads secrets

### CLI steps (machine1)

1. Install:
   - `go install github.com/jasonchiu/envlock@latest`
2. Generate local key:
   - `envlock init`
3. Login:
   - `envlock login`
4. Create/select project:
   - `envlock project create my-app`
   - `envlock project use my-app`
5. Upload secrets (encrypt locally, upload ciphertext):
   - `envlock secrets push .env`
   - `envlock secrets push .env.production`
6. Create invite:
   - `envlock invite create --ttl 10m`
7. Share generated invite URL/token with teammate

### Web UI (optional machine1 admin actions)

- verify device registration
- review secrets metadata list
- create invite in UI instead of CLI (optional)

## Scenario 2: Machine2 joins and downloads `.env`s (no repo clone)

### CLI steps (machine2)

1. Install:
   - `go install github.com/jasonchiu/envlock@latest`
2. Generate local key:
   - `envlock init`
3. Login:
   - `envlock login`
4. Join invite:
   - `envlock invite join <token>`
   - or `envlock invite join "https://.../join?token=..."`
5. Wait for approval
6. List/pull secrets:
   - `envlock secrets ls`
   - `envlock secrets pull .env`
   - `envlock secrets pull .env.production`

### Web UI steps (admin)

1. Open `Pending Requests`
2. Review join request (email, device name, fingerprint)
3. Click `Approve`

Result:

- device added as active recipient
- invite consumed
- request status updated to approved

## Scenario 3: Machine2 revoked (and rekey)

### Web UI steps (admin)

1. Open `Devices`
2. Revoke machine2
3. Confirm warning that rekey is still needed for existing ciphertext

Server effects:

- device status -> revoked
- project secrets marked `rekey_required`
- audit event recorded

### CLI steps (active device/admin machine)

1. Check rekey status:
   - `envlock secrets status`
2. Rekey all:
   - `envlock secrets rekey --all`
3. Verify:
   - `envlock secrets status`

Expected outcome:

- new secret versions uploaded encrypted to active recipients only
- revoked device excluded from recipient set
- `rekey_required` cleared

## API Contracts (MVP)

### Auth

- Browser-based Google login
- CLI login via localhost callback
- Fallback CLI auth code exchange endpoint

Suggested routes:

- `GET /login`
- `GET /login/google`
- `GET /login/google/callback`
- `POST /api/cli/login/start`
- `POST /api/cli/login/exchange`

### Projects

- `GET /api/projects`
- `POST /api/projects`
- `GET /api/projects/{id}`
- `GET /api/projects/{id}/manifest`

### Invites / Enrollment

- `POST /api/projects/{id}/invites`
- `POST /api/invites/{token}/join`
- `GET /api/projects/{id}/requests`
- `POST /api/projects/{id}/requests/{id}/approve`
- `POST /api/projects/{id}/requests/{id}/reject`

### Devices

- `GET /api/projects/{id}/devices`
- `POST /api/projects/{id}/devices/{id}/revoke`

### Secrets (ciphertext + metadata)

- `GET /api/projects/{id}/secrets`
- `PUT /api/projects/{id}/secrets/{name}`
  - request: ciphertext bytes + metadata (`base_version`, `sha256`, `size`, recipient-set hash)
- `GET /api/projects/{id}/secrets/{name}`
  - returns latest ciphertext version by default

## Concurrency + Versioning Rules (Required)

To avoid clobbering/overwrites:

- each secret has monotonic `version`
- uploads must provide `base_version` (or equivalent ETag)
- server rejects stale writes with `409 Conflict`
- CLI surfaces retry guidance

This is required before multi-machine writes are considered reliable.

## Migration Strategy (Phased, Low-Risk)

## Phase 0: CLI surface + abstraction groundwork (this repo)

Purpose:

- prepare for server backend without breaking current Tigris behavior

Work:

- introduce backend interface abstraction over remote store operations
- keep Tigris implementation as the current backend
- add new command aliases (`invite`, `devices`, `requests`, `secrets`, `login`, `whoami`) with compatibility shims
- allow `invite join` to accept token or URL
- document future behavior in help output

Exit criteria:

- current commands still function
- new aliases route to equivalent current behavior where possible

## Phase 1: Server repository scaffold

Purpose:

- create server control plane and admin UI

Work:

- start from patterns in `/Users/jasonchiu/Documents/WVC/go-datastar1` (auth/router/session/UI)
- strip unrelated demo features
- add envlock domain models and routes
- add SQLite migrations
- add local blob storage adapter

Exit criteria:

- user can sign in with Google
- team allowlist enforced
- admin can create project/invite and approve requests in UI

## Phase 2: CLI server auth + invite/enrollment backend

Purpose:

- support server-backed onboarding without repo clone

Work:

- implement `envlock login`
- local token/session storage (0600)
- server API client
- `invite create/join`, `requests`, `devices` commands against server API

Exit criteria:

- machine2 can join via invite URL/token and get approved in web UI

## Phase 3: Secrets push/pull (server blobs + manifests)

Purpose:

- make server the shared ciphertext source of truth

Work:

- local `age` encrypt/decrypt commands
- `secrets push/pull/ls`
- manifest/versioning and conflict handling
- safe overwrite defaults for pull (`--force`)
- `--out` support for alternate path

Exit criteria:

- machine1 uploads ciphertext
- machine2 downloads and decrypts after approval

## Phase 4: Revoke + rekey workflow

Purpose:

- enforce device removal safely

Work:

- revoke device in UI + CLI
- mark secrets `rekey_required`
- implement `secrets rekey --all`
- clear flags when rekey completes

Exit criteria:

- revoked device cannot access latest secrets after rekey

## Phase 5: Cleanup / deprecations

Purpose:

- reduce confusion and legacy paths

Work:

- deprecate Tigris-only workflow in docs/help
- keep legacy commands as aliases for a transition period
- add migration guides from `.envlock/project.toml` + Tigris

Exit criteria:

- new server-first workflow is the documented default

## Dependency Checklist (Do Not Miss)

## Existing dependencies already in CLI repo

- `filippo.io/age` (crypto)
- `BurntSushi/toml` (config)
- AWS SDK (`tigris` adapter only; legacy path)

## Planned CLI additions (server backend)

- HTTP client stack (stdlib `net/http` is sufficient for MVP)
- browser opener helper (optional; can shell out carefully)
- local callback listener (`net/http`, `net`)
- secure local credential/session file storage (stdlib is sufficient)

## Planned server dependencies (new server repo or module)

- HTTP router (Chi is already proven in `go-datastar1`)
- Google OAuth/OIDC libraries (already proven in template)
- SQLite driver + migrations
- templ/templating stack (if using template UI approach)
- session/token generation + hashing (template patterns exist)
- optional structured logging and tracing (can defer heavy telemetry)

## Non-goals for MVP

- multi-tenant SaaS
- browser plaintext editing
- hardware-backed key storage
- device auto-approval by default
- advanced policy engine
- P2P relay / no-storage architecture

## Notes for Command Behavior

### `envlock secrets pull`

Default:

- writes to local file named `<secret-name>` in current directory

Options:

- `--out <path>` writes to explicit path
- `--force` required to overwrite existing files

This matches the agreed UX: simple by default, explicit when needed.
