# AGENTS.md

Guidance for coding agents working on distraction.today.

## Project Overview

A daily quote and text service written in Go (`github.com/icco/distraction.today`) served at <https://distraction.today>.

## Commands

```sh
go test ./...       # Run tests
go vet ./...        # Vet Go code
go run main.go      # Run locally (port 8080 by default)
go build .          # Build binary
```

## Architecture & Layout

- `main.go` — Entrypoint, Chi router, and HTTP middleware setup.
- `quotes.go` / `quotes_test.go` — Quote retrieval, storage, and date indexing.
- `templates/` — HTML rendering templates.

## Conventions

- Follow icco Go conventions (`github.com/icco/gutil` for logging and middleware).
- PR titles and commits must follow Conventional Commits with lowercase subjects.
- Ensure all tests pass before submitting PRs.
