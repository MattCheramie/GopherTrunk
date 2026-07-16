-- GopherTrunk — Supabase schema (Phase 5: content moderation — bans, report
-- workflow, and an audit log)
--
-- Apply AFTER schema-forum.sql and schema-forum-moderation.sql (this file builds
-- on app_admins / is_admin() and the forum tables). SQL Editor -> paste -> Run.
-- Idempotent: re-running is safe.
--
-- Everything is ADDITIVE and FEATURE-DETECTED by the client, so the forum and the
-- /moderation/ dashboard keep working before this file is applied. Same security
-- model as every phase: the browser holds only the public anon key; all writes
-- are guarded by Row Level Security.
--
-- This file adds three things:
--   1. User bans / suspensions, enforced server-side so a banned user cannot post
--      even though they still hold a valid session and anon key.
--   2. A report workflow: reports can target a THREAD as well as a post, and each
--      report carries a status (open / reviewing / resolved / dismissed).
--   3. A moderation audit log recording every admin action.

create extension if not exists pgcrypto;   -- gen_random_uuid()

-- ---------------------------------------------------------------------------
-- 1. Bans / suspensions.
--
-- expires_at IS NULL  -> permanent ban.
-- expires_at > now()  -> temporary suspension (auto-lifts when it passes).
-- Grant/lift bans from the /moderation/ dashboard (admins), or directly via SQL:
--   insert into public.user_bans (user_id, reason) values ('<uuid>', 'spam');
--   delete from public.user_bans where user_id = '<uuid>';   -- unban
-- ---------------------------------------------------------------------------
create table if not exists public.user_bans (
  user_id    uuid primary key references public.profiles(id) on delete cascade,
  banned_by  uuid references public.profiles(id) on delete set null,
  reason     text check (reason is null or char_length(reason) <= 500),
  created_at timestamptz not null default now(),
  expires_at timestamptz            -- null = permanent; future ts = suspension
);
alter table public.user_bans enable row level security;

-- A signed-in user may read only their OWN ban row (so the forum can show
-- "you're suspended until…"); admins may read every row.
drop policy if exists "read own ban" on public.user_bans;
create policy "read own ban" on public.user_bans
  for select to authenticated using (user_id = auth.uid() or public.is_admin());

-- Only admins write bans. RLS combines permissive policies with OR, and no other
-- write policy exists, so non-admins are denied insert/update/delete.
drop policy if exists "admins manage bans" on public.user_bans;
create policy "admins manage bans" on public.user_bans
  for all to authenticated using (public.is_admin()) with check (public.is_admin());

-- is_banned() — true while the caller has an active (non-expired) ban. SECURITY
-- DEFINER so it can read user_bans past RLS; it only ever checks auth.uid().
create or replace function public.is_banned()
returns boolean language sql stable security definer set search_path = public as $$
  select exists (
    select 1 from public.user_bans
    where user_id = auth.uid()
      and (expires_at is null or expires_at > now())
  );
$$;
grant execute on function public.is_banned() to authenticated, anon;

-- Enforcement. The insert-own policies in schema-forum.sql are PERMISSIVE, which
-- OR together — so a new permissive policy could never *subtract* the right to
-- post. RESTRICTIVE policies AND with everything else, which is exactly what we
-- want: "you may post (existing rule) AND you are not banned". This keeps
-- schema-forum.sql untouched.
drop policy if exists "posts blocked while banned" on public.forum_posts;
create policy "posts blocked while banned" on public.forum_posts
  as restrictive for insert to authenticated with check (not public.is_banned());

-- Also block edits, so a banned user can't rewrite an existing post to route
-- around the insert block.
drop policy if exists "post edits blocked while banned" on public.forum_posts;
create policy "post edits blocked while banned" on public.forum_posts
  as restrictive for update to authenticated with check (not public.is_banned());

