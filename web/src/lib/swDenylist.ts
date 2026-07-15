// Paths the main console's service worker must NOT answer with its own
// index.html navigateFallback. Every daemon-mounted sibling console
// (Config Builder, Signal Lab, RF Scope, Crypto Lab) lives at a subpath under
// the service worker's "/" scope. Without an entry here the SW shadows the
// new-tab navigation to that subpath with our own cached index.html; the main
// SPA then boots at a path React Router has no route for and the tab opens
// blank. Keep this in sync with the external console links in
// src/nav/registry.ts (enforced by swDenylist.test.ts).
export const SW_NAVIGATE_FALLBACK_DENYLIST: RegExp[] = [
  /^\/api\//,
  /^\/metrics/,
  /^\/config\//,
  /^\/siglab\//,
  /^\/rfscope\//,
  /^\/cryptolab\//,
];
