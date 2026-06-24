/*
 * GopherTrunk learning-path enhancements. Progressive: every page is fully
 * usable with this file absent or blocked. Adds, when JS is available:
 *   - a "mark complete" control on lessons, persisted in localStorage
 *   - a progress bar + per-card checkmarks on the hub
 *   - frequency <-> wavelength and dB/dBm calculators
 *   - self-check quizzes
 */
(function () {
  'use strict';

  var STORE = 'gt-learn-progress';        // legacy single-path store
  var LEGACY_PATH = 'rf-sdr';             // where legacy progress belongs

  function storeKey(path) { return STORE + ':' + path; }

  // One-time migration: the original single-path store held bare RF slugs.
  // Move it under the rf-sdr key so existing visitors keep their checkmarks.
  (function migrateLegacy() {
    try {
      var legacy = localStorage.getItem(STORE);
      if (legacy && !localStorage.getItem(storeKey(LEGACY_PATH))) {
        localStorage.setItem(storeKey(LEGACY_PATH), legacy);
      }
      if (legacy) localStorage.removeItem(STORE);
    } catch (e) {}
  })();

  function load(path) {
    try {
      var raw = localStorage.getItem(storeKey(path));
      var arr = raw ? JSON.parse(raw) : [];
      return Array.isArray(arr) ? arr : [];
    } catch (e) { return []; }
  }
  function save(path, list) {
    try { localStorage.setItem(storeKey(path), JSON.stringify(list)); } catch (e) {}
  }
  function has(path, slug) { return load(path).indexOf(slug) !== -1; }
  function setDone(path, slug, done) {
    var list = load(path);
    var i = list.indexOf(slug);
    if (done && i === -1) list.push(slug);
    if (!done && i !== -1) list.splice(i, 1);
    save(path, list);
  }
  // Parse /learn/<path>/<slug>/ into { path, slug }; null off the learning path.
  function currentLesson() {
    var m = location.pathname.match(/\/learn\/([^/]+)\/([^/]+)\/?$/);
    return m ? { path: m[1], slug: m[2] } : null;
  }

  /* ---- Lesson page: mark complete ---- */
  function initLessonComplete() {
    var cur = currentLesson();
    if (!cur || cur.slug === 'glossary') return;
    var nav = document.querySelector('.path-nav');
    var article = document.querySelector('.lesson');
    if (!article) return;

    var wrap = document.createElement('div');
    wrap.className = 'lesson-complete';
    var btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'btn btn--secondary lesson-complete__btn';
    wrap.appendChild(btn);

    function render() {
      var done = has(cur.path, cur.slug);
      btn.setAttribute('aria-pressed', done ? 'true' : 'false');
      btn.classList.toggle('is-done', done);
      btn.textContent = done ? '✓ Completed — click to undo' : 'Mark this lesson complete';
    }
    btn.addEventListener('click', function () {
      setDone(cur.path, cur.slug, !has(cur.path, cur.slug));
      render();
    });
    render();

    if (nav && nav.parentNode) nav.parentNode.insertBefore(wrap, nav);
    else article.appendChild(wrap);
  }

  /* ---- Hub page: progress bar + card checks ---- */
  function initHubProgress() {
    var box = document.querySelector('[data-progress]');
    if (!box) return;
    var path = box.getAttribute('data-path');
    if (!path) return;
    var total = parseInt(box.getAttribute('data-total'), 10) || 0;
    var bar = box.querySelector('[data-progress-bar]');
    var count = box.querySelector('[data-progress-count]');
    var reset = box.querySelector('[data-progress-reset]');

    function paint() {
      var done = load(path);
      var cards = document.querySelectorAll('[data-lesson]');
      var n = 0;
      cards.forEach(function (card) {
        // data-lesson is "<pathId>/<slug>"; only this hub's cards count.
        var parts = card.getAttribute('data-lesson').split('/');
        if (parts[0] !== path) return;
        var slug = parts[1];
        var isDone = done.indexOf(slug) !== -1;
        card.classList.toggle('is-done', isDone);
        if (isDone && slug !== 'glossary') n++;
      });
      var pct = total ? Math.round((n / total) * 100) : 0;
      if (bar) bar.style.width = pct + '%';
      if (count) count.textContent = n;
    }
    box.hidden = false;
    if (reset) reset.addEventListener('click', function () { save(path, []); paint(); });
    paint();
  }

  /* ---- Chooser page: cross-path progress dashboard ---- */
  var RING_C = 119.38;   // 2*pi*19, matches the per-path/journey ring dasharray
  var DONUT_C = 326.726; // 2*pi*52, matches the aggregate donut dasharray

  function setRing(el, pct, circumference) {
    if (el) el.setAttribute('stroke-dashoffset', (circumference * (1 - pct)).toFixed(2));
  }

  // Count completed lessons for a path and find its first incomplete one.
  // Only slugs in the ordered `lessons` list count, so the glossary (stored
  // but never listed) is ignored for both totals and the resume target.
  function pathState(p) {
    var done = load(p.id);
    var n = 0, first = -1, i;
    for (i = 0; i < p.lessons.length; i++) {
      if (done.indexOf(p.lessons[i].slug) !== -1) n++;
      else if (first === -1) first = i;
    }
    return { n: n, total: p.total, first: first };
  }

  function paintCard(p, s) {
    var card = document.querySelector('[data-path-card="' + p.id + '"]');
    if (!card) return;
    var pct = s.total ? s.n / s.total : 0;
    setRing(card.querySelector('[data-card-ring]'), pct, RING_C);

    var count = card.querySelector('[data-card-count]');
    if (count) count.textContent = s.n + ' of ' + s.total + ' lessons';

    var actions = card.querySelector('[data-card-actions]');
    if (actions) actions.hidden = false;

    card.classList.toggle('is-complete', s.total > 0 && s.n >= s.total);

    var cont = card.querySelector('[data-card-continue]');
    if (cont) {
      if (s.total > 0 && s.n >= s.total) {
        cont.textContent = '✓ Completed';
        cont.href = p.hubUrl;
        cont.removeAttribute('aria-label');
      } else if (s.n === 0) {
        cont.textContent = 'Start lesson 1 →';
        cont.href = p.lessons[0] ? p.lessons[0].url : p.hubUrl;
        cont.removeAttribute('aria-label');
      } else {
        var next = p.lessons[s.first];
        cont.textContent = 'Continue →';
        cont.href = next.url;
        cont.setAttribute('aria-label', 'Continue ' + p.label + ': ' + next.title);
      }
    }

    var reset = card.querySelector('[data-card-reset]');
    if (reset) reset.hidden = s.n === 0;
  }

  function paintCluster(p, s) {
    var row = document.querySelector('[data-cluster-path="' + p.id + '"]');
    if (!row) return;
    var pct = s.total ? Math.round((s.n / s.total) * 100) : 0;
    var bar = row.querySelector('[data-cluster-bar]');
    var val = row.querySelector('[data-cluster-val]');
    if (bar) bar.style.width = pct + '%';
    if (val) val.textContent = pct + '%';
  }

  function paintJourney(p, s) {
    var step = document.querySelector('[data-journey-path="' + p.id + '"]');
    if (!step) return;
    var pct = s.total ? s.n / s.total : 0;
    setRing(step.querySelector('[data-journey-ring]'), pct, RING_C);
    step.classList.toggle('is-started', s.n > 0);
    step.classList.toggle('is-done', s.total > 0 && s.n >= s.total);
    var pctEl = step.querySelector('[data-journey-pct]');
    if (pctEl) {
      pctEl.textContent = Math.round(pct * 100) + '%';
      pctEl.hidden = s.n === 0;
    }
  }

  function paintDashboard(done, total, started, completed) {
    var box = document.querySelector('[data-learn-dashboard]');
    if (!box) return;
    box.hidden = false;
    var pct = total ? done / total : 0;
    setRing(box.querySelector('[data-donut-fill]'), pct, DONUT_C);
    var pctEl = box.querySelector('[data-donut-pct]');
    var cap = box.querySelector('[data-donut-caption]');
    if (pctEl) pctEl.textContent = Math.round(pct * 100) + '%';
    if (cap) cap.textContent = done + ' of ' + total + ' lessons complete';
    var l = box.querySelector('[data-stat-lessons]');
    var st = box.querySelector('[data-stat-started]');
    var cp = box.querySelector('[data-stat-completed]');
    if (l) l.textContent = done;
    if (st) st.textContent = started;
    if (cp) cp.textContent = completed;
  }

  function initChooserProgress() {
    var dataEl = document.getElementById('learn-paths-data');
    if (!dataEl) return; // not the chooser page
    var data;
    try { data = JSON.parse(dataEl.textContent); } catch (e) { return; }
    if (!data || !data.paths) return;

    var grandDone = 0, grandTotal = 0, started = 0, completed = 0;
    data.paths.forEach(function (p) {
      var s = pathState(p);
      grandDone += s.n;
      grandTotal += s.total;
      if (s.n > 0) started++;
      if (s.total > 0 && s.n >= s.total) completed++;
      paintCard(p, s);
      paintCluster(p, s);
      paintJourney(p, s);
    });
    paintDashboard(grandDone, grandTotal, started, completed);

    // Delegate reset clicks once: re-running initChooserProgress repaints
    // everything from fresh storage, so a single bound listener is enough.
    var grid = document.querySelector('.learn-grid');
    if (grid && !grid.getAttribute('data-reset-bound')) {
      grid.setAttribute('data-reset-bound', '1');
      grid.addEventListener('click', function (ev) {
        var btn = ev.target.closest && ev.target.closest('[data-card-reset]');
        if (!btn) return;
        ev.preventDefault();
        var card = btn.closest('[data-path-card]');
        if (!card) return;
        save(card.getAttribute('data-path-card'), []);
        initChooserProgress();
      });
    }
  }

  /* ---- Calculators ---- */
  var C = 299792458; // speed of light, m/s

  function fmt(n) {
    if (!isFinite(n)) return '—';
    if (n !== 0 && (Math.abs(n) < 1e-3 || Math.abs(n) >= 1e6)) return n.toExponential(3);
    return (Math.round(n * 1000) / 1000).toString();
  }

  function initFreqCalc(el) {
    var freq = el.querySelector('[data-freq]');
    var unit = el.querySelector('[data-freq-unit]');
    var out = el.querySelector('[data-wavelength]');
    if (!freq || !out) return;
    function calc() {
      var mult = unit ? parseFloat(unit.value) : 1e6;
      var hz = parseFloat(freq.value) * mult;
      var lambda = C / hz; // metres
      var disp = lambda >= 1 ? fmt(lambda) + ' m' : fmt(lambda * 100) + ' cm';
      out.textContent = isFinite(lambda) && hz > 0 ? disp : '—';
    }
    freq.addEventListener('input', calc);
    if (unit) unit.addEventListener('change', calc);
    calc();
  }

  function initDbCalc(el) {
    var watts = el.querySelector('[data-watts]');
    var dbm = el.querySelector('[data-dbm]');
    if (!watts || !dbm) return;
    function fromWatts() {
      var w = parseFloat(watts.value);
      if (!(w > 0)) { dbm.value = ''; return; }
      dbm.value = fmt(10 * Math.log10(w * 1000));
    }
    function fromDbm() {
      var d = parseFloat(dbm.value);
      if (isNaN(d)) { watts.value = ''; return; }
      watts.value = fmt(Math.pow(10, d / 10) / 1000);
    }
    watts.addEventListener('input', fromWatts);
    dbm.addEventListener('input', fromDbm);
    fromWatts();
  }

  /* ---- Quizzes ---- */
  function initQuiz(el) {
    var options = el.querySelectorAll('[data-answer]');
    var feedback = el.querySelector('[data-quiz-feedback]');
    options.forEach(function (opt) {
      opt.addEventListener('click', function () {
        var correct = opt.getAttribute('data-answer') === 'correct';
        options.forEach(function (o) { o.classList.remove('is-correct', 'is-wrong'); });
        opt.classList.add(correct ? 'is-correct' : 'is-wrong');
        if (feedback) {
          feedback.hidden = false;
          feedback.textContent = correct
            ? '✓ ' + (el.getAttribute('data-correct-msg') || 'Correct!')
            : '✗ Not quite — try again.';
          feedback.className = 'quiz__feedback ' + (correct ? 'is-correct' : 'is-wrong');
        }
      });
    });
  }

  document.addEventListener('DOMContentLoaded', function () {
    initLessonComplete();
    initHubProgress();
    initChooserProgress();
    document.querySelectorAll('[data-calc="freq"]').forEach(initFreqCalc);
    document.querySelectorAll('[data-calc="db"]').forEach(initDbCalc);
    document.querySelectorAll('[data-quiz]').forEach(initQuiz);
  });
})();
