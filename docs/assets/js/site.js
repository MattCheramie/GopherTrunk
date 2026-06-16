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

  // In-content CTAs: relocate the "try" / "support" call-outs to roughly
  // 1/3 and 2/3 down the content. They render at the end of the host as a
  // no-JS fallback; here we move them into the reading flow by pixel height.
  var ctaHost = document.querySelector('[data-cta-host]');
  if (ctaHost) {
    var tryCta = ctaHost.querySelector(':scope > [data-cta="try"]');
    var supportCta = ctaHost.querySelector(':scope > [data-cta="support"]');
    if (tryCta && supportCta) {
      var blocks = Array.prototype.filter.call(ctaHost.children, function (el) {
        return el !== tryCta && el !== supportCta;
      });
      if (blocks.length >= 2) {
        var hostTop = ctaHost.getBoundingClientRect().top;
        var hostHeight = ctaHost.offsetHeight;
        // First block whose top crosses the given fraction of the content.
        var blockAt = function (fraction) {
          var target = hostTop + hostHeight * fraction;
          for (var i = 0; i < blocks.length; i++) {
            if (blocks[i].getBoundingClientRect().top >= target) return blocks[i];
          }
          return null;
        };
        var tryRef = blockAt(1 / 3);
        var supportRef = blockAt(2 / 3);
        // Guarantee support lands after try even when both resolve to the
        // same (or no) reference block.
        if (supportRef === tryRef) supportRef = null;
        if (tryRef) ctaHost.insertBefore(tryCta, tryRef);
        if (supportRef) ctaHost.insertBefore(supportCta, supportRef);
        else ctaHost.appendChild(supportCta);
      }
    }
  }

  // Google Analytics: fire a gtag event whenever a CTA link is clicked, so
  // we can measure whether these call-to-actions convert.
  document.addEventListener('click', function (e) {
    var link = e.target.closest && e.target.closest('[data-cta-event]');
    if (!link || typeof window.gtag !== 'function') return;
    window.gtag('event', 'cta_click', {
      cta_id: link.getAttribute('data-cta-event'),
      page_path: location.pathname
    });
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
        e.stopPropagation();
      });
    });
  }
})();
