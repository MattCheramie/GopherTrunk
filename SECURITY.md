# Security policy

GopherTrunk's threat model centres on operators running the daemon
on a private network or a single host they own (LAN deployment,
home lab, an EC2 box for a small fleet of SDR receivers). The
binary is single-process, single-host, and stores everything it
decodes to local disk; the API is gated by bearer-token
authentication (see [`docs/hardening.md`](docs/hardening.md)).
Optional TLS for the HTTP and gRPC servers is documented in the
same file.

## Supported versions

Security fixes land on the default branch (`main`). The most
recent tagged release receives back-ported fixes; older tags are
not maintained.

| Version    | Status     |
| ---------- | ---------- |
| `main`     | Supported  |
| Latest tag | Supported  |
| Older tags | Best-effort, no SLA |

## Reporting a vulnerability

**Do not open a public issue** for a suspected vulnerability.

Use **GitHub's private security advisory** workflow:

1. Visit the repository's **Security → Advisories** tab.
2. Click **Report a vulnerability** (or hit
   `https://github.com/mattcheramie/gophertrunk/security/advisories/new`).
3. Provide a description, reproducer (or minimal IQ capture / API
   request that triggers the issue), and the affected version /
   commit SHA.

GitHub's advisory workflow keeps the report private until a fix
ships. The maintainer aims for an initial response within 7 days
and a fix or mitigation within 30 days for critical issues; less
severe issues land on the regular development cadence.

If GitHub advisories aren't available to you (e.g. you're not on
GitHub), open a public issue marked `security: contact requested`
asking the maintainer to reach out by email — do **not** include
exploit details in the public issue.

## What counts as a vulnerability

In scope:

- Authentication / authorisation bypass on
  `/api/v1/{mutations,scanner,audio,...}` mutation endpoints
  (token validation, trusted-network parsing).
- Path traversal / arbitrary file write through
  `recordings.dir` or `talkgroup_file` config keys (the daemon
  reads / writes these on disk).
- SQL injection through the SQLite call-log queries (the
  `internal/storage` package uses parameterised statements; report
  any path that builds a query through string concatenation).
- Denial-of-service that crashes the daemon or wedges it via a
  malformed input from one of the supported wire formats (RTL-SDR
  IQ, control-channel frames, API JSON, gRPC requests).
- Memory-safety bugs (overflow, out-of-bounds slice access) in
  any non-test Go code, even if Go's bounds-checking turns them
  into runtime panics.

Out of scope:

- AMBE+2 patent-encumbered decode (see
  [`docs/vocoders.md`](docs/vocoders.md) for the licensing
  posture).
- "The daemon does what its config tells it to" — operator
  misconfiguration isn't a vulnerability.
- Local-only deployments where the operator already has root
  (you can't escalate above the privilege you started with).
- Issues that only manifest with `auth.mode: disabled` set —
  that's explicitly documented as the wide-open legacy mode, and
  operators who set it are opting out of the auth boundary.

## Cryptography

- The HTTP and gRPC servers use Go's stdlib `crypto/tls`. Cipher
  suite selection and TLS version negotiation follow Go's
  built-in policy (TLS 1.2 minimum, modern AEAD suites). Cert
  rotation requires a daemon restart.
- The bearer-token auth uses constant-time comparison
  (`crypto/subtle.ConstantTimeCompare`) — token-comparison side
  channels are not in scope by construction.

## Disclosure timeline expectations

| Severity | Initial response | Fix / mitigation |
| -------- | ---------------- | ---------------- |
| Critical (RCE, auth bypass, data loss) | ≤ 7 days | ≤ 30 days |
| High (DoS, partial info disclosure) | ≤ 14 days | ≤ 60 days |
| Medium / low | ≤ 30 days | Next release cycle |

Fixes are coordinated via the GitHub advisory; once a fix is
merged on `main` and (where applicable) back-ported to the latest
tag, the advisory is published and a CVE requested for issues at
the high or critical level.
