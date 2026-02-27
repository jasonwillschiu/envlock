# 2.0.0 - Update: Refactor to core/ + feature/ package layout
- Replace `internal/` package tree with `core/` (shared logic) and `feature/` (domain features)
- Add `cmd/server/` entrypoint for server-mode binary
- Add `core/auth`, `core/authstate`, `core/router`, `core/serverapi` and `feature/cliauth` packages
- Update `main.go` imports to `core/config` and `feature/cli`
- THIS UPDATE HAS NOT BEEN HUMAN TESTED YET

# 1.1.0 - Add: Tigris-backed enroll invite flow
- Implement `enroll invite/join/list/approve/reject` and `login`/`whoami`/`devices`/`requests` commands
- Add Tigris-backed remote store for recipients and enrollment metadata (replaces local `recipients.json`)
- Add `internal/backend`, `internal/enroll`, `internal/remote`, `internal/tigris` packages
- Add two-machine quickstart with invite flow and install verification to README
- Add `task release` helper and `go.mod` AWS SDK v2 dependencies

# 1.0.0 - Update: Rename module and root CLI install
- Change module path from `github.com/jasonchiu/envlock-com` to `github.com/jasonchiu/envlock`
- Move the CLI entrypoint to `main.go` so `go install github.com/jasonchiu/envlock@latest` works
- Update Taskfile and README commands to build and run from the module root (`go build .`, `go run .`)
- Update agent docs to reflect the renamed module path and root entrypoint

# 0.2.0 - Update: Project init defaults and prefixes
- `project init --app` is now optional and infers the app name from the current directory
- Default project prefix is now `<app>` instead of `envlock/<app>`
- Dotenv loading now also skips when `ENV=production`
- README and spec examples/docs updated for object-based naming and machine-local Tigris credentials

# 0.1.0 - Initial commit: Bootstrap envlock CLI
- Initial repository commit
