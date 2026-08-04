/*
 * GopherTrunk promo engine. Loaded (deferred) site-wide from default.html,
 * after analytics.js and site.js.
 *
 * A small, declarative system for pop-ups, pop-ins, and fly-outs. Adding a new
 * promo requires NO change to this file: drop an include into the page markup
 * whose root carries the data-promo-* attributes below, and the engine
 * discovers it, arms its triggers, shows it once, and reports every impression,
 * dismissal, and action through the shared metrics module (window.GTMetrics) —
 * the same tracking path the site's CTAs use.
 *
 * Markup contract (on the promo root element):
 *   class="gt-promo gt-promo--{popup|popin|flyout}"  presentation variant
 *   hidden                                            engine reveals it
 *   data-promo="<unique-id>"                          analytics + dedup key
 *   data-promo-variant="popup|popin|flyout"           behavior (backdrop/lock)
 *   data-promo-trigger-time="<ms>"                    show after N ms (optional)
 *   data-promo-trigger-scroll="<0..1>"                show at scroll fraction (optional)
 *   data-promo-frequency="once|session|always"        dedup scope (default once)
 * Dismiss controls (anywhere inside): [data-promo-close], optionally with
 *   data-promo-close-method="close_button|maybe_later|backdrop|escape|action".
 * Action links use data-cta-event="cta_*" — tracked by site.js as a CTA click.
 *
 * Triggers are OR'd: the first to fire shows the promo; the rest are disarmed.
 */
