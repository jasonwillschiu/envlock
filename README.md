# envlock

`envlock` is a Go CLI for encrypting `.env` files, storing the ciphertext in Tigris (S3-compatible object storage), and sharing decryption access across your machines using per-device keys.

The project is intentionally focused on a narrow workflow:

- keep `.env` bytes exactly as-is (comments/order/spacing/newlines preserved)
- encrypt to one or more device recipients
- store only encrypted blobs remotely
- add/remove machines safely by rekeying
- make onboarding better than manual file copying via Tigris enrollment invites (planned v1.1)

## Status

This repository is in early scaffold stage.

Implemented now:

- `envlock init`
- `envlock status`
- `envlock project init`
- `envlock project show`
- `envlock recipients list/add/remove`

Planned next:

- local `encrypt` / `decrypt`
- Tigris `push` / `pull`
- `rekey`
- Tigris enrollment invites (`enroll invite/join/list/approve/reject`)

## Why envlock (vs just AirDroping `.env`)

AirDrop is excellent for one-off nearby transfers on Macs. `envlock` is useful when you want a repeatable, safer workflow over time:

- Tigris as a shared encrypted source of truth
- no plaintext `.env` in cloud storage
- per-device access control (each machine has its own private key)
- device removal / rekeying workflow
- future cross-platform support (not Apple-only)

The key product challenge is onboarding. `envlock` addresses this by planning a Tigris-based invite enrollment flow so you do not manually copy/paste public keys in the normal path.

## Security Model (What It Does / Does Not Protect)

### Protects

- `.env` confidentiality at rest in Tigris
- remote blob exposure if someone gets Tigris read access but not a device private key
- multi-machine sharing without sharing private keys

### Does not protect

- plaintext `.env` on a machine after decryption
- a compromised machine that already has your private key and local access
- leaked secrets that were copied elsewhere
- operational mistakes (for example, checking decrypted `.env` into git)

## Core Concepts

### Device keypair

Each machine generates its own keypair locally:

- private key stays on the machine
- public key can be shared (or later handled through enrollment)

### Recipient encryption

`envlock` will use `age` multi-recipient encryption.

When encrypting a file for multiple devices, the tool encrypts the payload once and wraps the file key for each recipient public key. The result is one ciphertext that any authorized machine can decrypt with its own private key.

### Rekey

If you add or remove a machine later, old ciphertext must be re-encrypted to update the recipient list.

This is expected and secure behavior.

### Tigris as transport + source of truth

Tigris stores encrypted blobs and (planned) enrollment metadata.

Important: Tigris access is not the same as decryption authorization. The private key still gates access to secrets.

## Planned Architecture

### v1

- per-device keypairs (local file storage)
- project config in `./.envlock/project.toml`
- recipients in `./.envlock/recipients.json` (safe to commit)
- local encrypt/decrypt
- Tigris push/pull
- single-object rekey
- safe overwrite defaults (`--force` required)

### v1.1

- Tigris-based enrollment invites (short-lived, single-use)
- `envlock enroll invite`
- `envlock enroll join`
- `envlock enroll list`
- `envlock enroll approve/reject`
- optional rekey on approval (`--rekey <env>` / `--rekey-all`)

## Encryption and Key Choices

### Encryption library

Planned default crypto stack:

