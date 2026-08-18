#!/usr/bin/env bash
# govulncheck gated by the documented-acceptance registry
# (vulncheck-accepted.json, enforced by internal/tools/vulngate). ci.yml's
# vulncheck job and `make vulncheck` both run this script, so CI and a
# contributor's laptop apply the identical policy: every symbol-level finding
# either blocks the build or is listed in the registry with a written
# justification and an unexpired review_by date.
set -uo pipefail
cd "$(dirname "$0")/.."

out="$(mktemp)"
trap 'rm -f "$out"' EXIT

# `-format json` exits 0 even when findings exist (older govulncheck versions
# still exit 3 there); both mean "the scan completed" and the gate decides
# from the stream. Any other exit is an operational failure — network, flags,
# build errors — and must fail the check rather than pass it quietly. The
# gate separately refuses a stream with no scanner config message, so a scan
# that silently produced nothing cannot read as clean.
govulncheck -format json ./... >"$out"
code=$?
if [ "$code" -ne 0 ] && [ "$code" -ne 3 ]; then
  cat "$out"
  echo "vulncheck: govulncheck failed with exit $code" >&2
  exit "$code"
fi

# Built rather than `go run`, which collapses every nonzero exit to 1 and
# would lose the gate's govulncheck-mirroring exit codes (0 pass, 3 blocked).
gate="$(mktemp)"
trap 'rm -f "$out" "$gate"' EXIT
go build -o "$gate" ./internal/tools/vulngate
exec "$gate" -findings "$out" -accepted vulncheck-accepted.json
