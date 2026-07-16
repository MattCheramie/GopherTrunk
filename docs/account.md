---
layout: page
title: Account
description: Create a free GopherTrunk account or sign in — GitHub, Google, email, or a magic link. Syncs your learning-path progress and unlocks the community forum and a public profile.
permalink: /account/
nav_group: Community
hide_ctas: true
---

# Account

A free account is optional. It **syncs your learning-path progress** across devices, and
unlocks the **[community forum](/forum/)** and a **[public profile](/u/)**. Signed out,
everything still works — progress just lives in this browser.

<div class="account" data-account>
  <p class="account__loading" data-account-loading>Loading…</p>

  <section class="account__signin" data-account-signin hidden>
    <h2>Sign in or create an account</h2>
    <p class="account__lede">New here? Use any method below — the first time you sign in,
      your account is created automatically. Creating one lets you post in the forum and
      gives you a public profile you can fill out.</p>

    <div class="account__oauth">
      <button type="button" class="btn btn--secondary" data-oauth="github">Continue with GitHub</button>
      <button type="button" class="btn btn--secondary" data-oauth="google">Continue with Google</button>
    </div>

    <form class="account__form" data-form="password" novalidate>
      <h3>Email &amp; password</h3>
      <label>Email <input type="email" name="email" autocomplete="email" required></label>
      <label>Password <input type="password" name="password" autocomplete="current-password" minlength="6" required></label>
      <div class="account__form-actions">
        <button type="submit" class="btn" data-mode="signin">Sign in</button>
        <button type="submit" class="btn btn--secondary" data-mode="signup">Create account</button>
      </div>
    </form>

    <form class="account__form" data-form="magic" novalidate>
      <h3>Or get a magic link</h3>
      <label>Email <input type="email" name="email" autocomplete="email" required></label>
      <button type="submit" class="btn btn--secondary">Email me a sign-in link</button>
    </form>

    <p class="account__msg" data-account-msg role="status" aria-live="polite"></p>
  </section>

  <section class="account__panel" data-account-panel hidden>
    <h2>You're signed in</h2>
    <p>Signed in as <strong data-account-name>…</strong> <span class="account__email" data-account-email></span>.
       Your learning-path progress syncs automatically.</p>

    <p class="account__panel-links">
      <a class="btn" data-profile-edit href="/profile/">Edit your profile</a>
      <a class="btn btn--secondary" data-profile-view href="/u/">View public profile</a>
    </p>

    <p class="account__panel-actions">
      <button type="button" class="btn btn--secondary" data-signout>Sign out</button>
    </p>
    <details class="account__danger">
      <summary>Delete account</summary>
      <p>Permanently deletes your account and all synced progress. This can't be undone —
         local progress in this browser is kept.</p>
      <button type="button" class="btn btn--danger" data-delete>Delete my account</button>
    </details>
    <p class="account__msg" data-account-panel-msg role="status" aria-live="polite"></p>
  </section>

  <noscript>
    <p class="account__msg is-error">Signing in needs JavaScript. Your progress still
    works without it — it's just kept in this browser.</p>
  </noscript>
</div>

