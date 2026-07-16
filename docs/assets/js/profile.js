/*
 * GopherTrunk public profile. Client-rendered into /u/ (Jekyll can't
 * pre-generate a page per user, same constraint as /forum/thread/), routed by
 * ?u=<username> or ?id=<uuid>. Talks to the Supabase `profiles` table (and,
 * for recent activity, `forum_posts`) via window.GT (auth.js).
 *
 * Safety: every piece of user content (names, bio, location, call sign) is
 * rendered as TEXT via createTextNode / textContent — never innerHTML — so a
 * malicious profile field cannot inject markup. The one attribute we set from
 * data is an <img src>, pointed only at the stored avatar_url (a Supabase
 * Storage public URL) with a same-origin default fallback.
 *
 * Progressive enhancement: with Supabase unconfigured, or before the Phase 4
 * profile columns exist, the page degrades gracefully — it feature-detects the
 * rich columns and falls back to the base set.
 */
(function () {
  'use strict';

  var DEFAULT_AVATAR = '/assets/gophertrunk-logo.png';
  var BASE_COLS = 'id, username, display_name, avatar_url, created_at';
  var RICH_COLS = BASE_COLS + ', bio, website, callsign, country, region, city';

  function sb() { return (window.GT && window.GT.supabase) || null; }

  // Minimal DOM builder — children are strings (text) or nodes, never HTML.
  function h(tag, attrs, kids) {
    var e = document.createElement(tag);
    if (attrs) Object.keys(attrs).forEach(function (k) {
      if (k === 'class') e.className = attrs[k];
      else if (k === 'text') e.textContent = attrs[k];
      else e.setAttribute(k, attrs[k]);
    });
    (kids || []).forEach(function (c) {
      if (c == null) return;
      e.appendChild(typeof c === 'string' ? document.createTextNode(c) : c);
    });
    return e;
  }
  function multiline(text) {
    var frag = document.createDocumentFragment();
    String(text == null ? '' : text).split('\n').forEach(function (line, i) {
      if (i) frag.appendChild(document.createElement('br'));
      frag.appendChild(document.createTextNode(line));
    });
    return frag;
  }
  function fmtDate(iso) { try { return new Date(iso).toLocaleString(); } catch (e) { return ''; } }
  function memberSince(iso) {
    try { return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'long' }); }
    catch (e) { return ''; }
  }
  function threadUrl(id) { return '/forum/thread/?t=' + encodeURIComponent(id); }
  function isHttpUrl(s) { return /^https?:\/\/[^ ]{3,200}$/i.test(String(s || '')); }

  // ISO-3166 alpha-2 -> display name, for the location line. Small subset with a
  // graceful fallback to the raw code if we don't have a name for it.
  var COUNTRY_NAMES = {
    AU:'Australia',AT:'Austria',BE:'Belgium',BR:'Brazil',CA:'Canada',CH:'Switzerland',
    CL:'Chile',CN:'China',CZ:'Czechia',DE:'Germany',DK:'Denmark',ES:'Spain',FI:'Finland',
    FR:'France',GB:'United Kingdom',GR:'Greece',HK:'Hong Kong',HU:'Hungary',IE:'Ireland',
    IN:'India',IT:'Italy',JP:'Japan',KR:'South Korea',MX:'Mexico',NL:'Netherlands',
    NO:'Norway',NZ:'New Zealand',PH:'Philippines',PL:'Poland',PT:'Portugal',RO:'Romania',
    RU:'Russia',SE:'Sweden',SG:'Singapore',TR:'Türkiye',UA:'Ukraine',US:'United States',
    ZA:'South Africa'
  };
  function countryName(code) { return COUNTRY_NAMES[code] || code; }

  document.addEventListener('DOMContentLoaded', function () {
    var root = document.querySelector('[data-profile]');
    if (!root) return;
    var loading = root.querySelector('[data-profile-loading]');

    function fail(text) {
      root.textContent = '';
      root.appendChild(h('p', { class: 'forum__empty', text: text }));
    }

    if (!sb()) { fail('Profiles aren’t set up on this site yet.'); return; }

    var params = new URLSearchParams(location.search);
    var uname = params.get('u');
    var uid = params.get('id');
    if (!uname && !uid) {
      root.textContent = '';
      root.appendChild(h('p', { class: 'forum__empty' }, [
        'No profile selected. Open a profile from a forum author’s name, or ',
        h('a', { href: '/account/' }, ['sign in']), ' to edit your own.'
      ]));
      return;
    }

    fetchProfile(RICH_COLS);

    function fetchProfile(cols) {
      var q = sb().from('profiles').select(cols);
      q = uname ? q.eq('username', uname) : q.eq('id', uid);
      q.single().then(function (res) {
        if (res.error && cols === RICH_COLS &&
            /column|does not exist|schema cache/i.test(res.error.message || '')) {
          return fetchProfile(BASE_COLS);   // Phase 4 columns not applied yet
        }
        if (!res || res.error || !res.data) { fail('No such user.'); return; }
        render(res.data);
      });
    }

    function render(p) {
      var name = p.display_name || p.username || 'Operator';
      try { document.title = name + ' · GopherTrunk'; } catch (e) {}
      if (loading) loading.remove();
      root.textContent = '';

      // Header: avatar + name + @username + call-sign badge.
      var img = h('img', { class: 'profile__avatar', alt: '', width: '96', height: '96', loading: 'lazy' });
      img.src = p.avatar_url || DEFAULT_AVATAR;
      img.addEventListener('error', function () { img.src = DEFAULT_AVATAR; });

      var idLine = [];
      if (p.username) idLine.push(h('span', { class: 'profile__username', text: '@' + p.username }));
      if (p.callsign) idLine.push(h('span', { class: 'forum-badge', text: p.callsign }));

      var headText = h('div', { class: 'profile__headtext' }, [
        h('h1', { class: 'profile__name', text: name }),
        idLine.length ? h('p', { class: 'profile__id' }, idLine) : null
      ]);
      root.appendChild(h('header', { class: 'profile__head' }, [img, headText]));

      // Meta: location + member since.
      var loc = [p.city, p.region, p.country ? countryName(p.country) : null]
        .filter(function (x) { return x && String(x).trim(); }).join(', ');
      var metaKids = [];
      if (loc) metaKids.push(h('span', { class: 'profile__loc', text: loc }));
      if (p.created_at) {
        if (metaKids.length) metaKids.push(' · ');
        metaKids.push('Member since ' + memberSince(p.created_at));
      }
      if (metaKids.length) root.appendChild(h('p', { class: 'profile__meta' }, metaKids));

      // Bio.
      if (p.bio) root.appendChild(h('div', { class: 'profile__bio' }, [multiline(p.bio)]));

      // Website.
      if (p.website && isHttpUrl(p.website)) {
        root.appendChild(h('p', { class: 'profile__website' }, [
          h('a', { href: p.website, rel: 'nofollow noopener', target: '_blank' }, [p.website])
        ]));
      }

      loadActivity(root, p.id);
    }

    // Recent forum activity — last 10 non-deleted posts, linking to their threads.
    function loadActivity(container, authorId) {
      if (!authorId) return;
      var section = h('section', { class: 'profile__activity' }, [
        h('h2', { class: 'profile__activity-title', text: 'Recent forum activity' }),
        h('p', { class: 'account__loading', text: 'Loading…' })
      ]);
      container.appendChild(section);

      sb().from('forum_posts')
        .select('id, created_at, thread:forum_threads(id, title)')
        .eq('author_id', authorId).eq('is_deleted', false)
        .order('created_at', { ascending: false }).limit(10)
        .then(function (res) {
          section.textContent = '';
          section.appendChild(h('h2', { class: 'profile__activity-title', text: 'Recent forum activity' }));
          if (!res || res.error) { section.appendChild(h('p', { class: 'forum__empty', text: 'Couldn’t load activity.' })); return; }
          var posts = (res.data || []).filter(function (p) { return p.thread; });
          if (!posts.length) { section.appendChild(h('p', { class: 'forum__empty', text: 'No forum posts yet.' })); return; }
          var list = h('ul', { class: 'profile__activity-list' });
          posts.forEach(function (p) {
            list.appendChild(h('li', {}, [
              h('a', { href: threadUrl(p.thread.id) }, [p.thread.title || 'a thread']),
              h('span', { class: 'profile__activity-date', text: ' · ' + fmtDate(p.created_at) })
            ]));
          });
          section.appendChild(list);
        });
    }
  });
})();
