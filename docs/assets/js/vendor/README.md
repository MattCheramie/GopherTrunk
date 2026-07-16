# Vendored third-party JS

Same-origin copies of third-party libraries the docs site loads at runtime. The
docs site is built by `actions/jekyll-build-pages` with **no Node/Ruby build step**,
so browser dependencies are committed here rather than installed or pulled from a CDN.
Serving them from gophertrunk.org keeps the auth-critical path off third-party
availability and stays CSP-clean (`script-src 'self'`).

## `supabase.min.js` — `@supabase/supabase-js`

- **Version:** 2.58.0 (UMD build, `dist/umd/supabase.js` from the npm package).
- **Global:** exposes `window.supabase` with `createClient(url, anonKey)`.
- **Loaded by:** `_layouts/default.html` (after `supabase-config.js`, before `auth.js`).

### Upgrade procedure

```sh
VER=<new-version>
curl -sSL "https://registry.npmjs.org/@supabase/supabase-js/-/supabase-js-${VER}.tgz" -o /tmp/sb.tgz
tar xzf /tmp/sb.tgz -C /tmp
# prepend the version header comment, then the bundle:
cp /tmp/package/dist/umd/supabase.js docs/assets/js/vendor/supabase.min.js
```

Then bump the version in the header comment of `supabase.min.js` and in this file.
Smoke-test in Node (the UMD references `self`, which Node lacks, so shim it):

```sh
node -e "global.self=globalThis; const sb=require('./docs/assets/js/vendor/supabase.min.js'); console.log(typeof sb.createClient)"
```