<script>
document.addEventListener('DOMContentLoaded', function () {
  'use strict';
  var root = document.querySelector('[data-account]');
  if (!root) return;
  var loading = root.querySelector('[data-account-loading]');
  var signin  = root.querySelector('[data-account-signin]');
  var panel   = root.querySelector('[data-account-panel]');
  var msg     = root.querySelector('[data-account-msg]');
  var panelMsg = root.querySelector('[data-account-panel-msg]');

  function show(el) { if (el) el.hidden = false; }
  function hide(el) { if (el) el.hidden = true; }
  function setMsg(box, text, ok) {
    if (!box) return;
    box.textContent = text || '';
    box.className = 'account__msg' + (ok === false ? ' is-error' : ok === true ? ' is-ok' : '');
  }

  var GT = window.GT;
  if (!GT || !GT.isConfigured) {
    hide(loading); show(signin);
    root.querySelectorAll('button, input').forEach(function (x) { x.disabled = true; });
    setMsg(msg, 'Accounts aren’t set up on this site yet — check back soon.', false);
    return;
  }

  function displayName(u) {
    var m = u.user_metadata || {};
    return m.user_name || m.preferred_username || m.name ||
      (u.email ? u.email.split('@')[0] : 'you');
  }

  // Fill in the "view public profile" link with the user's username once known,
  // so it points at /u/?u=<username> rather than the bare /u/ prompt page.
  var viewLink = panel.querySelector('[data-profile-view]');
  var profileLinkedFor = null;
  function linkPublicProfile() {
    if (!viewLink || !GT.user || profileLinkedFor === GT.user.id) return;
    profileLinkedFor = GT.user.id;
    GT.supabase.from('profiles').select('username').eq('id', GT.user.id).single()
      .then(function (res) {
        if (res && res.data && res.data.username) {
          viewLink.setAttribute('href', '/u/?u=' + encodeURIComponent(res.data.username));
        }
      });
  }

  function render() {
    hide(loading);
    if (GT.user) {
      hide(signin); show(panel);
      var n = panel.querySelector('[data-account-name]');
      var e = panel.querySelector('[data-account-email]');
      if (n) n.textContent = displayName(GT.user);
      if (e) e.textContent = GT.user.email ? '(' + GT.user.email + ')' : '';
      linkPublicProfile();
    } else {
      hide(panel); show(signin);
      profileLinkedFor = null;
    }
  }
  GT.onChange(render);
  if (GT.ready && GT.ready.then) GT.ready.then(render);

  root.querySelectorAll('[data-oauth]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      setMsg(msg, 'Redirecting…', true);
      GT.signInWithOAuth(btn.getAttribute('data-oauth')).then(function (res) {
        if (res && res.error) setMsg(msg, res.error.message, false);
      });
    });
  });

  var pw = root.querySelector('[data-form="password"]');
  if (pw) pw.addEventListener('submit', function (ev) {
    ev.preventDefault();
    var email = pw.querySelector('[name=email]').value.trim();
    var password = pw.querySelector('[name=password]').value;
    var signup = ev.submitter && ev.submitter.getAttribute('data-mode') === 'signup';
    setMsg(msg, 'Working…', true);
    (signup ? GT.signUpWithPassword(email, password)
            : GT.signInWithPassword(email, password)).then(function (res) {
      if (res.error) { setMsg(msg, res.error.message, false); return; }
      if (signup && res.data && res.data.user && !res.data.session) {
        setMsg(msg, 'Almost there — check your email to confirm your account, then you’re in.', true);
      } else if (signup) {
        setMsg(msg, 'Account created — you’re signed in.', true);
      } else {
        setMsg(msg, 'Signed in.', true);
      }
    });
  });

  var mg = root.querySelector('[data-form="magic"]');
  if (mg) mg.addEventListener('submit', function (ev) {
    ev.preventDefault();
    var email = mg.querySelector('[name=email]').value.trim();
    setMsg(msg, 'Sending link…', true);
    GT.signInWithMagicLink(email).then(function (res) {
      setMsg(msg, res.error ? res.error.message : 'Check your email for a sign-in link.',
        res.error ? false : true);
    });
  });

  var so = panel.querySelector('[data-signout]');
  if (so) so.addEventListener('click', function () {
    GT.signOut().then(function () { setMsg(panelMsg, 'Signed out.', true); });
  });

  var del = panel.querySelector('[data-delete]');
  if (del) del.addEventListener('click', function () {
    if (!window.confirm('Permanently delete your account and all synced progress? This cannot be undone.')) return;
    setMsg(panelMsg, 'Deleting…', true);
    GT.deleteAccount().then(function (res) {
      if (res && res.error) { setMsg(panelMsg, res.error.message, false); return; }
      GT.signOut().then(function () { setMsg(panelMsg, 'Your account was deleted.', true); });
    });
  });
});
</script>
