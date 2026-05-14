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
