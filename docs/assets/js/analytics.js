/*
 * GopherTrunk shared metrics. Loaded (deferred) site-wide from default.html
 * BEFORE site.js and promos.js so both can use the same tracking surface.
 *
 * This is the single place that talks to Google Analytics (gtag) and the
 * localStorage CTA memory. The CTA click tracking used to live privately inside
 * site.js; it now lives here so promos (pop-ups / pop-ins / fly-outs) record
 * impressions, clicks, and dismissals through the exact same path — no parallel
 * analytics.
 *
 * Exposes window.GTMetrics:
 *   GTMetrics.event(name, params)   fire a gtag event (no-op if gtag absent);
 *                                   page_path is attached automatically.
 *   GTMetrics.cta(id)               remember the CTA family in localStorage and
 *                                   fire a 'cta_click' event. Called by site.js
 *                                   for every [data-cta-event] click.
 */
(function () {
  'use strict';

  var CTA_KEY = 'gt-cta';

  // Fire a gtag event, defensively. gtag may be missing (blocked/offline) — in
  // that case this is a silent no-op so callers never need to guard.
  function event(name, params) {
    if (!name) return;
    var payload = { page_path: location.pathname };
    if (params) {
      for (var k in params) {
        if (Object.prototype.hasOwnProperty.call(params, k)) payload[k] = params[k];
      }
    }
    try {
      if (typeof window.gtag === 'function') window.gtag('event', name, payload);
    } catch (e) {}
  }

  // CTA interaction memory (localStorage). We record which call-to-action
  // "families" a visitor has already acted on so we can adapt later pages:
  // someone who has downloaded sees the Support ask promoted, and vice-versa.
  function readCtaState() {
    try { return JSON.parse(localStorage.getItem(CTA_KEY)) || {}; }
    catch (e) { return {}; }
  }
  function rememberCta(family) {
    if (!family) return;
    try {
      var s = readCtaState();
      s[family] = true;
      localStorage.setItem(CTA_KEY, JSON.stringify(s));
    } catch (e) {}
  }
  function ctaFamily(id) {
    if (!id) return null;
    if (id === 'cta_try_download') return 'try';
    if (id.indexOf('cta_support_') === 0) return 'support';
    if (id.indexOf('cta_join_') === 0) return 'join';
    if (id.indexOf('cta_star_') === 0) return 'star';
    return null;
  }

  // Record a CTA click: remember its family (for adaptive ordering) and fire a
  // gtag event so we can measure whether these CTAs convert.
  function cta(id) {
    if (!id) return;
    rememberCta(ctaFamily(id));
    event('cta_click', { cta_id: id });
  }

  window.GTMetrics = {
    event: event,
    cta: cta,
    readCtaState: readCtaState
  };
})();
