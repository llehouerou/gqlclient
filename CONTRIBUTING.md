# Contributing

Thanks for your interest in `gqlclient`. This is a small library; this guide
is correspondingly short.

## Development environment

A Nix flake provides the toolchain (Go, golangci-lint, golines,
goimports-reviser, delve). With Nix and direnv installed:

```sh
direnv allow         # or: nix develop
```

Without Nix: install Go ≥ 1.25 and `golangci-lint` v2.6+ on your
PATH. The `Makefile` works either way.

## Common commands

```sh
make build              # go build ./...
make test               # go test ./...
make test-coverage      # generates coverage.html
make lint               # golangci-lint run ./...
make format             # golines (80-col) + goimports-reviser
make vet                # go vet ./...
```

## Conventions

- **Commit messages**: conventional-commits style — `type(scope): subject`.
  Existing scopes: `decode`, `query`, `lint`, `ci`, `docs`. Use `chore`
  or `refactor` when nothing else fits.
- **Linting**: CI gates on `golangci-lint run`. The config in
  `.golangci.yml` is strict by design; add `//nolint:linter // reason`
  inline rather than disabling the linter globally.
- **Tests**: TDD when fixing bugs — write the failing test first, then
  the fix. `synctest` is preferred over `time.After` for timing-sensitive
  tests (Go 1.25+).
- **API stability**: pre-1.0. Breaking changes are recorded under the
  `Unreleased` heading in `CHANGELOG.md`.

## Pull request process

1. Fork and create a topic branch from `master`.
2. Make changes; ensure `make lint test` passes.
3. Open a PR. CI runs lint, tidy check, and tests on Linux (Nix), macOS,
   and Windows. All must pass before merge.
4. Squash-merge is preferred to keep the history linear.

## Reporting bugs

Open an issue with:
- Go version (`go version`).
- A minimal reproducer if possible — a struct + the GraphQL response
  bytes is usually enough.
- The actual vs. expected behavior.

For security vulnerabilities, see [`SECURITY.md`](SECURITY.md) instead
of the public issue tracker.
