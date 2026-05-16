// Seal web/node_modules/ as a Go submodule sentinel so the parent
// module's `go list ./...` / `go test ./...` / `go build ./...` skip
// the stray Go packages npm dependencies occasionally ship inside
// their tarballs (e.g. flatted/golang/pkg/flatted/flatted.go).
//
// Wired as the npm `postinstall` hook in package.json so it runs
// automatically after every `npm install` / `npm ci` (which is the
// only time node_modules/ is wiped + recreated). Safe to run by hand
// (idempotent) and a no-op if node_modules/ doesn't exist yet.
//
// Background: Go's recursive package discovery normally walks every
// directory under the module root; the only directory-name-level
// skips are `_*`, `.*`, `testdata`, `vendor`, and a `go.mod` marking
// a nested module. Dropping this sentinel here is the smallest
// change that hides node_modules from `go list ./...`.

import { existsSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const target = join(here, "..", "node_modules", "go.mod");

if (!existsSync(join(here, "..", "node_modules"))) {
  // npm hasn't actually populated node_modules yet (rare — happens
  // when the install errored before reaching postinstall). Nothing
  // to seal.
  process.exit(0);
}

const body = `// Sentinel — see web/scripts/seal-node-modules.mjs.
// Marks node_modules/ as a separate Go module so the parent
// module's package discovery skips the stray Go packages npm
// dependencies sometimes ship (e.g. flatted/golang/pkg/flatted/
// flatted.go). Recreated automatically by npm install /
// npm ci's postinstall hook; safe to delete by hand.

module gophertrunk-web-node-modules-sentinel

go 1.25
`;

writeFileSync(target, body);
console.log(`seal-node-modules: wrote ${target}`);
