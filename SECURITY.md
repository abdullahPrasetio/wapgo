# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| `main`  | ✅ Yes    |
| < v1.0  | ❌ No     |

## Reporting a Vulnerability

**Please do NOT open a public GitHub issue for security vulnerabilities.**

Report security issues privately by emailing **temancode@gmail.com** with the subject line:

```
[SECURITY] wapgo — <short description>
```

Include:
- Description of the vulnerability and potential impact
- Steps to reproduce or a proof-of-concept (if available)
- Affected versions / components
- Suggested fix (optional)

You will receive an acknowledgement within **48 hours**. We aim to release a patch within **14 days** for critical issues.

Once a fix is released, the vulnerability will be disclosed publicly in the release notes with appropriate credit to the reporter.

## Security Controls Built Into the Framework

| Area | Control |
|---|---|
| **Secrets** | Never stored in code; loaded from ENV / K8s Secrets only. Field redaction in logs. |
| **HTTP inbound** | HSTS, X-Content-Type-Options, X-Frame-Options, CSP headers. Rate-limit per-IP. Body size capped at 4 MB. |
| **Auth** | JWT HS256 with pinned algorithm (`alg:none` rejected). Validates `exp`, `iat`, `iss`, `aud`. Secret enforced ≥ 32 bytes. |
| **HTTP outbound** | TLS verify ON. SSRF guard blocks loopback / private / link-local. Response body capped. |
| **Database** | GORM parameterized queries (no string concatenation). |
| **Messaging** | DLQ for poison messages. TLS/SASL support via ENV. |
| **Observability** | `/metrics` returns 404 in `APP_ENV=production`. |
| **Container** | Distroless base, non-root UID 65532, read-only root filesystem, all capabilities dropped. |
| **Kubernetes** | `securityContext` enforces non-root + read-only FS + drop ALL caps. `NetworkPolicy` limits pod-to-pod traffic. |
| **Supply chain** | `go.sum` verified, deps pinned. `govulncheck` + `gosec` + `gitleaks` + `trivy` in CI. Dependabot enabled. |

## CI Security Gates

All of the following must pass before a PR can be merged:

| Gate | Tool |
|---|---|
| SAST | `gosec` |
| Dependency CVE scan | `govulncheck` |
| Secret scan | `gitleaks` |
| Image scan | `trivy` (CRITICAL + HIGH = fail) |
| Static analysis | `golangci-lint` |
| Test coverage | `go test` — gate ≥ 80 % |

## Pre-commit Hooks

Install the pre-commit hooks to catch issues before they reach CI:

```bash
# Install gitleaks
brew install gitleaks         # macOS
# or: https://github.com/gitleaks/gitleaks/releases

# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run manually before committing
gitleaks protect --staged
gosec ./...
```
