# CLAUDE.md

## Must Follow
- Use `task` targets for routine development work when possible.
- Build and run from the module root (`main.go`), not `cmd/envlock/`.
- Keep secrets and private keys out of git (`.env`, machine credentials, private key files).
- Treat `README.md` as the canonical human-facing product/setup reference.

## Essential Commands
- `task build` -> builds `bin/envlock`
- `task run -- --help` -> runs the CLI locally
- `task test` -> `go test -v ./...`
- `task fmt` -> `go fmt ./...`
- `task vet` -> `go vet ./...`
- `task lint` -> golangci-lint + gopls checks (best effort)
- `task release` -> runs `mdrelease` helper
- `go install github.com/jasonchiu/envlock@latest` -> install published CLI

## Quick Facts
- Module path: `github.com/jasonchiu/envlock`
- CLI entrypoint: `main.go`
- Binary name: `envlock`
- Core logic packages: `core/config`, `core/keys`, `core/remote`, `core/tigris`, `core/backend`, `core/auth`, `core/authstate`, `core/router`, `core/serverapi`
- Feature packages: `feature/cli`, `feature/cliauth`, `feature/enroll`, `feature/recipients`

## Hard Invariants
- `main.go` is the installable package root so `go install github.com/jasonchiu/envlock@latest` works.
- Shared logic lives in `core/`; domain features live in `feature/`; `main.go` remains a thin CLI bootstrap.
- Do not commit plaintext project secrets or machine-local credentials.
- Project metadata under `.envlock/` may be committed only when it is non-secret (see `README.md`).

## Project Structure
- `main.go` -> CLI bootstrap (installable root)
- `cmd/server/` -> server-mode binary entrypoint
- `core/config/` -> dotenv/project config handling
- `core/keys/` -> local age key management
- `core/remote/` -> remote store interface for Tigris metadata
- `core/tigris/` -> Tigris S3-compatible client
- `core/backend/` -> shared storage backend abstraction
- `core/auth/` -> CLI auth logic
- `core/authstate/` -> auth state store
- `core/router/` -> server routing
- `core/serverapi/` -> server API client
- `feature/cli/` -> command dispatch and CLI behavior
- `feature/cliauth/` -> CLI auth handler
- `feature/enroll/` -> enrollment invite metadata (Tigris-backed)
- `feature/recipients/` -> recipient store and validation
- `docs/` -> design notes/specs/prompts

## Key Paths
- CLI entrypoint: `main.go`
- Server entrypoint: `cmd/server/main.go`
- Command routing: `feature/cli/app.go`
- Dotenv loading: `core/config/dotenv.go`
- Project config model/load/save: `core/config/config.go`
- Recipient storage: `feature/recipients/store.go`
- Remote Tigris store: `core/remote/store.go`
- Enroll invite store: `feature/enroll/store.go`
- Tigris S3 client: `core/tigris/client.go`
- Dev workflows: `Taskfile.yml`

## Reference Docs
- `README.md` -> installation, quick start, credential model, roadmap
- `docs/SPEC.md` -> product/spec details
- `docs/prompt-plan1.md`, `docs/prompt-plan2.md` -> planning prompts/notes
- `changelog.md` -> release/version history
