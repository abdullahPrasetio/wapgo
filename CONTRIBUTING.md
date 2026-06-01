# Contributing to wapgo

Thank you for considering a contribution!

## How to Contribute

1. **Fork** the repository and create a branch from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Make your changes** following the conventions below.

3. **Run the full check suite locally** before pushing:
   ```bash
   make check
   ```
   This runs: lint · gosec · govulncheck · tests + coverage gate.

4. **Open a Pull Request** against `main`. The CI pipeline must be fully green.

## Branch Naming

| Prefix | Use |
|--------|-----|
| `feat/` | New feature |
| `fix/` | Bug fix |
| `chore/` | Maintenance, deps, CI |
| `docs/` | Documentation only |

## Commit Messages (Conventional Commits)

```
feat(auth): add RBAC role check to JWT middleware
fix(httpclient): respect caller deadline in retry loop
chore(deps): bump gofiber/fiber to v2.52.14
docs(readme): add kubernetes quickstart
```

## Code Conventions

- **Clean Architecture** — keep layer boundaries: Handler → Usecase → Repository (via interface). No concrete cross-layer imports.
- **ENV-first config** — all tuneable values must be readable from ENV via Viper. Add explicit `BindEnv` to `config.go`.
- **No magic** — DI is manual via constructor in `cmd/api/main.go`.
- **No comments that describe the what** — identifiers should be self-documenting. Only add a comment to explain a non-obvious *why*.
- **Test coverage > 80 %** — every new package must ship with tests. Run `make coverage` to check.
- **Security by default** — see [SECURITY.md](SECURITY.md). Run `gosec ./...` before opening a PR.

## Definition of Done (per PR)

- [ ] `go build ./...` + `go vet ./...` clean.
- [ ] `golangci-lint run ./...` clean.
- [ ] `gosec ./...` — no new findings.
- [ ] `govulncheck ./...` — no new CVEs introduced.
- [ ] Unit tests covering the changed packages, coverage > 80 %.
- [ ] `ExampleXxx` function or entry in `examples/` if adding a new public feature.
- [ ] No TODOs / placeholders in the diff.
- [ ] PR description explains *why*, not just *what*.

## Running Tests

```bash
# Unit tests (no Docker needed)
make test

# With race detector
go test -race ./...

# Integration tests (requires Docker)
go test -tags=integration -v ./internal/integration/...

# Coverage report
make coverage
```

## Project Layout

See [devplan.md](devplan.md) §5 for the full folder structure and the phase each component belongs to.

## Security Vulnerability Reporting

Please read [SECURITY.md](SECURITY.md) before opening any issues related to security.
