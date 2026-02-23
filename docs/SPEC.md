# envlock Specification (Draft v1 / v1.1)

This document captures the locked implementation plan and command/data schemas for `envlock`.

## Goals

- Encrypt `.env` files while preserving bytes exactly
- Store encrypted blobs in Tigris (S3-compatible)
- Support per-device keys and multi-recipient access
- Support adding/removing devices via rekeying
- Hide manual pubkey sharing behind Tigris enrollment invites (v1.1)

## Non-Goals (v1)

- Secret value parsing/editing/merging
- Password-based file encryption mode
- OS keychain integration
- Batch rekey across prefixes
- Team policy/roles

## Crypto

- Library: `filippo.io/age`
- Recipient type: X25519 recipients
- Multi-recipient encryption: supported by default
- Payload: raw file bytes (exact preservation)

## Local Files

### Private keys (local only)

- Path: `~/.config/envlock/keys/<key-name>.agekey`
- Default key name: `default`
- File mode: `0600`
- Directory mode: `0700`

Private key file may include metadata comment:

```text
# envlock-device: <device-name>
AGE-SECRET-KEY-...
```

### Project config (safe to commit)

`./.envlock/project.toml`

```toml
version = 1
app_name = "my-app"
bucket = "my-bucket"
prefix = "envlock/my-app"
endpoint = "" # optional
```

### Recipients store (safe to commit)

`./.envlock/recipients.json`

```json
{
  "version": 1,
  "recipients": [
    {
      "name": "mbp-personal",
      "public_key": "age1...",
      "fingerprint": "abcd1234ef567890",
      "created_at": "2026-02-23T00:00:00Z",
      "status": "active",
      "source": "local-init",
      "note": "Added during project init"
    }
  ]
}
```

Status values:

- `active`
- `revoked`

## Tigris Object Layout

Base prefix:

- `envlock/<app-name>/`

Env blobs:

- `envlock/<app-name>/<env>.envlock`

Examples:

- `envlock/my-app/dev.envlock`
- `envlock/my-app/prod.envlock`

### Enrollment metadata (v1.1)

Internal prefixes (implementation detail):

- `envlock/<app-name>/_enroll/invites/<invite-id>.json`
- `envlock/<app-name>/_enroll/requests/<request-id>.json`

## Command Spec

## `envlock init`

Generate a local device keypair.

Flags:

- `--name <device-name>` optional (defaults to hostname)
- `--key-name <name>` default `default`
- `--force` overwrite existing key file

Behavior:

- creates local private key file
- prints public key and fingerprint

## `envlock status`

Show local key and project setup state.

Flags:

- `--key-name <name>` default `default`

## `envlock project init`

Initialize project config and recipients store.

Flags:

- `--app <name>` required
- `--bucket <bucket>` required
- `--prefix <prefix>` optional; default `envlock/<app>`
- `--endpoint <url>` optional
- `--key-name <name>` default `default`
- `--name <device-name>` optional recipient name override
- `--force` overwrite existing project config

Behavior:

- requires local key to exist
- writes `.envlock/project.toml`
- creates/updates `.envlock/recipients.json`
- auto-adds current machine as recipient (idempotent if duplicate)

## `envlock project show`

Display project config loaded from `./.envlock/project.toml`.

## `envlock recipients list`

List recipients.

Flags:

- `--all` include revoked recipients

## `envlock recipients add`

Manual fallback to add a recipient public key.

Usage:

```bash
envlock recipients add <name> <age-public-key> [--note <text>]
```

Validation:

- public key must parse as age X25519 recipient
- duplicate names and duplicate fingerprints rejected

## `envlock recipients remove`

Revoke or delete a recipient entry.

Usage:

```bash
envlock recipients remove <name|fingerprint>
```

Flags:

- `--hard` permanently delete instead of marking revoked

Notes:

- config change does not rekey existing ciphertext
- user must run `rekey` later (v1)

## Planned v1 Commands

### `envlock encrypt`

Local encrypt `.env` to `.envlock` using active recipients.

### `envlock decrypt`

Local decrypt `.envlock` to output path using local private key.

### `envlock push`

Encrypt and upload to Tigris.

Proposed flags:

- `--env <name>` (maps to `<env>.envlock` under project prefix)
- `--in <path>` default `.env`
- `--force` overwrite existing remote object

Default overwrite behavior:

- fail if remote object exists
- require `--force` to replace

### `envlock pull`

Download and decrypt from Tigris.

Proposed flags:

- `--env <name>`
- `--out <path>` default `.env`
- `--force` overwrite local output
- `--backup` create backup before overwrite

Default overwrite behavior:

- fail if output exists
- require `--force`
- atomic write

### `envlock rekey`

Single-object rekey in Tigris.

Proposed forms:

```bash
envlock rekey --env dev --add-recipient <age1...>
envlock rekey --env dev --remove-recipient <name|fingerprint>
```

Behavior:

- download ciphertext
- decrypt with local key
- re-encrypt to updated recipient set
- upload replacement (force/etag policy TBD in implementation)

## Enrollment (v1.1)

Chosen model: short-lived invite token from trusted machine (Option C).

### `envlock enroll invite`

Create a short-lived, single-use invite token.

Flags (planned):

- `--app <name>` or use project config
- `--ttl <duration>` default `15m`

Output:

- invite token (high-entropy string)
- invite ID
- expiry timestamp

### `envlock enroll join`

New machine joins using invite token.

Flags (planned):

- `--app <name>` or use project config
- `--token <token>` required
- `--name <device-name>` optional (default hostname)

Behavior:

- generates local keypair (if missing)
- uploads enrollment request to Tigris with pubkey/fingerprint
- request remains pending until approval

### `envlock enroll list`

List pending enrollment requests for a project.

### `envlock enroll approve`

Approve a pending request and add recipient.

Flags (planned):

- `--rekey <env>` optional
- `--rekey-all` optional

Behavior:

- validates invite token/TTL/single-use
- adds recipient to `.envlock/recipients.json`
- marks request approved and invite used
- optional rekey workflow

### `envlock enroll reject`

Reject a pending request and mark request terminal.

## Overwrite and Safety Policy

### Remote (`push`)

v1 default:

- fail on existing object
- require explicit `--force`

Future improvement:

- optimistic concurrency using ETag (`--if-match`)

### Local (`pull`)

v1 default:

- fail if output exists
- require explicit `--force`
- optional `--backup`
- atomic rename semantics

## Incident Response Guidance

If a device is lost/compromised:

1. revoke/remove recipient from project recipients
2. rekey affected encrypted blobs
3. rotate underlying secrets in the `.env` (API keys, credentials)

Rekey alone is not sufficient if secrets may have been exposed before revocation.

## Implementation Phases

1. Core local setup and recipients (current)
2. Local crypto round-trip (`encrypt`/`decrypt`)
3. Tigris push/pull
4. Single-object rekey
5. Enrollment invites (v1.1)
6. Polish + tests + README refinements
