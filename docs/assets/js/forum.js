/*
 * GopherTrunk forum. Client-side, rendered into the static Jekyll pages
 * /forum/ (thread list) and /forum/thread/?t=<id> (one thread). Talks to the
 * Supabase tables from docs/supabase/schema-forum.sql via window.GT (auth.js).
 *
 * Two pages, one file — dispatched by which container is present:
 *   [data-forum-list]   -> categories + recent threads + new-thread form
 *   [data-forum-thread] -> one thread's posts + reply box
 *
 * Safety: every piece of user content (titles, bodies, names) is rendered as
 * TEXT via createTextNode / textContent — never innerHTML — so a malicious post
 * body cannot inject markup. RLS enforces who may write; this stops stored XSS
 * on read, which RLS does not cover.
 *
 * Progressive enhancement: with JS off, or Supabase unconfigured, the pages show
 * a static "loading/​not set up" message and the rest of the site is unaffected.
 */
(function () {
  'use strict';

  function sb() { return (window.GT && window.GT.supabase) || null; }
  function signedIn() { return !!(window.GT && window.GT.user); }
  function uid() { return window.GT && window.GT.user ? window.GT.user.id : null; }

  // --- tiny DOM builder; children are strings (text) or nodes, never HTML ---
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
  function fmtDate(iso) {
    try { return new Date(iso).toLocaleString(); } catch (e) { return ''; }
  }
  function authorName(a) {
    if (!a) return 'someone';
    return a.display_name || a.username || 'someone';
  }
  function setMsg(box, text, ok) {
    if (!box) return;
    box.textContent = text || '';
    box.className = 'forum-msg' + (ok === false ? ' is-error' : ok === true ? ' is-ok' : '');
  }
  function threadUrl(id) { return '/forum/thread/?t=' + encodeURIComponent(id); }

  // Cached page state so auth changes can re-render only the gated bits.
  var state = { cats: [], activeCat: null, thread: null };

  document.addEventListener('DOMContentLoaded', function () {
    if (document.querySelector('[data-forum-list]')) initList();
    else if (document.querySelector('[data-forum-thread]')) initThread();
  });

  /* ============================ list page ============================ */
  function initList() {
    var root = document.querySelector('[data-forum-list]');
    var catBox = root.querySelector('[data-forum-categories]');
    var threadBox = root.querySelector('[data-forum-threads]');
    var newBox = root.querySelector('[data-forum-new]');
    if (!sb()) { threadBox.textContent = 'The forum isn’t set up on this site yet.'; return; }

    state.activeCat = new URLSearchParams(location.search).get('c');

    sb().from('forum_categories')
      .select('id, slug, title, description, sort_order')
      .order('sort_order', { ascending: true })
      .then(function (res) {
        state.cats = (res && res.data) || [];
        renderCategories(catBox);
        renderNewThread(newBox);
        loadThreads(threadBox);
      });

    if (window.GT && window.GT.onChange) {
      var first = true;
      window.GT.onChange(function () {
        if (first) { first = false; return; }  // initial fire handled above
        renderNewThread(newBox);
      });
    }
  }

  function renderCategories(box) {
    box.textContent = '';
    var all = h('a', { class: 'forum-cat' + (state.activeCat ? '' : ' is-active'), href: '/forum/' }, ['All']);
    box.appendChild(all);
    state.cats.forEach(function (c) {
      box.appendChild(h('a', {
        class: 'forum-cat' + (state.activeCat === c.slug ? ' is-active' : ''),
        href: '/forum/?c=' + encodeURIComponent(c.slug),
        title: c.description || ''
      }, [c.title]));
    });
  }

  function loadThreads(box) {
    box.textContent = 'Loading…';
    var q = sb().from('forum_threads')
      .select('id, title, is_locked, updated_at, category:forum_categories(slug,title), author:profiles(username,display_name), replies:forum_posts(count)')
      .order('updated_at', { ascending: false })
      .limit(30);
    var cat = state.cats.filter(function (c) { return c.slug === state.activeCat; })[0];
    if (cat) q = q.eq('category_id', cat.id);

    q.then(function (res) {
      box.textContent = '';
      if (!res || res.error) { box.appendChild(h('p', { class: 'forum__empty', text: 'Couldn’t load threads.' })); return; }
      var threads = res.data || [];
      if (!threads.length) { box.appendChild(h('p', { class: 'forum__empty', text: 'No threads here yet — start one.' })); return; }
      threads.forEach(function (t) { box.appendChild(threadRow(t)); });
    });
  }

  function threadRow(t) {
    var posts = (t.replies && t.replies[0] && t.replies[0].count) || 0;
    var meta = h('p', { class: 'forum-thread__meta' }, [
      (t.category ? t.category.title : ''), ' · by ', authorName(t.author),
      ' · ', posts + (posts === 1 ? ' post' : ' posts'),
      ' · ', fmtDate(t.updated_at)
    ]);
    var title = h('a', { class: 'forum-thread__title', href: threadUrl(t.id) }, [t.title]);
    var kids = [title, meta];
    if (t.is_locked) kids.push(h('span', { class: 'forum-badge', text: 'locked' }));
    return h('article', { class: 'forum-thread' }, kids);
  }

  function renderNewThread(box) {
    box.textContent = '';
    if (!signedIn()) {
      box.appendChild(h('p', { class: 'forum__signin' }, ['Want to post? ', h('a', { href: '/account/' }, ['Sign in']), '.']));
      return;
    }
    if (!state.cats.length) return;
    var details = h('details', { class: 'forum-new' }, [h('summary', { text: 'Start a new thread' })]);
    var form = h('form', { class: 'forum-form' });

    var sel = h('select', { name: 'category', 'aria-label': 'Category' });
    state.cats.forEach(function (c) {
      var o = h('option', { value: c.id }, [c.title]);
      if (c.slug === state.activeCat) o.selected = true;
      sel.appendChild(o);
    });
    var title = h('input', { name: 'title', type: 'text', maxlength: '200', placeholder: 'Thread title', required: 'required' });
    var body = h('textarea', { name: 'body', rows: '5', maxlength: '10000', placeholder: 'Write your first post…', required: 'required' });
    var msg = h('p', { class: 'forum-msg', role: 'status', 'aria-live': 'polite' });
    var submit = h('button', { class: 'btn', type: 'submit' }, ['Post thread']);

    form.appendChild(h('label', { class: 'forum-form__label' }, ['Category', sel]));
    form.appendChild(h('label', { class: 'forum-form__label' }, ['Title', title]));
    form.appendChild(h('label', { class: 'forum-form__label' }, ['First post', body]));
    form.appendChild(submit);
    form.appendChild(msg);
    form.addEventListener('submit', function (ev) {
      ev.preventDefault();
      createThread(sel.value, title.value.trim(), body.value, submit, msg);
    });
    details.appendChild(form);
    box.appendChild(details);
  }

  function createThread(categoryId, title, body, submit, msg) {
    if (title.length < 3) { setMsg(msg, 'Title needs at least 3 characters.', false); return; }
    if (!body.trim()) { setMsg(msg, 'The first post can’t be empty.', false); return; }
    submit.disabled = true;
    setMsg(msg, 'Posting…', true);
    sb().from('forum_threads')
      .insert({ category_id: categoryId, author_id: uid(), title: title })
      .select('id').single()
      .then(function (res) {
        if (res.error || !res.data) { submit.disabled = false; setMsg(msg, res.error ? res.error.message : 'Could not create the thread.', false); return; }
        var tid = res.data.id;
        sb().from('forum_posts')
          .insert({ thread_id: tid, author_id: uid(), body: body })
          .then(function (r2) {
            if (r2.error) { submit.disabled = false; setMsg(msg, r2.error.message, false); return; }
            location.href = threadUrl(tid);
          });
      });
  }

  /* ============================ thread page ============================ */
  function initThread() {
    var root = document.querySelector('[data-forum-thread]');
    var head = root.querySelector('[data-thread-head]');
    var postsBox = root.querySelector('[data-thread-posts]');
    var replyBox = root.querySelector('[data-thread-reply]');
    if (!sb()) { head.textContent = 'The forum isn’t set up on this site yet.'; return; }

    var tid = new URLSearchParams(location.search).get('t');
    if (!tid) { head.textContent = 'Thread not found.'; return; }

    sb().from('forum_threads')
      .select('id, title, is_locked, created_at, category:forum_categories(slug,title), author:profiles(username,display_name)')
      .eq('id', tid).single()
      .then(function (res) {
        if (!res || res.error || !res.data) { head.textContent = 'Thread not found.'; return; }
        state.thread = res.data;
        renderThreadHead(head);
        loadPosts(postsBox);
        renderReply(replyBox);
      });

    if (window.GT && window.GT.onChange) {
      var first = true;
      window.GT.onChange(function () {
        if (first) { first = false; return; }
        if (state.thread) { renderReply(replyBox); loadPosts(postsBox); }
      });
    }
  }

  function renderThreadHead(head) {
    var t = state.thread;
    try { document.title = t.title + ' · Forum · GopherTrunk'; } catch (e) {}
    head.textContent = '';
    head.appendChild(h('h1', { class: 'forum-thread__heading', text: t.title }));
    var meta = h('p', { class: 'forum-thread__meta' }, [
      (t.category ? t.category.title : ''), ' · started by ', authorName(t.author),
      ' · ', fmtDate(t.created_at)
    ]);
    if (t.is_locked) meta.appendChild(h('span', { class: 'forum-badge', text: 'locked' }));
    head.appendChild(meta);
  }

  function loadPosts(box) {
    sb().from('forum_posts')
      .select('id, body, created_at, edited_at, is_deleted, author_id, author:profiles(username,display_name)')
      .eq('thread_id', state.thread.id)
      .order('created_at', { ascending: true })
      .then(function (res) {
        box.textContent = '';
        if (!res || res.error) { box.appendChild(h('p', { text: 'Couldn’t load posts.' })); return; }
        (res.data || []).forEach(function (p) { box.appendChild(postCard(p)); });
      });
  }

  function postCard(p) {
    var card = h('article', { class: 'forum-post', id: 'post-' + p.id });
    card.appendChild(h('p', { class: 'forum-post__meta' }, [
      p.is_deleted ? 'unknown' : authorName(p.author),
      ' · ', fmtDate(p.created_at), p.edited_at ? ' · edited' : ''
    ]));
    var bodyEl = h('div', { class: 'forum-post__body' });
    if (p.is_deleted) bodyEl.appendChild(h('em', { text: '[deleted]' }));
    else bodyEl.appendChild(multiline(p.body));
    card.appendChild(bodyEl);

    if (!p.is_deleted && signedIn() && p.author_id === uid()) {
      var del = h('button', { class: 'forum-post__del', type: 'button' }, ['Delete']);
      del.addEventListener('click', function () {
        if (!window.confirm('Delete this post?')) return;
        sb().from('forum_posts').update({ is_deleted: true }).eq('id', p.id).then(function (r) {
          if (r.error) return;
          bodyEl.textContent = '';
          bodyEl.appendChild(h('em', { text: '[deleted]' }));
          del.remove();
        });
      });
      card.appendChild(del);
    }
    return card;
  }

  function renderReply(box) {
    var t = state.thread;
    box.textContent = '';
    if (t.is_locked) { box.appendChild(h('p', { class: 'forum-msg', text: 'This thread is locked.' })); return; }
    if (!signedIn()) {
      box.appendChild(h('p', { class: 'forum__signin' }, ['Want to reply? ', h('a', { href: '/account/' }, ['Sign in']), '.']));
      return;
    }
    var form = h('form', { class: 'forum-form' });
    var body = h('textarea', { name: 'body', rows: '4', maxlength: '10000', placeholder: 'Write a reply…', required: 'required' });
    var msg = h('p', { class: 'forum-msg', role: 'status', 'aria-live': 'polite' });
    var submit = h('button', { class: 'btn', type: 'submit' }, ['Reply']);
    form.appendChild(body);
    form.appendChild(submit);
    form.appendChild(msg);
    form.addEventListener('submit', function (ev) {
      ev.preventDefault();
      if (!body.value.trim()) { setMsg(msg, 'Your reply is empty.', false); return; }
      submit.disabled = true;
      setMsg(msg, 'Posting…', true);
      sb().from('forum_posts')
        .insert({ thread_id: t.id, author_id: uid(), body: body.value })
        .select('id, body, created_at, edited_at, is_deleted, author_id, author:profiles(username,display_name)')
        .single()
        .then(function (res) {
          submit.disabled = false;
          if (res.error) { setMsg(msg, res.error.message, false); return; }
          body.value = '';
          setMsg(msg, '', true);
          var postsBox = document.querySelector('[data-thread-posts]');
          if (postsBox) postsBox.appendChild(postCard(res.data));
        });
    });
    box.appendChild(form);
  }
})();
