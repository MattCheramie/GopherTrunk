/*
 * GopherTrunk learning-path progress store. Loaded (deferred) BEFORE learn.js on
 * the learn layouts. Owns the localStorage layer that learn.js used to inline,
 * and — when a user is signed in (see auth.js / window.GT) — mirrors progress to
 * a Supabase `learn_progress` table so it follows the learner across devices.
 *
 * Design guarantees:
 *   - Reads/writes stay SYNCHRONOUS against localStorage, so learn.js keeps
 *     painting instantly and works with this file present but signed out, or
 *     with the network down, or with Supabase unconfigured.
 *   - Cloud writes are fire-and-forget and DEBOUNCED (never block the UI).
 *   - On sign-in, local and cloud progress are UNION-merged (a completion is
 *     never lost), then `gt:progress-hydrated` fires so learn.js repaints.
 *
 * Public surface (what learn.js calls): window.GTStore.load(path) -> string[],
 * window.GTStore.save(path, list). Same keys/semantics as the old inline store,
 * so anonymous behaviour is byte-for-byte identical.
 */
(function () {
  'use strict';

  var STORE = 'gt-learn-progress';   // key prefix: gt-learn-progress:<pathId>
  var LEGACY_PATH = 'rf-sdr';        // where the original single-path store belongs

  function storeKey(path) { return STORE + ':' + path; }

  // One-time migration of the original single-path store (bare RF slugs under
  // `gt-learn-progress`) into the rf-sdr key. Mirrors learn.js's own migration
  // so whichever loads first wins; both are idempotent.
  (function migrateLegacy() {
    try {
      var legacy = localStorage.getItem(STORE);
      if (legacy && !localStorage.getItem(storeKey(LEGACY_PATH))) {
        localStorage.setItem(storeKey(LEGACY_PATH), legacy);
      }
      if (legacy) localStorage.removeItem(STORE);
    } catch (e) {}
  })();

  function readLocal(path) {
    try {
      var raw = localStorage.getItem(storeKey(path));
      var arr = raw ? JSON.parse(raw) : [];
      return Array.isArray(arr) ? arr : [];
    } catch (e) { return []; }
  }
  function writeLocal(path, list) {
    try { localStorage.setItem(storeKey(path), JSON.stringify(list)); } catch (e) {}
  }
  function localPaths() {
    var out = [];
    try {
      for (var i = 0; i < localStorage.length; i++) {
        var k = localStorage.key(i);
        if (k && k.indexOf(STORE + ':') === 0) out.push(k.slice(STORE.length + 1));
      }
    } catch (e) {}
    return out;
  }

  /* ---- Cloud sync (only engages while signed in) ---- */

  function signedIn() {
    return !!(window.GT && window.GT.isConfigured && window.GT.user && window.GT.supabase);
  }

  var queue = [];          // { type:'add'|'del', path, slug }
  var flushTimer = null;

  function scheduleFlush() {
    if (flushTimer) return;
    flushTimer = setTimeout(flush, 800);
  }
  function flush() {
    flushTimer = null;
    if (!signedIn()) { queue = []; return; }
    var ops = queue; queue = [];
    var uid = window.GT.user.id;
    var client = window.GT.supabase;

    var adds = [];
    ops.forEach(function (o) {
      if (o.type === 'add') adds.push({ user_id: uid, path_id: o.path, slug: o.slug });
    });
    if (adds.length) {
      client.from('learn_progress')
        .upsert(adds, { onConflict: 'user_id,path_id,slug' })
        .then(function () {}, function () {});
    }
    ops.forEach(function (o) {
      if (o.type !== 'del') return;
      client.from('learn_progress').delete()
        .match({ user_id: uid, path_id: o.path, slug: o.slug })
        .then(function () {}, function () {});
    });
  }

  // Merge cloud <-> local on sign-in (and on any page load while already signed
  // in — that IS the cross-device pull). Union per path; push local-only rows up.
  function hydrateFromCloud() {
    if (!signedIn()) return;
    var uid = window.GT.user.id;
    window.GT.supabase.from('learn_progress').select('path_id, slug')
      .then(function (res) {
        if (!res || res.error) return;
        var rows = res.data || [];
        var cloud = {};
        rows.forEach(function (r) {
          (cloud[r.path_id] = cloud[r.path_id] || []).push(r.slug);
        });

        var paths = {};
        localPaths().forEach(function (p) { paths[p] = 1; });
        Object.keys(cloud).forEach(function (p) { paths[p] = 1; });

        var toUpsert = [];
        Object.keys(paths).forEach(function (p) {
          var local = readLocal(p);
          var cloudList = cloud[p] || [];
          var union = local.slice();
          cloudList.forEach(function (s) { if (union.indexOf(s) === -1) union.push(s); });
          writeLocal(p, union);
          union.forEach(function (s) {
            if (cloudList.indexOf(s) === -1) toUpsert.push({ user_id: uid, path_id: p, slug: s });
          });
        });

        if (toUpsert.length) {
          window.GT.supabase.from('learn_progress')
            .upsert(toUpsert, { onConflict: 'user_id,path_id,slug' })
            .then(function () {}, function () {});
        }
        try { document.dispatchEvent(new CustomEvent('gt:progress-hydrated')); } catch (e) {}
      }, function () {});
  }

  /* ---- Public API (mirrors the old inline store) ---- */

  var GTStore = window.GTStore = {
    load: function (path) { return readLocal(path); },
    save: function (path, list) {
      var prev = readLocal(path);
      writeLocal(path, list);
      if (signedIn()) {
        list.forEach(function (s) {
          if (prev.indexOf(s) === -1) queue.push({ type: 'add', path: path, slug: s });
        });
        prev.forEach(function (s) {
          if (list.indexOf(s) === -1) queue.push({ type: 'del', path: path, slug: s });
        });
        if (queue.length) scheduleFlush();
      }
    }
  };

  // Hydrate whenever we transition into a signed-in state (covers both a fresh
  // sign-in and a returning visitor whose session restores on load).
  var wasSignedIn = false;
  if (window.GT && typeof window.GT.onChange === 'function') {
    window.GT.onChange(function (user) {
      var now = !!user;
      if (now && !wasSignedIn) hydrateFromCloud();
      wasSignedIn = now;
    });
  }
})();