- [`filippo.io/age`](https://pkg.go.dev/filippo.io/age) for recipient-based encryption (`X25519` recipients)

Why:

- modern, well-understood file encryption UX
- supports multi-recipient encryption cleanly
- avoids designing a custom crypto format in v1

### Private key storage (v1)

Private keys are stored locally as files (no passphrase in v1), e.g.:

- macOS/Linux: `~/.config/envlock/keys/default.agekey`

Implications:

- simple and cross-platform
- easy to automate
- rely on OS account security and full-disk encryption
- file permissions must be strict (`0600`)

## Project File Layout

### Local machine files (private)

- `~/.config/envlock/keys/default.agekey`
- `~/.config/envlock/credentials.toml` (planned, per-machine Tigris credentials, `0600`)

### Project files (safe to commit)

- `./.envlock/project.toml`
- `./.envlock/recipients.json`

## Tigris Object Layout (Planned)

Object keys live under:

- `<app-name>/`

Examples:

- `my-app/.envlock`
- `my-app/worker.envlock`
- `my-app/secrets/main.envlock`

Planned internal enrollment metadata prefixes (v1.1):

- `<app-name>/_enroll/invites/...`
- `<app-name>/_enroll/requests/...`

## Install (Development)

Prerequisites:

- Go 1.23+

Install (Go):

```bash
go install github.com/jasonchiu/envlock@latest
```

Notes:

- The module path is now `github.com/jasonchiu/envlock`.
- If you previously used the `envlock-com` repo/module path, update local scripts and docs to the new import/install path.

Build (local checkout):

```bash
go build .
```

Run directly:

```bash
go run . --help
```

## Credentials (Per-Machine CLI Install)

Recommended precedence for Tigris credentials (planned):

1. Environment variables (`TIGRIS_ACCESS_KEY`, `TIGRIS_SECRET_KEY`, `TIGRIS_ENDPOINT`, `TIGRIS_REGION`, `TIGRIS_BUCKET`)
2. `~/.config/envlock/credentials.toml` (machine-local, not in git, file mode `0600`)
3. Project config (`.envlock/project.toml`) for non-secret defaults like bucket/prefix/endpoint only

Notes:

- `.env` is best for local development convenience only.
- Once installed as a CLI, machine-level credentials should live in the user config directory or be injected via shell environment.
- Project files should not contain secrets.

## Quick Start (Current Implemented Commands)

### 1. Generate a local device key

```bash
envlock init
```

This creates a local age private key file and prints:

- device name
- public key
- short fingerprint

Optional flags:

```bash
envlock init --name "mbp-personal"
envlock init --key-name default
```

### 2. Initialize a project in your repo

```bash
envlock project init --bucket my-tigris-bucket
```

By default, `envlock` infers the app name from the current folder name (for example, `/path/to/worker` becomes `worker`). You can still override this with `--app`.

This creates:

- `.envlock/project.toml`
- `.envlock/recipients.json`

It also auto-adds the current machine's public key as the first active recipient.

### 3. Inspect project config

```bash
envlock project show
```

### 4. Inspect status

```bash
envlock status
```

Shows:

- local key path and public key
- current project config (if present)
- recipient counts

### 5. Manage recipients (manual fallback path)

List recipients:

```bash
envlock recipients list
envlock recipients list --all
```

Add a recipient manually (advanced / fallback):

```bash
envlock recipients add macbook-air age1...
```

Remove (revoke) a recipient:

```bash
envlock recipients remove macbook-air
```

Hard delete recipient entry:

```bash
envlock recipients remove --hard macbook-air
```

Note: revoking/removing a recipient from the project file does not retroactively remove access from old ciphertext. You must rekey the encrypted object(s).

## Planned Workflow (End State)

### First machine

```bash
envlock init
envlock project init --bucket my-bucket
envlock push --in .env --object .envlock
```

### Add a new machine (v1.1 invite flow)

Trusted machine:

```bash
envlock enroll invite --ttl 15m
```

New machine:

```bash
envlock init
envlock enroll join --token <invite-token>
```

Trusted machine approves and optionally rekeys:

```bash
envlock enroll list
envlock enroll approve <request-id> --rekey-object .envlock
```

New machine can pull/decrypt:

```bash
envlock pull --object .envlock --out .env
```

## Overwrite Safety Model (Planned)

### `push`

Default behavior:

- fail if remote object exists
- show remote metadata (ETag/size/mtime)
- require `--force` to overwrite

Reason: scriptable, safe, low complexity.

### `pull`

Default behavior:

- fail if output file exists
- require `--force` to overwrite
- optional `--backup` to create a timestamped backup before replacing
- atomic writes (temp + rename)

Reason: avoid accidental local `.env` clobbering.

## Rekey Behavior (Planned)

Single-object rekey (v1):

- add recipient to an existing encrypted env blob
- remove recipient from an existing encrypted env blob

Examples:

```bash
envlock rekey --object .envlock --add-recipient age1...
envlock rekey --object .envlock --remove-recipient old-laptop
```

Important for lost/compromised devices:

- rekeying changes who can decrypt future ciphertext
- you should also rotate the actual secrets inside the `.env`

## Tigris Enrollment Invite Model (Planned v1.1)

Design choice: auto-approve enrollment only when the new machine presents a short-lived invite token created by a trusted machine.

Why this model:

- removes manual public-key sharing in the normal flow
- keeps per-device public/private key security model
- avoids treating Tigris access alone as sufficient trust

Invite properties:

- short TTL (default 15 minutes)
- single-use
- scoped to project/app

## Threat Model and Limitations

### Threats envlock addresses

- remote storage compromise without device private keys
- accidental plaintext `.env` storage in Tigris
- ad hoc secret sharing across machines without auditability

### Limitations (by design for v1)

- no password mode
- no OS keychain integration
- no recovery/offline admin key yet
- no batch rekey (single object only)
- no QR onboarding (deferred because Tigris invite flow is preferred)

## Troubleshooting (Current)

### `envlock project init` says key is missing

Run:

```bash
envlock init
```

first to create the local device key.

### `project config not found`

Run commands from the project root (the directory containing `.envlock/project.toml`) or initialize the project with `envlock project init`.

### Recipient removed but still can decrypt old file

That is expected until you rekey the encrypted object(s). Recipient config controls future encryption intent, not past ciphertext.

## Development Notes

Current focus:

1. local crypto round-trip (`encrypt` / `decrypt`)
2. Tigris push/pull
3. rekey
4. enrollment invites

Suggested internal package layout:

- `main.go` (CLI entrypoint)
- `internal/app/`
- `internal/config/`
- `internal/keys/`
- `internal/recipients/`
- `internal/crypto/` (planned)
- `internal/storage/s3/` (planned)
- `internal/enroll/` (planned)

## FAQ

### Is this just reinventing `age`?

Partly. `envlock` builds on the same crypto model, but the product goal is `.env`-specific UX + Tigris storage + onboarding/rekey workflows.

### Why not just AirDrop the `.env`?

AirDrop is great for nearby one-off transfers. `envlock` is meant to improve repeatability, remote access, and secure device lifecycle management over time.

### Why not trust Tigris credentials alone?

Because storage access and decryption authorization are different concerns. A machine should still need a device private key to decrypt secrets.

## Roadmap

- [ ] local encrypt/decrypt commands
- [ ] Tigris push/pull
- [ ] single-object rekey
- [ ] Tigris invite enrollment
- [ ] `--if-match`/ETag concurrency guard
- [ ] batch rekey (prefix)
- [ ] optional offline recovery key
- [ ] optional macOS Keychain backend

## License

TBD
