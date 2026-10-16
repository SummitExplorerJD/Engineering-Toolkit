# Go Conventions

- Preserve the module path declared in `go.mod` unless a separately approved migration changes it.
- Use standard Go repository structure: `cmd/engkit` for composition and `internal/` for implementation packages.
- Keep packages cohesive, names short, and dependency direction inward toward domain contracts.
- Accept `context.Context` at operation boundaries that may block, perform I/O, or invoke external tools.
- Return errors with actionable context while preserving causes for inspection.
- Avoid global mutable state and hidden filesystem or network access.
- Use argument-safe process execution; never construct shell commands from untrusted strings.
- Make filesystem roots explicit and validate resolved paths before writes.
- Prefer standard-library functionality. Every third-party dependency requires rationale and approval.
- Add table-driven unit tests for domain rules and boundary-focused integration tests for storage and adapters.
- Run `gofmt`, `go vet`, and the relevant `go test` scopes for implementation changes.
- Keep platform-specific code behind narrow interfaces and use Go build constraints only when behavior genuinely differs.