drop policy if exists "threads blocked while banned" on public.forum_threads;
create policy "threads blocked while banned" on public.forum_threads
  as restrictive for insert to authenticated with check (not public.is_banned());

-- ---------------------------------------------------------------------------
-- 2. Report workflow — extend forum_reports (from schema-forum-moderation.sql).
--
-- Adds thread reports (thread_id) and a status/resolution lifecycle. Additive:
-- existing post-only reports keep their post_id and default to status 'open'.
-- ---------------------------------------------------------------------------
alter table public.forum_reports add column if not exists thread_id   uuid references public.forum_threads(id) on delete cascade;
alter table public.forum_reports add column if not exists status      text not null default 'open';
alter table public.forum_reports add column if not exists resolved_by uuid references public.profiles(id) on delete set null;
alter table public.forum_reports add column if not exists resolved_at timestamptz;

do $$ begin
  if not exists (select 1 from pg_constraint where conname = 'forum_reports_status_chk') then
    alter table public.forum_reports add constraint forum_reports_status_chk
      check (status in ('open', 'reviewing', 'resolved', 'dismissed'));
  end if;
  -- Exactly one target: a report is about a post OR a thread, never both/neither.
  -- Added NOT VALID so it applies to new/updated rows without re-checking any
  -- legacy rows (all of which already have a post_id, so they'd pass anyway).
  if not exists (select 1 from pg_constraint where conname = 'forum_reports_target_chk') then
    alter table public.forum_reports add constraint forum_reports_target_chk
      check ((post_id is not null) <> (thread_id is not null)) not valid;
  end if;
end $$;

create index if not exists forum_reports_status_idx on public.forum_reports (status, created_at desc);

-- Admins can update a report (change status, stamp resolved_by/at). The
-- insert-as-self and admins-read policies from schema-forum-moderation.sql still
-- apply.
drop policy if exists "admins update reports" on public.forum_reports;
create policy "admins update reports" on public.forum_reports
  for update to authenticated using (public.is_admin()) with check (public.is_admin());

-- ---------------------------------------------------------------------------
-- 3. Moderation audit log — one row per admin action, written by the dashboard.
-- Admin-only readable; admins insert rows stamped as themselves.
-- ---------------------------------------------------------------------------
create table if not exists public.moderation_actions (
  id          uuid primary key default gen_random_uuid(),
  actor_id    uuid not null references public.profiles(id) on delete set null,
  action      text not null,   -- 'lock_thread','unlock_thread','remove_post','ban_user','suspend_user','unban_user','resolve_report','dismiss_report'
  target_type text not null,   -- 'thread','post','user','report'
  target_id   uuid,
  detail      text check (detail is null or char_length(detail) <= 1000),
  created_at  timestamptz not null default now()
);
alter table public.moderation_actions enable row level security;
create index if not exists moderation_actions_created_idx on public.moderation_actions (created_at desc);

drop policy if exists "admins read audit" on public.moderation_actions;
create policy "admins read audit" on public.moderation_actions
  for select to authenticated using (public.is_admin());

drop policy if exists "admins write audit" on public.moderation_actions;
create policy "admins write audit" on public.moderation_actions
  for insert to authenticated with check (public.is_admin() and actor_id = auth.uid());

-- ---------------------------------------------------------------------------
-- 4. Realtime (optional) — let the moderation queue live-update. Idempotent:
-- skips tables already in the publication. Reads still pass through RLS, so only
-- admins receive these rows.
-- ---------------------------------------------------------------------------
do $$
begin
  if not exists (select 1 from pg_publication_tables
                 where pubname = 'supabase_realtime' and schemaname = 'public' and tablename = 'forum_reports') then
    alter publication supabase_realtime add table public.forum_reports;
  end if;
  if not exists (select 1 from pg_publication_tables
                 where pubname = 'supabase_realtime' and schemaname = 'public' and tablename = 'user_bans') then
    alter publication supabase_realtime add table public.user_bans;
  end if;
end $$;
