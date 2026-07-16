/*
 * GopherTrunk moderation dashboard. Client-rendered into /moderation/, talks to
 * Supabase via window.GT (auth.js). Requires the Phase 5 schema
 * (docs/supabase/schema-moderation.sql) plus Phase 3
 * (schema-forum-moderation.sql) for is_admin() and forum_reports.
 *
 * Gating is defence-in-depth: the page calls is_admin() to decide what to show,
 * but the REAL gate is RLS — a non-admin who loads this page simply gets
 * empty/denied results from forum_reports / user_bans / moderation_actions.
 *
 * Every action (dismiss/resolve a report, lock a thread, remove a post, ban or
 * suspend a user) also writes a moderation_actions audit row, best-effort.
 *
 * Safety: all user content (report reasons, post bodies, names) is rendered as
 * TEXT via createTextNode / textContent — never innerHTML.
 */
(function () {
  'use strict';

  function sb() { return (window.GT && window.GT.supabase) || null; }
  function uid() { return window.GT && window.GT.user ? window.GT.user.id : null; }

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
  function fmtDate(iso) { try { return new Date(iso).toLocaleString(); } catch (e) { return ''; } }
  function authorName(a) { return (a && (a.display_name || a.username)) || 'someone'; }
  function userLink(a) {
    if (a && a.username) return h('a', { href: '/u/?u=' + encodeURIComponent(a.username) }, [authorName(a)]);
    return document.createTextNode(authorName(a));
  }
  function threadUrl(id) { return '/forum/thread/?t=' + encodeURIComponent(id); }
  function truncate(s, n) { s = String(s == null ? '' : s); return s.length > n ? s.slice(0, n) + '…' : s; }

  // Best-effort audit log write. Never blocks the action.
  function audit(action, targetType, targetId, detail) {
    if (!sb()) return;
    sb().from('moderation_actions').insert({
      actor_id: uid(), action: action, target_type: targetType,
      target_id: targetId || null, detail: detail || null
    }).then(function () {}, function () {});
  }

  document.addEventListener('DOMContentLoaded', function () {
    var root = document.querySelector('[data-moderation]');
    if (!root) return;
    var loading = root.querySelector('[data-mod-loading]');

    function notice(text) {
      root.textContent = '';
      root.appendChild(h('p', { class: 'forum__empty', text: text }));
    }

    if (!sb()) { notice('The moderation tools aren’t set up on this site yet.'); return; }

    function boot() {
      if (!window.GT.user) { notice('Sign in as an administrator to view this page.'); return; }
      sb().rpc('is_admin').then(function (res) {
        if (!res || res.error || res.data !== true) {
          notice('You must be an administrator to view this page.');
          return;
        }
        renderDashboard();
      }, function () { notice('You must be an administrator to view this page.'); });
    }

    if (window.GT.ready && window.GT.ready.then) window.GT.ready.then(boot);
    else boot();

    function renderDashboard() {
      if (loading) loading.remove();
      root.textContent = '';
      root.appendChild(h('section', { class: 'mod-section', 'data-mod-reports': '' }, [
        h('h2', { class: 'mod-section__title', text: 'Open reports' }),
        h('p', { class: 'account__loading', text: 'Loading reports…' })
      ]));
      root.appendChild(h('section', { class: 'mod-section', 'data-mod-bans': '' }, [
        h('h2', { class: 'mod-section__title', text: 'Active bans' }),
        h('p', { class: 'account__loading', text: 'Loading bans…' })
      ]));
      root.appendChild(h('section', { class: 'mod-section', 'data-mod-audit': '' }, [
        h('h2', { class: 'mod-section__title', text: 'Recent actions' }),
        h('p', { class: 'account__loading', text: 'Loading log…' })
      ]));
      loadReports();
      loadBans();
      loadAudit();
    }

    /* ----------------------------- reports ----------------------------- */
    var REPORT_COLS =
      'id, reason, status, created_at, post_id, thread_id, ' +
      'reporter:profiles!forum_reports_reporter_id_fkey(username, display_name), ' +
      'post:forum_posts(id, body, is_deleted, thread_id, author:profiles(username, display_name)), ' +
      'thread:forum_threads(id, title, is_locked, author:profiles(username, display_name))';

    function loadReports() {
      var box = root.querySelector('[data-mod-reports]');
      // Feature-detect the Phase-5 `status` column: try the rich query first,
      // fall back to a base query (no status filter / columns) on a column error.
      sb().from('forum_reports').select(REPORT_COLS).eq('status', 'open')
        .order('created_at', { ascending: false }).limit(100)
        .then(function (res) {
          if (res.error && /column|does not exist|schema cache/i.test(res.error.message || '')) {
            return sb().from('forum_reports')
              .select('id, reason, created_at, post_id, post:forum_posts(id, body, is_deleted, thread_id)')
              .order('created_at', { ascending: false }).limit(100);
          }
          return res;
        })
        .then(function (res) { renderReports(box, res); });
    }

    function renderReports(box, res) {
      box.textContent = '';
      box.appendChild(h('h2', { class: 'mod-section__title', text: 'Open reports' }));
      if (!res || res.error) { box.appendChild(h('p', { class: 'forum__empty', text: 'Couldn’t load reports.' })); return; }
      var reports = res.data || [];
      if (!reports.length) { box.appendChild(h('p', { class: 'forum__empty', text: 'No open reports. All clear.' })); return; }
      reports.forEach(function (r) { box.appendChild(reportCard(r)); });
    }

    function reportCard(r) {
      var card = h('article', { class: 'mod-report', id: 'report-' + r.id });
      var isThread = !!r.thread_id;
      var target = isThread ? r.thread : r.post;

      // Target line.
      var targetKids;
      if (isThread && target) {
        targetKids = ['Thread: ', h('a', { href: threadUrl(target.id) }, [target.title || 'a thread']),
          target.is_locked ? h('span', { class: 'forum-badge', text: 'locked' }) : null];
      } else if (target) {
        targetKids = ['Post in ', h('a', { href: threadUrl(target.thread_id) }, ['thread']),
          ': ', h('span', { class: 'mod-report__excerpt', text: truncate(target.body, 140) }),
          target.is_deleted ? h('span', { class: 'forum-badge', text: 'removed' }) : null];
      } else {
        targetKids = [h('em', { text: 'target no longer exists' })];
      }
      card.appendChild(h('p', { class: 'mod-report__target' }, targetKids));

      // Meta: reporter + reason + when.
      var metaKids = ['Reported by ', userLink(r.reporter), ' · ', fmtDate(r.created_at)];
      card.appendChild(h('p', { class: 'mod-report__meta' }, metaKids));
      if (r.reason) card.appendChild(h('p', { class: 'mod-report__reason' }, ['“', r.reason, '”']));

      // Actions.
      var actions = h('div', { class: 'mod-report__actions' });
      var msg = h('span', { class: 'account__msg', role: 'status', 'aria-live': 'polite' });

      if (isThread && target) {
        actions.appendChild(actBtn(target.is_locked ? 'Unlock thread' : 'Lock thread', function (btn) {
          var next = !target.is_locked;
          sb().from('forum_threads').update({ is_locked: next }).eq('id', target.id).then(function (res2) {
            if (res2.error) { setBtnMsg(msg, res2.error.message, false); btn.disabled = false; return; }
            target.is_locked = next; btn.textContent = next ? 'Unlock thread' : 'Lock thread'; btn.disabled = false;
            audit(next ? 'lock_thread' : 'unlock_thread', 'thread', target.id, null);
            setBtnMsg(msg, next ? 'Locked.' : 'Unlocked.', true);
          });
        }));
      }
      if (!isThread && target && !target.is_deleted) {
        actions.appendChild(actBtn('Remove post', function (btn) {
          sb().from('forum_posts').update({ is_deleted: true }).eq('id', target.id).then(function (res2) {
            if (res2.error) { setBtnMsg(msg, res2.error.message, false); btn.disabled = false; return; }
            target.is_deleted = true; btn.remove();
            audit('remove_post', 'post', target.id, null);
            setBtnMsg(msg, 'Post removed.', true);
          });
        }));
      }

      // Ban / suspend the reported author (if we know who it is). The report's
      // embedded author carries username/display_name; the ban flow resolves the
      // profile id from the username at click time.
      var author = target && target.author;
      if (author) {
        actions.appendChild(actBtn('Ban author', function (btn) {
          banFromReport(author, null, msg, btn);
        }));
        actions.appendChild(actBtn('Suspend author', function (btn) {
          var days = promptDays();
          if (!days) { btn.disabled = false; return; }   // cancelled/invalid -> don't fall through to a permanent ban
          banFromReport(author, days, msg, btn);
        }));
      }

      // Resolve / dismiss (status workflow; no-op-safe if column absent).
      actions.appendChild(actBtn('Resolve', function (btn) {
        setReportStatus(r.id, 'resolved', card, msg, btn);
      }));
      actions.appendChild(actBtn('Dismiss', function (btn) {
        setReportStatus(r.id, 'dismissed', card, msg, btn);
      }));

      card.appendChild(actions);
      card.appendChild(msg);
      return card;
    }

    function actBtn(label, onClick) {
      var b = h('button', { class: 'btn btn--secondary mod-btn', type: 'button' }, [label]);
      b.addEventListener('click', function () { b.disabled = true; onClick(b); });
      return b;
    }
    function setBtnMsg(msg, text, ok) {
      msg.textContent = text || '';
      msg.className = 'account__msg' + (ok === false ? ' is-error' : ok === true ? ' is-ok' : '');
    }
    function promptDays() {
      var v = window.prompt('Suspend for how many days?', '7');
      if (v === null) return null;
      var n = parseInt(v, 10);
      return (isNaN(n) || n <= 0) ? null : n;
    }

    // Resolve the target author's profile id from their username, then upsert a
    // ban. We look up the id because the embedded author only carries username /
    // display_name (author_id isn't in the report's foreign embed here).
    function banFromReport(author, days, msg, btn) {
      if (!author.username) { setBtnMsg(msg, 'Can’t identify that user.', false); btn.disabled = false; return; }
      var reason = window.prompt('Reason (optional):') || null;
      sb().from('profiles').select('id').eq('username', author.username).single().then(function (res) {
        if (res.error || !res.data) { setBtnMsg(msg, 'Couldn’t find that user.', false); btn.disabled = false; return; }
        var row = { user_id: res.data.id, banned_by: uid(), reason: reason };
        if (days) row.expires_at = new Date(Date.now() + days * 86400000).toISOString();
        sb().from('user_bans').upsert(row).then(function (res2) {
          btn.disabled = false;
          if (res2.error) { setBtnMsg(msg, res2.error.message, false); return; }
          audit(days ? 'suspend_user' : 'ban_user', 'user', res.data.id, days ? (days + 'd: ' + (reason || '')) : reason);
          setBtnMsg(msg, days ? ('Suspended for ' + days + ' days.') : 'User banned.', true);
          loadBans();
        });
      });
    }

    function setReportStatus(reportId, status, card, msg, btn) {
      sb().from('forum_reports')
        .update({ status: status, resolved_by: uid(), resolved_at: new Date().toISOString() })
        .eq('id', reportId)
        .then(function (res) {
          if (res.error) { setBtnMsg(msg, res.error.message, false); btn.disabled = false; return; }
          audit(status === 'resolved' ? 'resolve_report' : 'dismiss_report', 'report', reportId, null);
          card.classList.add('is-resolved');
          setBtnMsg(msg, status === 'resolved' ? 'Resolved.' : 'Dismissed.', true);
          setTimeout(function () { card.remove(); maybeEmptyReports(); }, 600);
          loadAudit();
        });
    }
    function maybeEmptyReports() {
      var box = root.querySelector('[data-mod-reports]');
      if (box && !box.querySelector('.mod-report')) {
        box.appendChild(h('p', { class: 'forum__empty', text: 'No open reports. All clear.' }));
      }
    }

    /* ------------------------------- bans ------------------------------ */
    function loadBans() {
      var box = root.querySelector('[data-mod-bans]');
      if (!box) return;
      sb().from('user_bans')
        .select('user_id, reason, created_at, expires_at, user:profiles!user_bans_user_id_fkey(username, display_name)')
        .order('created_at', { ascending: false }).limit(100)
        .then(function (res) {
          box.textContent = '';
          box.appendChild(h('h2', { class: 'mod-section__title', text: 'Active bans' }));
          if (!res || res.error) { box.appendChild(h('p', { class: 'forum__empty', text: 'Couldn’t load bans.' })); return; }
          var now = Date.now();
          var bans = (res.data || []).filter(function (b) { return !b.expires_at || new Date(b.expires_at).getTime() > now; });
          if (!bans.length) { box.appendChild(h('p', { class: 'forum__empty', text: 'No active bans.' })); return; }
          bans.forEach(function (b) { box.appendChild(banRow(b)); });
        });
    }

    function banRow(b) {
      var row = h('div', { class: 'mod-ban' });
      var kind = b.expires_at ? ('suspended until ' + fmtDate(b.expires_at)) : 'permanent ban';
      row.appendChild(h('span', { class: 'mod-ban__who' }, [userLink(b.user)]));
      row.appendChild(h('span', { class: 'mod-ban__meta', text: ' · ' + kind + (b.reason ? ' · ' + b.reason : '') }));
      var msg = h('span', { class: 'account__msg', role: 'status' });
      var unban = actBtn('Unban', function (btn) {
        sb().from('user_bans').delete().eq('user_id', b.user_id).then(function (res) {
          if (res.error) { setBtnMsg(msg, res.error.message, false); btn.disabled = false; return; }
          audit('unban_user', 'user', b.user_id, null);
          row.remove();
        });
      });
      row.appendChild(unban);
      row.appendChild(msg);
      return row;
    }

    /* ------------------------------ audit ------------------------------ */
    function loadAudit() {
      var box = root.querySelector('[data-mod-audit]');
      if (!box) return;
      sb().from('moderation_actions')
        .select('id, action, target_type, detail, created_at, actor:profiles!moderation_actions_actor_id_fkey(username, display_name)')
        .order('created_at', { ascending: false }).limit(25)
        .then(function (res) {
          box.textContent = '';
          box.appendChild(h('h2', { class: 'mod-section__title', text: 'Recent actions' }));
          if (!res || res.error) { box.appendChild(h('p', { class: 'forum__empty', text: 'No log yet.' })); return; }
          var rows = res.data || [];
          if (!rows.length) { box.appendChild(h('p', { class: 'forum__empty', text: 'No actions logged yet.' })); return; }
          var list = h('ul', { class: 'mod-audit' });
          rows.forEach(function (a) {
            list.appendChild(h('li', {}, [
              h('span', { class: 'mod-audit__action', text: a.action.replace(/_/g, ' ') }),
              ' · ', a.target_type, a.detail ? ' (' + a.detail + ')' : '',
              ' · by ', userLink(a.actor),
              h('span', { class: 'mod-audit__date', text: ' · ' + fmtDate(a.created_at) })
            ]));
          });
          box.appendChild(list);
        });
    }
  });
})();
