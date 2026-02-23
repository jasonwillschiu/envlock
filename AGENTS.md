# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go CLI project (`envlock`). Module path is `github.com/jasonchiu/envlock`. The entrypoint is `main.go`, and application logic lives under `internal/` by domain (`app`, `config`, `keys`, `recipients`). Build outputs go to `bin/` (for example `bin/envlock`). Design notes and planning docs live in `docs/`. Root files include `Taskfile.yml` for common workflows and `README.md` for product scope/status.

## Build, Test, and Development Commands
Prefer `task` (from `Taskfile.yml`) for local development:

- `task build` - builds the CLI to `bin/envlock`
- `task run -- --help` - runs the CLI locally with arguments
- `go install github.com/jasonchiu/envlock@latest` - installs the CLI from the renamed module root
- `task test` - runs `go test -v ./...`
- `task fmt` - formats Go packages with `go fmt ./...`
- `task vet` - runs `go vet ./...`
- `task lint` - runs `golangci-lint` and `gopls` diagnostics (best effort)

Direct Go commands also work, e.g. `go build .` and `go run . --help`.

## Coding Style & Naming Conventions
Use standard Go conventions and keep code `gofmt`-formatted. Package names should be short, lowercase, and noun-based (as in `internal/keys`, `internal/config`). Exported identifiers use `PascalCase`; unexported identifiers use `camelCase`. Prefer small functions with explicit error returns and wrap errors with context.

## Testing Guidelines
There are currently no `*_test.go` files in the repo. Add tests next to the code they cover (same package directory) using Go’s testing package. Name files `*_test.go` and tests `TestXxx`. Run all tests with `task test` (or `go test ./...`) before opening a PR. Add table-driven tests for CLI parsing/config logic where practical.

## Commit & Pull Request Guidelines
Git history currently contains only an initial commit, so no strict convention is established yet. Use concise, imperative commit subjects (for example: `add recipient validation`). Keep commits focused and logically grouped.

PRs should include: purpose/summary, key behavior changes, commands run (`task test`, `task lint`, etc.), and sample CLI output for user-facing changes. Link related issues/tasks when available.

## Security & Configuration Tips
Do not commit plaintext `.env` files, machine-local credentials, or private keys. Follow the project model in `README.md`: project metadata under `./.envlock/` may be safe to commit, but machine secrets (for example under `~/.config/envlock/`) are not.
