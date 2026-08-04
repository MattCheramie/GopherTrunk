(function () {
  'use strict';

  var root = document.documentElement;
  var body = document.body;

  // Dark-mode toggle
  var darkBtn = document.querySelector('.dark-toggle');
  if (darkBtn) {
    darkBtn.addEventListener('click', function () {
      var nowDark = root.classList.toggle('dark');
      try { localStorage.setItem('gt-theme', nowDark ? 'dark' : 'light'); } catch (e) {}
      darkBtn.setAttribute('aria-pressed', nowDark ? 'true' : 'false');
    });
  }

  // Mobile hamburger
  var burger = document.querySelector('.site-nav__hamburger');
  if (burger) {
    burger.addEventListener('click', function () {
      var open = body.classList.toggle('nav-open');
      burger.setAttribute('aria-expanded', open ? 'true' : 'false');
    });
  }

  // Downloads page: highlight the card matching the visitor's OS
  var cards = document.querySelectorAll('.download-card[data-platform]');
  if (cards.length) {
    var uad = navigator.userAgentData;
    var hay = ((uad && uad.platform) || navigator.platform || '') + ' ' + (navigator.userAgent || '');
    hay = hay.toLowerCase();
    var detected = null;
    if (/win/.test(hay))                              detected = 'windows';
    else if (/mac|darwin|iphone|ipad/.test(hay))      detected = 'macos';
    else if (/linux|x11|android|cros/.test(hay))      detected = 'linux';
    if (detected) {
      cards.forEach(function (card) {
        if (card.dataset.platform !== detected) return;
        card.classList.add('download-card--match');
        var h3 = card.querySelector('h3');
        if (h3 && !h3.querySelector('.download-card__badge')) {
          var badge = document.createElement('span');
          badge.className = 'download-card__badge';
          badge.textContent = 'Your platform';
          h3.appendChild(badge);
        }
      });
    }
  }

  // CTA interaction memory now lives in the shared metrics module
  // (assets/js/analytics.js, window.GTMetrics), loaded before this file, so
  // promos and CTAs record through the same path. Read state through it, with a
  // safe fallback if GTMetrics somehow failed to load.
  var readCtaState = function () {
    if (window.GTMetrics && window.GTMetrics.readCtaState) return window.GTMetrics.readCtaState();
    return {};
  };

  // In-content CTAs: relocate the "try" / "support" call-outs into the
  // reading flow (~1/3 and ~2/3 down the content). They render at the end of
  // the host as a no-JS fallback. Order adapts to prior clicks: the CTA the
  // visitor has NOT yet acted on is promoted first and the other is demoted.
  var ctaHost = document.querySelector('[data-cta-host]');
  if (ctaHost) {
    var tryCta = ctaHost.querySelector(':scope > [data-cta="try"]');
    var supportCta = ctaHost.querySelector(':scope > [data-cta="support"]');
    if (tryCta && supportCta) {
      var state = readCtaState();
      var promoteSupport = state['try'] && !state['support'];
      var promoteTry = state['support'] && !state['try'];
      var firstCta = promoteSupport ? supportCta : tryCta;
      var secondCta = promoteSupport ? tryCta : supportCta;
      if (promoteSupport) tryCta.classList.add('page-cta--demoted');
      if (promoteTry) supportCta.classList.add('page-cta--demoted');

      var blocks = Array.prototype.filter.call(ctaHost.children, function (el) {
        return el !== tryCta && el !== supportCta && el.tagName !== 'SCRIPT';
      });
      // First block whose top crosses the given fraction of the content.
      var hostTop = ctaHost.getBoundingClientRect().top;
      var hostHeight = ctaHost.offsetHeight;
      var blockAt = function (fraction) {
        var target = hostTop + hostHeight * fraction;
        for (var i = 0; i < blocks.length; i++) {
          if (blocks[i].getBoundingClientRect().top >= target) return blocks[i];
        }
        return null;
      };
      if (blocks.length >= 2) {
        var firstRef = blockAt(1 / 3);
        var secondRef = blockAt(2 / 3);
        // Guarantee the second CTA lands after the first even when both
        // resolve to the same (or no) reference block.
        if (secondRef === firstRef) secondRef = null;
        if (firstRef) ctaHost.insertBefore(firstCta, firstRef);
        if (secondRef) ctaHost.insertBefore(secondCta, secondRef);
        else ctaHost.appendChild(secondCta);
      } else if (blocks.length === 1) {
        // Short page: first CTA right after the only block, second at the end.
        ctaHost.insertBefore(firstCta, blocks[0].nextSibling);
        ctaHost.appendChild(secondCta);
      } else {
        // No content blocks: keep both at the end, in the promoted order.
        ctaHost.appendChild(firstCta);
        ctaHost.appendChild(secondCta);
      }
    }
  }

  // CTA clicks: delegate to the shared metrics module, which remembers the
  // family in localStorage (for adaptive ordering) and fires the gtag event.
  document.addEventListener('click', function (e) {
    var link = e.target.closest && e.target.closest('[data-cta-event]');
    if (!link) return;
    if (window.GTMetrics && window.GTMetrics.cta) {
      window.GTMetrics.cta(link.getAttribute('data-cta-event'));
    }
  });

  // Mobile: tap a group label to expand its submenu (desktop uses :hover/:focus-within)
  var isCoarse = window.matchMedia('(max-width: 800px)').matches;
  if (isCoarse) {
    document.querySelectorAll('.nav-group__label').forEach(function (label) {
      label.addEventListener('click', function (e) {
        var group = label.parentElement;
        if (!group) return;
        var wasOpen = group.classList.contains('is-open');
        document.querySelectorAll('.nav-group.is-open').forEach(function (g) {
          if (g !== group) g.classList.remove('is-open');
        });
        group.classList.toggle('is-open', !wasOpen);
        label.setAttribute('aria-expanded', !wasOpen ? 'true' : 'false');
        // Drop focus when collapsing so no lingering :focus state can keep
        // the submenu visually open after the class is removed.
        if (wasOpen) label.blur();
        e.stopPropagation();
      });
    });
  }
})();
