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
- `go install github.com/jasonchiu/envlock@latest` -> install published CLI

## Quick Facts
- Module path: `github.com/jasonchiu/envlock`
- CLI entrypoint: `main.go`
- Binary name: `envlock`
- Core logic packages: `internal/app`, `internal/config`, `internal/keys`, `internal/recipients`

## Hard Invariants
- `main.go` is the installable package root so `go install github.com/jasonchiu/envlock@latest` works.
- Keep app logic under `internal/` packages; `main.go` should remain a thin CLI bootstrap.
- Do not commit plaintext project secrets or machine-local credentials.
- Project metadata under `.envlock/` may be committed only when it is non-secret (see `README.md`).

## Project Structure
- `main.go` -> CLI bootstrap
- `internal/app/` -> command dispatch and CLI behavior
- `internal/config/` -> dotenv/project config handling
- `internal/keys/` -> local age key management
- `internal/recipients/` -> recipient store and validation
- `docs/` -> design notes/specs/prompts

## Key Paths
- CLI entrypoint: `main.go`
- Command routing: `internal/app/app.go`
- Dotenv loading: `internal/config/dotenv.go`
- Project config model/load/save: `internal/config/config.go`
- Recipient storage: `internal/recipients/store.go`
- Dev workflows: `Taskfile.yml`

## Reference Docs
- `README.md` -> installation, quick start, credential model, roadmap
- `docs/SPEC.md` -> product/spec details
- `docs/prompt-plan1.md`, `docs/prompt-plan2.md` -> planning prompts/notes
- `changelog.md` -> release/version history
