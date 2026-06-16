/*
 * Client-side site search. Fetches /search.json (built by Jekyll) once and
 * filters it in the browser — no server, no external service. Progressive:
 * the page works without this script (the form just submits ?q= to /search/).
 */
(function () {
  'use strict';

  var input = document.getElementById('gt-search-input');
  var results = document.getElementById('gt-search-results');
  var status = document.getElementById('gt-search-status');
  if (!input || !results) return;

  var index = null;
  var pending = null;

  function esc(s) {
    return String(s).replace(/[&<>"]/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c];
    });
  }

  function terms(q) {
    return q.toLowerCase().split(/\s+/).filter(Boolean);
  }

  // Highlight each term in an escaped snippet.
  function highlight(text, ts) {
    var out = esc(text);
    ts.forEach(function (t) {
      var re = new RegExp('(' + t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')', 'ig');
      out = out.replace(re, '<mark>$1</mark>');
    });
    return out;
  }

  // Build a ~160-char snippet centred on the first term match in the body.
  function snippet(item, ts) {
    var desc = item.desc || '';
    if (desc) return desc;
    var body = item.body || '';
    var low = body.toLowerCase();
    var at = -1;
    for (var i = 0; i < ts.length; i++) {
      var p = low.indexOf(ts[i]);
      if (p !== -1) { at = p; break; }
    }
    if (at === -1) return body.slice(0, 160);
    var start = Math.max(0, at - 60);
    return (start > 0 ? '…' : '') + body.slice(start, start + 160) + '…';
  }

  function score(item, ts) {
    var title = (item.title || '').toLowerCase();
    var kw = (item.kw || '').toLowerCase();
    var group = (item.group || '').toLowerCase();
    var desc = (item.desc || '').toLowerCase();
    var body = (item.body || '').toLowerCase();
    var total = 0;
    for (var i = 0; i < ts.length; i++) {
      var t = ts[i], hit = 0;
      if (title.indexOf(t) !== -1) hit += 10;
      if (kw.indexOf(t) !== -1) hit += 5;
      if (group.indexOf(t) !== -1) hit += 4;
      if (desc.indexOf(t) !== -1) hit += 2;
      if (body.indexOf(t) !== -1) hit += 1;
      if (hit === 0) return 0; // every term must match somewhere (AND)
      total += hit;
    }
    if (title.indexOf(ts.join(' ')) === 0) total += 8; // title prefix boost
    return total;
  }

  function render(q) {
    var ts = terms(q);
    results.innerHTML = '';
    if (!ts.length) {
      status.textContent = index ? 'Type to search ' + index.length + ' pages.' : '';
      return;
    }
    if (!index) { status.textContent = 'Loading the search index…'; return; }

    var hits = [];
    for (var i = 0; i < index.length; i++) {
      var s = score(index[i], ts);
      if (s > 0) hits.push({ s: s, item: index[i] });
    }
    hits.sort(function (a, b) {
      return b.s - a.s || a.item.title.localeCompare(b.item.title);
    });

    if (!hits.length) {
      status.textContent = 'No results for “' + q + '”. Try a broader term.';
      return;
    }
    status.textContent = hits.length + (hits.length === 1 ? ' result' : ' results') + ' for “' + q + '”';

    var frag = document.createDocumentFragment();
    hits.slice(0, 40).forEach(function (h) {
      var li = document.createElement('li');
      li.className = 'search-result';
      var groupHtml = h.item.group
        ? '<span class="search-result__group">' + esc(h.item.group) + '</span>'
        : '';
      li.innerHTML =
        '<a class="search-result__link" href="' + esc(h.item.url) + '">' +
          '<span class="search-result__title">' + highlight(h.item.title, ts) + '</span>' +
          groupHtml +
          '<span class="search-result__excerpt">' + highlight(snippet(h.item, ts), ts) + '</span>' +
        '</a>';
      frag.appendChild(li);
    });
    results.appendChild(frag);
  }

  function getQ() {
    var m = location.search.match(/[?&]q=([^&]*)/);
    return m ? decodeURIComponent(m[1].replace(/\+/g, ' ')) : '';
  }

  function onInput() {
    var q = input.value.trim();
    if (pending) clearTimeout(pending);
    pending = setTimeout(function () {
      render(q);
      var url = q ? '?q=' + encodeURIComponent(q) : location.pathname;
      try { history.replaceState(null, '', url); } catch (e) {}
    }, 120);
  }

  input.addEventListener('input', onInput);

  // Load the index, then run any ?q= already in the URL.
  fetch('/search.json', { credentials: 'same-origin' })
    .then(function (r) { return r.json(); })
    .then(function (data) {
      index = data;
      var q = getQ();
      if (q && !input.value) input.value = q;
      render(input.value.trim());
    })
    .catch(function () {
      status.textContent = 'Could not load the search index. Please reload the page.';
    });
})();
