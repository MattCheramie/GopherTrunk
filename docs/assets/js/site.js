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