(function () {
  'use strict';

  var SEEN_KEY = 'gt-promos';

  function metrics() { return window.GTMetrics || null; }
  function fire(name, params) { var m = metrics(); if (m) m.event(name, params); }

  // --- Per-frequency "seen" memory -----------------------------------------
  function store(freq) {
    // 'session' -> sessionStorage; 'once' (default) -> localStorage; 'always'
    // -> no persistence (handled by callers skipping read/write).
    try { return freq === 'session' ? window.sessionStorage : window.localStorage; }
    catch (e) { return null; }
  }
  function readSeen(freq) {
    var s = store(freq);
    if (!s) return {};
    try { return JSON.parse(s.getItem(SEEN_KEY)) || {}; }
    catch (e) { return {}; }
  }
  function hasSeen(id, freq) {
    if (freq === 'always') return false;
    return !!readSeen(freq)[id];
  }
  function markSeen(id, freq) {
    if (freq === 'always') return;
    var s = store(freq);
    if (!s) return;
    try {
      var seen = readSeen(freq);
      seen[id] = true;
      s.setItem(SEEN_KEY, JSON.stringify(seen));
    } catch (e) {}
  }

  // --- Scroll fraction ------------------------------------------------------
  function scrollFraction() {
    var doc = document.documentElement;
    var max = (doc.scrollHeight || document.body.scrollHeight) - window.innerHeight;
    if (max <= 0) return 0; // page can't scroll — time trigger will cover it
    var y = window.pageYOffset || doc.scrollTop || 0;
    return Math.min(1, Math.max(0, y / max));
  }

  // --- One promo ------------------------------------------------------------
  function Promo(el) {
    this.el = el;
    this.id = el.getAttribute('data-promo');
    this.variant = el.getAttribute('data-promo-variant') || 'popup';
    this.frequency = el.getAttribute('data-promo-frequency') || 'once';
    this.timeMs = parseInt(el.getAttribute('data-promo-trigger-time'), 10);
    this.scrollAt = parseFloat(el.getAttribute('data-promo-trigger-scroll'));
    this.isModal = this.variant === 'popup';
    this.shown = false;
    this.closed = false;
    this.timer = null;
    this.prevFocus = null;
    this._onScroll = null;
    this._onKey = null;
    this._bound = false;
  }

  Promo.prototype.arm = function () {
    if (!this.id || hasSeen(this.id, this.frequency)) return;
    var self = this;

    if (!isNaN(this.timeMs) && this.timeMs >= 0) {
      this.timer = window.setTimeout(function () { self.show('time'); }, this.timeMs);
    }

    if (!isNaN(this.scrollAt)) {
      this._onScroll = function () {
        if (scrollFraction() >= self.scrollAt) self.show('scroll');
      };
      window.addEventListener('scroll', this._onScroll, { passive: true });
      // In case the page already loaded scrolled past the threshold.
      this._onScroll();
    }
  };

  Promo.prototype.disarmTriggers = function () {
    if (this.timer) { window.clearTimeout(this.timer); this.timer = null; }
    if (this._onScroll) { window.removeEventListener('scroll', this._onScroll); this._onScroll = null; }
  };

  Promo.prototype.show = function (trigger) {
    if (this.shown) return;
    this.shown = true;
    this.disarmTriggers();

    // Write the seen flag on show (not on interaction) so an ignored or
    // reloaded promo does not re-appear.
    markSeen(this.id, this.frequency);

    var el = this.el;
    el.removeAttribute('hidden');
    el.setAttribute('aria-hidden', 'false');

    if (this.isModal) {
      this.prevFocus = document.activeElement;
      document.body.classList.add('gt-promo-open');
      this.bindModalDismiss();
      // Move focus into the dialog for keyboard users.
      var focusTarget = el.querySelector('[autofocus], [data-promo-focus], .btn, a[href], button');
      if (focusTarget && focusTarget.focus) {
        try { focusTarget.focus(); } catch (e) {}
      } else if (el.focus) {
        try { el.focus(); } catch (e) {}
      }
    }

    fire('promo_impression', { promo_id: this.id, variant: this.variant, trigger: trigger || '' });
  };

  Promo.prototype.bindModalDismiss = function () {
    if (this._bound) return;
    this._bound = true;
    var self = this;
    this._onKey = function (e) {
      if (e.key === 'Escape' || e.keyCode === 27) self.close('escape');
    };
    document.addEventListener('keydown', this._onKey);
  };

  Promo.prototype.close = function (method) {
    if (this.closed || !this.shown) return;
    this.closed = true;

    var el = this.el;
    el.setAttribute('hidden', '');
    el.setAttribute('aria-hidden', 'true');

    if (this.isModal) {
      document.body.classList.remove('gt-promo-open');
      if (this._onKey) { document.removeEventListener('keydown', this._onKey); this._onKey = null; }
      // Restore focus to wherever it was before we opened.
      if (this.prevFocus && this.prevFocus.focus) {
        try { this.prevFocus.focus(); } catch (e) {}
      }
    }

    fire('promo_dismiss', { promo_id: this.id, method: method || 'unknown' });
  };

  // Delegated clicks within the promo: dismiss controls and action links.
  Promo.prototype.bindClicks = function () {
    var self = this;
    this.el.addEventListener('click', function (e) {
      var closer = e.target.closest && e.target.closest('[data-promo-close]');
      if (closer) {
        var method = closer.getAttribute('data-promo-close-method') || 'close_button';
        self.close(method);
        return;
      }
      // A primary action (e.g. the star link) also closes the promo. The click
      // itself is tracked by site.js via its data-cta-event attribute.
      var action = e.target.closest && e.target.closest('[data-cta-event]');
      if (action) self.close('action');
    });
  };

  // --- Engine ---------------------------------------------------------------
  var registered = {};

  function register(el) {
    if (!el || el.getAttribute('data-promo-bound') === '1') return;
    var promo = new Promo(el);
    if (!promo.id || registered[promo.id]) return;
    el.setAttribute('data-promo-bound', '1');
    registered[promo.id] = promo;
    promo.bindClicks();
    promo.arm();
  }

  function scan() {
    var nodes = document.querySelectorAll('.gt-promo[data-promo]');
    for (var i = 0; i < nodes.length; i++) register(nodes[i]);
  }

  window.GTPromos = { register: register, scan: scan };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', scan);
  } else {
    scan();
  }
})();
