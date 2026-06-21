/*
 * Site-wide reference auto-linker. Progressive enhancement: with this file
 * absent or blocked, pages render normally. When present, it links the first
 * mention (per page) of each curated reference term to its /reference/ article.
 *
 * Safety rules: only inside .prose; never inside a/code/pre/headings/svg or the
 * infobox; whole-word (alnum-boundary) matching; longest term first; one link
 * per term per page; never self-link the current article. Terms come from
 * /reference-terms.json (only entries with `autolink: true`).
 */
(function () {
  'use strict';

  var SKIP_TAGS = { A: 1, CODE: 1, PRE: 1, H1: 1, H2: 1, H3: 1, H4: 1, H5: 1, H6: 1, SCRIPT: 1, STYLE: 1, svg: 1, BUTTON: 1, SUMMARY: 1 };

  function escapeRe(s) { return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }

  function currentPath() {
    return location.pathname.replace(/\/$/, '');
  }

  function inSkippedAncestor(node) {
    for (var el = node.parentNode; el && el !== document.body; el = el.parentNode) {
      if (el.nodeType !== 1) continue;
      var tag = el.tagName;
      if (SKIP_TAGS[tag] || SKIP_TAGS[tag && tag.toLowerCase()]) return true;
      if (el.classList && (el.classList.contains('ref-infobox') || el.classList.contains('no-autolink'))) return true;
    }
    return false;
  }

  function run(terms) {
    if (!terms || !terms.length) return;
    var here = currentPath();

    // Build candidates: {token, url}. Use aka synonyms plus the title with any
    // "(…)" abbreviation stripped. Drop candidates pointing at the current page.
    var candidates = [];
    terms.forEach(function (t) {
      var url = (t.url || '').replace(/\/$/, '');
      if (!url || url === here) return;            // never self-link
      var tokens = (t.aka || []).slice();
      var base = (t.term || '').replace(/\s*\(.*?\)\s*/g, ' ').trim();
      if (base) tokens.push(base);
      tokens.forEach(function (tok) {
        tok = (tok || '').trim();
        if (tok.length < 3) return;                // skip ambiguous short tokens
        candidates.push({ token: tok, url: t.url });
      });
    });
    if (!candidates.length) return;

    // Longest token first so multi-word terms win over their substrings.
    candidates.sort(function (a, b) { return b.token.length - a.token.length; });

    var used = {};                                  // url -> true (one link per term/page)
    var parts = candidates.map(function (c) { return escapeRe(c.token); });
    var re;
    try {
      re = new RegExp('(?<![A-Za-z0-9])(' + parts.join('|') + ')(?![A-Za-z0-9])', 'i');
    } catch (e) {
      return;                                       // lookbehind unsupported — bail quietly
    }
    var byToken = {};
    candidates.forEach(function (c) { byToken[c.token.toLowerCase()] = c.url; });

    // Collect target text nodes first (don't mutate while walking).
    var containers = document.querySelectorAll('.prose');
    var nodes = [];
    containers.forEach(function (root) {
      var walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, null);
      var n;
      while ((n = walker.nextNode())) {
        if (n.nodeValue.trim().length >= 3 && !inSkippedAncestor(n)) nodes.push(n);
      }
    });

    nodes.forEach(function (node) {
      var text = node.nodeValue;
      var m = re.exec(text);
      if (!m) return;
      var frag = document.createDocumentFragment();
      var last = 0;
      // Iterate matches within this node with a global, sticky-ish scan.
      var g = new RegExp(re.source, 'gi');
      var match;
      while ((match = g.exec(text))) {
        var token = match[1];
        var url = byToken[token.toLowerCase()];
        if (!url || used[url]) continue;
        used[url] = true;
        if (match.index > last) frag.appendChild(document.createTextNode(text.slice(last, match.index)));
        var a = document.createElement('a');
        a.className = 'ref-autolink';
        a.href = url;
        a.textContent = token;
        a.title = 'Field Guide: ' + token;
        frag.appendChild(a);
        last = match.index + token.length;
      }
      if (last > 0) {
        if (last < text.length) frag.appendChild(document.createTextNode(text.slice(last)));
        node.parentNode.replaceChild(frag, node);
      }
    });
  }

  document.addEventListener('DOMContentLoaded', function () {
    if (!document.querySelector('.prose')) return;
    var url = (document.documentElement.getAttribute('data-baseurl') || '') + '/reference-terms.json';
    fetch('/reference-terms.json')
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (terms) { if (terms) run(terms); })
      .catch(function () {});
  });
})();
