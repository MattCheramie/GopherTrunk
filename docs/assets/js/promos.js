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
 *   data-promo-fallback="inline"                      if the modal is suppressed
 *                                                     (content/ad blocker), relocate
 *                                                     the card to the page midpoint
 *                                                     and expand it in-flow instead
 * Dismiss controls (anywhere inside): [data-promo-close], optionally with
 *   data-promo-close-method="close_button|maybe_later|backdrop|escape|action".
 * Multi-step promos: wrap each step in [data-promo-step="<name>"] (the first, or
 *   one without `hidden`, shows first); a control with
 *   [data-promo-advance="<name>"] swaps to that step without closing the promo.
 * Action links use data-cta-event="cta_*" — tracked by site.js as a CTA click.
 *
 * Triggers are OR'd: the first to fire shows the promo; the rest are disarmed.
 *
 * QA override: append ?gtpromo=show to force-show immediately (bypassing the
 * time/scroll triggers and the once-per-browser dedup), or ?gtpromo=inline to
 * force the blocked-modal inline fallback, so both presentations can be checked
 * deterministically.
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

  // --- QA override (?gtpromo=show|inline) -----------------------------------
  function queryPromo() {
    try {
      var m = /[?&]gtpromo=([^&#]+)/.exec(window.location.search || '');
      return m ? decodeURIComponent(m[1]) : null;
    } catch (e) { return null; }
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
    this.fallback = el.getAttribute('data-promo-fallback') || '';
    this.shown = false;
    this.inlined = false;
    this.closed = false;
    this.timer = null;
    this.prevFocus = null;
    this._onScroll = null;
    this._onKey = null;
    this._bound = false;
  }

  Promo.prototype.arm = function () {
    if (!this.id) return;
    var self = this;

    // QA override: ?gtpromo=show forces an immediate modal; ?gtpromo=inline
    // forces the inline fallback (only for promos that opt into it). Both bypass
    // the triggers and the once-per-browser dedup.
    var q = queryPromo();
    if (q === 'show' || (q === 'inline' && this.fallback === 'inline')) {
      window.setTimeout(function () {
        self.show(q, { debug: true, forceInline: q === 'inline' });
      }, 0);
      return;
    }

    if (hasSeen(this.id, this.frequency)) return;

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

  Promo.prototype.show = function (trigger, opts) {
    if (this.shown) return;
    this.shown = true;
    this.disarmTriggers();
    opts = opts || {};

    // Write the seen flag on show (not on interaction) so an ignored or
    // reloaded promo does not re-appear. The QA override skips this so repeated
    // debug loads keep working.
    if (!opts.debug) markSeen(this.id, this.frequency);

    var el = this.el;
    el.removeAttribute('hidden');
    el.setAttribute('aria-hidden', 'false');

    // QA: jump straight to the inline presentation.
    if (opts.forceInline) { this.convertToInline('debug'); return; }

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

      // If this promo opts into the inline fallback, defer the impression until
      // we know whether the modal actually rendered — a content/ad blocker can
      // reveal-then-hide it. scheduleBlockedCheck fires the right impression
      // (popup when visible, fallback_inline when suppressed).
      if (this.fallback === 'inline') { this.scheduleBlockedCheck(trigger); return; }
    }

    fire('promo_impression', { promo_id: this.id, variant: this.variant, trigger: trigger || '' });
  };

  // --- Blocked-modal detection + inline fallback ---------------------------
  // Best-effort: after revealing the modal, verify on the next frames that the
  // card is actually painted. Cosmetic filters that hide the popup by class/id
  // are defeated by reclassing + relocating the node; a filter on [data-promo]
  // itself cannot be evaded — acceptable.
  Promo.prototype.scheduleBlockedCheck = function (trigger) {
    var self = this;
    var run = function () {
      if (self.closed) return;
      if (self.isBlocked()) self.convertToInline(trigger);
      else fire('promo_impression', { promo_id: self.id, variant: self.variant, trigger: trigger || '' });
    };
    if (window.requestAnimationFrame) {
      window.requestAnimationFrame(function () { window.requestAnimationFrame(run); });
    } else {
      window.setTimeout(run, 50);
    }
  };

  Promo.prototype.isBlocked = function () {
    var card = this.el.querySelector('.gt-promo__card') || this.el;
    // No box at all (display:none anywhere up the chain) or zero-sized.
    if (!card.getClientRects || !card.getClientRects().length) return true;
    var r = card.getBoundingClientRect();
    if (r.height === 0 || r.width === 0) return true;
    // Explicit hide on the root or the card. Cosmetic blockers use
    // `display:none`/`visibility:hidden`; opacity is deliberately NOT checked —
    // the modal's entry animation legitimately starts at opacity:0, and
    // opacity-hiding leaves layout height intact so it isn't a reliable signal.
    var nodes = [this.el, card];
    for (var i = 0; i < nodes.length; i++) {
      var cs = window.getComputedStyle(nodes[i]);
      if (!cs) continue;
      if (cs.display === 'none' || cs.visibility === 'hidden') return true;
    }
    return false;
  };

  // Nearest clean top-level content-block boundary to 50% of the content height,
  // so the two page halves separate cleanly (no element cut mid-block).
  Promo.prototype.relocateToMidpoint = function () {
    var host = document.querySelector('.prose') || document.getElementById('content');
    if (!host) return false;
    var kids = [];
    for (var i = 0; i < host.children.length; i++) {
      if (host.children[i] !== this.el) kids.push(host.children[i]);
    }
    if (!kids.length) return false;
    var hostTop = host.getBoundingClientRect().top + (window.pageYOffset || 0);
    var target = hostTop + host.offsetHeight / 2;
    var bestChild = null, bestDist = Infinity;
    for (var j = 0; j < kids.length; j++) {
      var top = kids[j].getBoundingClientRect().top + (window.pageYOffset || 0);
      var d = Math.abs(top - target);
      if (d < bestDist) { bestDist = d; bestChild = kids[j]; }
    }
    if (bestChild) host.insertBefore(this.el, bestChild);
    else host.appendChild(this.el);
    return true;
  };

  Promo.prototype.convertToInline = function (trigger) {
    if (this.inlined) return;
    this.inlined = true;

    // Shed the modal-only behaviors.
    document.body.classList.remove('gt-promo-open');
    if (this._onKey) { document.removeEventListener('keydown', this._onKey); this._onKey = null; }
    this._bound = false;
    this.isModal = false;

    var el = this.el;
    el.classList.remove('gt-promo--popup');
    el.classList.add('gt-promo--inline');
    // No longer a modal dialog; present as a labelled region. Dropping the id
    // also dodges the common cosmetic filters keyed on it.
    el.removeAttribute('aria-modal');
    el.setAttribute('role', 'region');
    el.setAttribute('aria-label', 'Support GopherTrunk');
    el.removeAttribute('id');
    el.removeAttribute('hidden');
    el.setAttribute('aria-hidden', 'false');

    this.relocateToMidpoint();

    // Expand from the insertion point on the next frame.
    var open = function () { el.classList.add('is-open'); };
    if (window.requestAnimationFrame) window.requestAnimationFrame(open); else open();

    fire('promo_impression', {
      promo_id: this.id,
      variant: 'inline',
      trigger: trigger === 'debug' ? 'debug_inline' : 'fallback_inline'
    });
  };

  // --- Steps ---------------------------------------------------------------
  Promo.prototype.showStep = function (name, opts) {
    opts = opts || {};
    var steps = this.el.querySelectorAll('[data-promo-step]');
    if (!steps.length) return;
    var from = null, target = null;
    for (var i = 0; i < steps.length; i++) {
      var s = steps[i];
      if (!s.hasAttribute('hidden')) from = s.getAttribute('data-promo-step');
      if (s.getAttribute('data-promo-step') === name) { target = s; s.removeAttribute('hidden'); }
      else s.setAttribute('hidden', '');
    }
    if (!target) return;

    // Keep the dialog's accessible name pointed at the active step.
    var title = target.querySelector('.gt-promo__title');
    var body = target.querySelector('.gt-promo__body');
    if (title && title.id) this.el.setAttribute('aria-labelledby', title.id);
    if (body && body.id) this.el.setAttribute('aria-describedby', body.id);

    if (opts.focus) {
      var f = target.querySelector('a[href], button, [tabindex]');
      if (f && f.focus) { try { f.focus(); } catch (e) {} }
    }
    if (opts.track && from && from !== name) {
      fire('promo_step', { promo_id: this.id, from: from, to: name });
    }
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
      // Step advance (e.g. "I'm already using GopherTrunk"): swap steps in place,
      // do NOT close the promo.
      var adv = e.target.closest && e.target.closest('[data-promo-advance]');
      if (adv) {
        e.preventDefault();
        self.showStep(adv.getAttribute('data-promo-advance'), { focus: true, track: true });
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
