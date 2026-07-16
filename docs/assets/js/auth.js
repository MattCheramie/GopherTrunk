/*
 * GopherTrunk auth core. Shared, site-wide. Loaded (deferred) from head.html
 * AFTER supabase-config.js and vendor/supabase.min.js, so it runs before the
 * body/content scripts (store.js, learn.js, the account page).
 *
 * Progressive enhancement: if the config still holds placeholders, or the
 * vendored supabase library failed to load, this file goes dormant — it only
 * renders a static "Account" link in the nav and leaves the rest of the site
 * exactly as it was (localStorage-only progress). Nothing here is required for
 * a page to be usable.
 *
 * Exposes a small, stable surface on window.GT:
 *   GT.isConfigured        boolean — a real Supabase project is wired up
 *   GT.supabase            the supabase-js client (only when configured)
 *   GT.user / GT.session   current auth state (null when signed out)
 *   GT.ready               Promise resolving once the initial session loads
 *   GT.onChange(cb)        subscribe to auth state; fires immediately + on change
 *   GT.signInWithOAuth(provider)   'github' | 'google'
 *   GT.signInWithPassword(email, password)
 *   GT.signUpWithPassword(email, password)
 *   GT.signInWithMagicLink(email)
 *   GT.signOut()
 *   GT.deleteAccount()     calls the delete_own_account() RPC
 * A `gt:auth` DOM event fires on document for every state change too.
 */
(function () {
  'use strict';

  var cfg = window.GT_SUPABASE || {};
  var configured = !!(cfg.url && cfg.anonKey &&
    cfg.url.indexOf('YOUR-') === -1 && cfg.anonKey.indexOf('YOUR-') === -1);

  var GT = window.GT = window.GT || {};
  GT.isConfigured = configured && !!window.supabase;
  GT.user = null;
  GT.session = null;

  var listeners = [];
  GT.onChange = function (cb) {
    if (typeof cb !== 'function') return function () {};
    listeners.push(cb);
    try { cb(GT.user, GT.session); } catch (e) {}
    return function () { var i = listeners.indexOf(cb); if (i !== -1) listeners.splice(i, 1); };
  };
  function emit() {
    listeners.forEach(function (cb) { try { cb(GT.user, GT.session); } catch (e) {} });
    try {
      document.dispatchEvent(new CustomEvent('gt:auth', {
        detail: { user: GT.user, session: GT.session }
      }));
    } catch (e) {}
  }

  function displayName(u) {
    if (!u) return 'Account';
    var m = u.user_metadata || {};
    return m.user_name || m.preferred_username || m.name ||
      (u.email ? u.email.split('@')[0] : 'Account');
  }

  // Nav auth control — a link with [data-auth-nav] rendered in nav.html. Updated
  // in place; harmless if the element is absent (e.g. a minimal layout).
  function renderNav() {
    var el = document.querySelector('[data-auth-nav]');
    if (!el) return;
    var label = el.querySelector('.nav-auth__label');
    if (GT.user) {
      var name = displayName(GT.user);
      if (label) label.textContent = name;
      el.setAttribute('data-signed-in', '1');
      el.setAttribute('title', 'Account — ' + name);
      el.setAttribute('aria-label', 'Account — signed in as ' + name);
    } else {
      if (label) label.textContent = GT.isConfigured ? 'Sign in' : 'Account';
      el.removeAttribute('data-signed-in');
      el.setAttribute('title', GT.isConfigured ? 'Sign in' : 'Account');
      el.setAttribute('aria-label', GT.isConfigured ? 'Sign in' : 'Account');
    }
  }
  document.addEventListener('DOMContentLoaded', renderNav);

  // Dormant path: no real project or library missing. Keep the surface present
  // (so callers can feature-detect) but do nothing that needs a client.
  if (!GT.isConfigured) {
    GT.ready = Promise.resolve(null);
    var noop = function () {
      return Promise.reject(new Error('Supabase is not configured for this site.'));
    };
    GT.signInWithOAuth = GT.signInWithPassword = GT.signUpWithPassword =
      GT.signInWithMagicLink = GT.signOut = GT.deleteAccount = noop;
    return;
  }

  // Active path -------------------------------------------------------------
  var client = GT.supabase = window.supabase.createClient(cfg.url, cfg.anonKey, {
    auth: { persistSession: true, autoRefreshToken: true, detectSessionInUrl: true }
  });

  // OAuth / email links come back to the account page, where detectSessionInUrl
  // finishes the exchange. location.origin keeps prod (gophertrunk.org) and dev
  // (localhost) correct without hardcoding — register both in Supabase.
  var REDIRECT = location.origin + '/account/';

  GT.signInWithOAuth = function (provider) {
    return client.auth.signInWithOAuth({
      provider: provider, options: { redirectTo: REDIRECT }
    });
  };
  GT.signInWithPassword = function (email, password) {
    return client.auth.signInWithPassword({ email: email, password: password });
  };
  GT.signUpWithPassword = function (email, password) {
    return client.auth.signUp({
      email: email, password: password, options: { emailRedirectTo: REDIRECT }
    });
  };
  GT.signInWithMagicLink = function (email) {
    return client.auth.signInWithOtp({
      email: email, options: { emailRedirectTo: REDIRECT }
    });
  };
  GT.signOut = function () { return client.auth.signOut(); };
  GT.deleteAccount = function () { return client.rpc('delete_own_account'); };

  // Load the current session, then subscribe to changes. GT.ready lets callers
  // (account page, store.js) await the first resolution before painting.
  GT.ready = client.auth.getSession().then(function (res) {
    GT.session = (res && res.data) ? res.data.session : null;
    GT.user = GT.session ? GT.session.user : null;
    emit();
    return GT.session;
  }).catch(function () { emit(); return null; });

  client.auth.onAuthStateChange(function (_event, session) {
    GT.session = session || null;
    GT.user = session ? session.user : null;
    emit();
  });

  GT.onChange(renderNav);
})();
