/*
 * GopherTrunk forum. Client-side, rendered into the static Jekyll pages
 * /forum/ (thread list) and /forum/thread/?t=<id> (one thread). Talks to the
 * Supabase tables (docs/supabase/schema-forum.sql + schema-forum-moderation.sql)
 * via window.GT (auth.js).
 *
 * Two pages, one file — dispatched by which container is present:
 *   [data-forum-list]   -> categories + recent threads + new-thread form
 *   [data-forum-thread] -> one thread's posts + reply box
 *
 * Phase 3 adds: Supabase Realtime live updates, "load more" pagination, post
 * reporting, and admin moderation (lock threads, remove any post). Every Phase 3
 * piece is FEATURE-DETECTED — if the moderation SQL isn't applied yet, those
 * calls quietly no-op and the core forum still works.
 *
 * Safety: all user content (titles, bodies, names) is rendered as TEXT via
 * createTextNode / textContent — never innerHTML — so a malicious post body
 * cannot inject markup. RLS enforces who may write; this stops stored XSS on
 * read, which RLS does not cover.
 *
 * Progressive enhancement: with JS off, or Supabase unconfigured, the pages show
 * a static message and the rest of the site is unaffected.
 */
(function () {
  'use strict';

  var PAGE_THREADS = 20;
  var PAGE_POSTS = 30;

  function sb() { return (window.GT && window.GT.supabase) || null; }
  function signedIn() { return !!(window.GT && window.GT.user); }
  function uid() { return window.GT && window.GT.user ? window.GT.user.id : null; }

  var THREAD_COLS = 'id, title, is_locked, updated_at, category:forum_categories(slug,title), author:profiles(username,display_name), replies:forum_posts(count)';
  var POST_COLS = 'id, body, created_at, edited_at, is_deleted, author_id, author:profiles(username,display_name)';

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
  function fmtDate(iso) { try { return new Date(iso).toLocaleString(); } catch (e) { return ''; } }
  function authorName(a) { return (a && (a.display_name || a.username)) || 'someone'; }
  function setMsg(box, text, ok) {
    if (!box) return;
    box.textContent = text || '';
    box.className = 'forum-msg' + (ok === false ? ' is-error' : ok === true ? ' is-ok' : '');
  }
  function threadUrl(id) { return '/forum/thread/?t=' + encodeURIComponent(id); }

  function removeMore(box) { var b = box.querySelector(':scope > .forum-more'); if (b) b.remove(); }
  function addMore(box, label, onClick) {
    var b = h('button', { class: 'forum-more', type: 'button' }, [label]);
    b.addEventListener('click', function () { b.disabled = true; onClick(); });
    box.appendChild(b);
  }

  var state = {
    cats: [], activeCat: null, thread: null, isAdmin: false,
    threadsOffset: 0, postsOffset: 0, postsFullyLoaded: false, channel: null
  };

  // is_admin() RPC (from schema-forum-moderation.sql). Absent -> treat as false.
  function checkAdmin(done) {
    if (!signedIn()) { state.isAdmin = false; return done(); }
    sb().rpc('is_admin').then(function (res) {
      state.isAdmin = !!(res && !res.error && res.data === true);
      done();
    }, function () { state.isAdmin = false; done(); });
  }

  document.addEventListener('DOMContentLoaded', function () {
    if (document.querySelector('[data-forum-list]')) initList();
    else if (document.querySelector('[data-forum-thread]')) initThread();
  });
  window.addEventListener('beforeunload', function () {
    if (state.channel && sb()) { try { sb().removeChannel(state.channel); } catch (e) {} }
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
        loadThreads(threadBox, true);
        subscribeList(threadBox);
      });

    if (window.GT && window.GT.onChange) {
      var first = true;
      window.GT.onChange(function () {
        if (first) { first = false; return; }
        checkAdmin(function () { renderNewThread(newBox); });
      });
    }
  }

  function renderCategories(box) {
    box.textContent = '';
    box.appendChild(h('a', { class: 'forum-cat' + (state.activeCat ? '' : ' is-active'), href: '/forum/' }, ['All']));
    state.cats.forEach(function (c) {
      box.appendChild(h('a', {
        class: 'forum-cat' + (state.activeCat === c.slug ? ' is-active' : ''),
        href: '/forum/?c=' + encodeURIComponent(c.slug), title: c.description || ''
      }, [c.title]));
    });
  }

  function loadThreads(box, reset) {
    if (reset) { state.threadsOffset = 0; box.textContent = 'Loading…'; }
    var from = state.threadsOffset, to = from + PAGE_THREADS - 1;
    var q = sb().from('forum_threads').select(THREAD_COLS)
      .order('updated_at', { ascending: false }).range(from, to);
    var cat = state.cats.filter(function (c) { return c.slug === state.activeCat; })[0];
    if (cat) q = q.eq('category_id', cat.id);

    q.then(function (res) {
      if (reset) box.textContent = '';
      removeMore(box);
      if (!res || res.error) { if (reset) box.appendChild(h('p', { class: 'forum__empty', text: 'Couldn’t load threads.' })); return; }
      var threads = res.data || [];
      if (reset && !threads.length) { box.appendChild(h('p', { class: 'forum__empty', text: 'No threads here yet — start one.' })); return; }
      threads.forEach(function (t) { if (!document.getElementById('thr-' + t.id)) box.appendChild(threadRow(t)); });
      state.threadsOffset += threads.length;
      if (threads.length === PAGE_THREADS) addMore(box, 'Load more threads', function () { loadThreads(box, false); });
    });
  }

  function threadRow(t) {
    var posts = (t.replies && t.replies[0] && t.replies[0].count) || 0;
    var title = h('a', { class: 'forum-thread__title', href: threadUrl(t.id) }, [t.title]);
    var meta = h('p', { class: 'forum-thread__meta' }, [
      (t.category ? t.category.title : ''), ' · by ', authorName(t.author),
      ' · ', posts + (posts === 1 ? ' post' : ' posts'), ' · ', fmtDate(t.updated_at)
    ]);
    var kids = [title, meta];
    if (t.is_locked) kids.push(h('span', { class: 'forum-badge', text: 'locked' }));
    return h('article', { class: 'forum-thread', id: 'thr-' + t.id }, kids);
  }

  function subscribeList(box) {
    if (!sb().channel) return;
    var timer = null;
    state.channel = sb().channel('forum-threads')
      .on('postgres_changes', { event: '*', schema: 'public', table: 'forum_threads' }, function () {
        clearTimeout(timer);
        timer = setTimeout(function () { loadThreads(box, true); }, 3000);
      })
      .subscribe();
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
    sb().from('forum_threads').insert({ category_id: categoryId, author_id: uid(), title: title })
      .select('id').single().then(function (res) {
        if (res.error || !res.data) { submit.disabled = false; setMsg(msg, res.error ? res.error.message : 'Could not create the thread.', false); return; }
        var tid = res.data.id;
        sb().from('forum_posts').insert({ thread_id: tid, author_id: uid(), body: body }).then(function (r2) {
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
      .eq('id', tid).single().then(function (res) {
        if (!res || res.error || !res.data) { head.textContent = 'Thread not found.'; return; }
        state.thread = res.data;
        checkAdmin(function () {
          renderThreadHead(head);
          loadPosts(postsBox, true);
          renderReply(replyBox);
          subscribeThread(postsBox);
        });
      });

    if (window.GT && window.GT.onChange) {
      var first = true;
      window.GT.onChange(function () {
        if (first) { first = false; return; }
        if (state.thread) checkAdmin(function () { renderThreadHead(head); renderReply(replyBox); loadPosts(postsBox, true); });
      });
    }
  }

  function renderThreadHead(head) {
    var t = state.thread;
    try { document.title = t.title + ' · Forum · GopherTrunk'; } catch (e) {}
    head.textContent = '';
    head.appendChild(h('h1', { class: 'forum-thread__heading', text: t.title }));
    var meta = h('p', { class: 'forum-thread__meta' }, [
      (t.category ? t.category.title : ''), ' · started by ', authorName(t.author), ' · ', fmtDate(t.created_at)
    ]);
    if (t.is_locked) meta.appendChild(h('span', { class: 'forum-badge', text: 'locked' }));
    head.appendChild(meta);

    if (state.isAdmin) {
      var lock = h('button', { class: 'btn btn--secondary forum-mod', type: 'button' }, [t.is_locked ? 'Unlock thread' : 'Lock thread']);
      lock.addEventListener('click', function () {
        var next = !t.is_locked;
        lock.disabled = true;
        sb().from('forum_threads').update({ is_locked: next }).eq('id', t.id).then(function (r) {
          lock.disabled = false;
          if (r.error) return;
          t.is_locked = next;
          renderThreadHead(head);
          renderReply(document.querySelector('[data-thread-reply]'));
        });
      });
      head.appendChild(lock);
    }
  }

  function loadPosts(box, reset) {
    if (reset) { state.postsOffset = 0; state.postsFullyLoaded = false; box.textContent = ''; }
    var from = state.postsOffset, to = from + PAGE_POSTS - 1;
    sb().from('forum_posts').select(POST_COLS).eq('thread_id', state.thread.id)
      .order('created_at', { ascending: true }).range(from, to).then(function (res) {
        removeMore(box);
        if (!res || res.error) { if (reset) box.appendChild(h('p', { text: 'Couldn’t load posts.' })); return; }
        var posts = res.data || [];
        posts.forEach(function (p) { if (!document.getElementById('post-' + p.id)) box.appendChild(postCard(p)); });
        state.postsOffset += posts.length;
        if (posts.length === PAGE_POSTS) addMore(box, 'Load more posts', function () { loadPosts(box, false); });
        else state.postsFullyLoaded = true;
      });
  }

  function markDeleted(card) {
    var body = card.querySelector('.forum-post__body');
    if (body) { body.textContent = ''; body.appendChild(h('em', { text: '[deleted]' })); }
    var acts = card.querySelector('.forum-post__actions');
    if (acts) acts.remove();
  }

  function postCard(p) {
    var card = h('article', { class: 'forum-post', id: 'post-' + p.id });
    card.appendChild(h('p', { class: 'forum-post__meta' }, [
      p.is_deleted ? 'unknown' : authorName(p.author), ' · ', fmtDate(p.created_at), p.edited_at ? ' · edited' : ''
    ]));
    var bodyEl = h('div', { class: 'forum-post__body' });
    if (p.is_deleted) bodyEl.appendChild(h('em', { text: '[deleted]' }));
    else bodyEl.appendChild(multiline(p.body));
    card.appendChild(bodyEl);

    if (!p.is_deleted && signedIn()) {
      var mine = p.author_id === uid();
      var actions = h('div', { class: 'forum-post__actions' });
      if (mine || state.isAdmin) {
        var del = h('button', { class: 'forum-post__act', type: 'button' }, [(state.isAdmin && !mine) ? 'Remove' : 'Delete']);
        del.addEventListener('click', function () {
          if (!window.confirm('Delete this post?')) return;
          sb().from('forum_posts').update({ is_deleted: true }).eq('id', p.id).then(function (r) { if (!r.error) markDeleted(card); });
        });
        actions.appendChild(del);
      }
      if (!mine) {
        var rep = h('button', { class: 'forum-post__act', type: 'button' }, ['Report']);
        rep.addEventListener('click', function () { reportPost(p.id, rep); });
        actions.appendChild(rep);
      }
      if (actions.childNodes.length) card.appendChild(actions);
    }
    return card;
  }

  function reportPost(postId, btn) {
    var reason = window.prompt('Report this post — reason (optional):');
    if (reason === null) return;
    btn.disabled = true;
    sb().from('forum_reports').insert({ post_id: postId, reporter_id: uid(), reason: reason || null })
      .then(function (res) { btn.textContent = res.error ? 'Report failed' : 'Reported'; });
  }

  function subscribeThread(box) {
    if (!sb().channel) return;
    var tid = state.thread.id;
    state.channel = sb().channel('thread-' + tid)
      .on('postgres_changes', { event: 'INSERT', schema: 'public', table: 'forum_posts', filter: 'thread_id=eq.' + tid },
        function (payload) { onRealtimeInsert(box, payload.new); })
      .on('postgres_changes', { event: 'UPDATE', schema: 'public', table: 'forum_posts', filter: 'thread_id=eq.' + tid },
        function (payload) { onRealtimeUpdate(payload.new); })
      .subscribe();
  }

  function onRealtimeInsert(box, row) {
    if (!row || !state.postsFullyLoaded) return;           // will appear when they page to the end
    if (document.getElementById('post-' + row.id)) return; // dedup with our own append
    sb().from('forum_posts').select(POST_COLS).eq('id', row.id).single().then(function (res) {
      if (res.error || !res.data) return;
      if (document.getElementById('post-' + res.data.id)) return;
      box.appendChild(postCard(res.data));
    });
  }
  function onRealtimeUpdate(row) {
    if (!row) return;
    var card = document.getElementById('post-' + row.id);
    if (card && row.is_deleted) markDeleted(card);
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
      sb().from('forum_posts').insert({ thread_id: t.id, author_id: uid(), body: body.value })
        .select(POST_COLS).single().then(function (res) {
          submit.disabled = false;
          if (res.error) { setMsg(msg, res.error.message, false); return; }
          body.value = '';
          setMsg(msg, '', true);
          var postsBox = document.querySelector('[data-thread-posts]');
          if (postsBox && !document.getElementById('post-' + res.data.id)) postsBox.appendChild(postCard(res.data));
        });
    });
    box.appendChild(form);
  }
})();
