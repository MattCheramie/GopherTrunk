-- GopherTrunk — Supabase schema (Phase 3: forum moderation + rate limiting)
--
-- Apply AFTER schema-forum.sql, in the SQL Editor. Idempotent. Everything here
-- is additive and degrades gracefully: the client feature-detects each piece, so
-- the forum keeps working even before this file is applied.

create extension if not exists pgcrypto;

-- ---------------------------------------------------------------------------
-- Admins. Grant with:  insert into public.app_admins (user_id) values ('<uuid>');
-- (find the uuid under Authentication -> Users, or `select id from profiles`).
-- ---------------------------------------------------------------------------
create table if not exists public.app_admins (
  user_id    uuid primary key references public.profiles(id) on delete cascade,
  created_at timestamptz not null default now()
);
alter table public.app_admins enable row level security;

-- A signed-in user may read only their own admin row (so the client can tell
-- whether to show moderation controls). No client writes — grant via SQL.
drop policy if exists "see own admin row" on public.app_admins;
create policy "see own admin row" on public.app_admins
  for select to authenticated using (user_id = auth.uid());

create or replace function public.is_admin()
returns boolean language sql stable security definer set search_path = public as $$
  select exists (select 1 from public.app_admins where user_id = auth.uid());
$$;
grant execute on function public.is_admin() to authenticated, anon;

-- Admin override: admins may update/delete ANY thread or post. RLS combines
-- policies with OR, so these sit alongside the owner policies from schema-forum.
drop policy if exists "admins manage threads" on public.forum_threads;
create policy "admins manage threads" on public.forum_threads
  for all to authenticated using (public.is_admin()) with check (public.is_admin());

drop policy if exists "admins manage posts" on public.forum_posts;
create policy "admins manage posts" on public.forum_posts
  for all to authenticated using (public.is_admin()) with check (public.is_admin());

-- ---------------------------------------------------------------------------
-- Reports. Anyone signed in can file one; only admins can read them (review in
-- the dashboard, or build an admin view later).
-- ---------------------------------------------------------------------------
create table if not exists public.forum_reports (
  id          uuid primary key default gen_random_uuid(),
  post_id     uuid references public.forum_posts(id) on delete cascade,
  reporter_id uuid not null references public.profiles(id) on delete cascade,
  reason      text check (char_length(reason) <= 500),
  created_at  timestamptz not null default now()
);
alter table public.forum_reports enable row level security;

drop policy if exists "report as self" on public.forum_reports;
create policy "report as self" on public.forum_reports
  for insert to authenticated with check (reporter_id = auth.uid());

drop policy if exists "admins read reports" on public.forum_reports;
create policy "admins read reports" on public.forum_reports
  for select to authenticated using (public.is_admin());

-- ---------------------------------------------------------------------------
-- Rate limiting (server-side; the anon key is public so this cannot be bypassed
-- from the client). Tune the thresholds to taste.
-- ---------------------------------------------------------------------------
create or replace function public.enforce_post_rate_limit()
returns trigger language plpgsql security definer set search_path = public as $$
declare recent int;
begin
  select count(*) into recent from public.forum_posts
    where author_id = new.author_id and created_at > now() - interval '1 minute';
  if recent >= 8 then
    raise exception 'Rate limit: too many posts in a short time — slow down.';
  end if;
  return new;
end;
$$;
drop trigger if exists forum_posts_rate_limit on public.forum_posts;
create trigger forum_posts_rate_limit before insert on public.forum_posts
  for each row execute function public.enforce_post_rate_limit();

create or replace function public.enforce_thread_rate_limit()
returns trigger language plpgsql security definer set search_path = public as $$
declare recent int;
begin
  select count(*) into recent from public.forum_threads
    where author_id = new.author_id and created_at > now() - interval '5 minutes';
  if recent >= 3 then
    raise exception 'Rate limit: too many new threads — slow down.';
  end if;
  return new;
end;
$$;
drop trigger if exists forum_threads_rate_limit on public.forum_threads;
create trigger forum_threads_rate_limit before insert on public.forum_threads
  for each row execute function public.enforce_thread_rate_limit();

-- ---------------------------------------------------------------------------
-- Realtime. Add the forum tables to Supabase's realtime publication so the
-- client's live-update subscriptions receive INSERT/UPDATE events. Idempotent
-- (skips tables already in the publication). Reads still go through RLS, so
-- only world-readable rows are broadcast.
-- ---------------------------------------------------------------------------
do $$
begin
  if not exists (select 1 from pg_publication_tables
                 where pubname = 'supabase_realtime' and schemaname = 'public' and tablename = 'forum_posts') then
    alter publication supabase_realtime add table public.forum_posts;
  end if;
  if not exists (select 1 from pg_publication_tables
                 where pubname = 'supabase_realtime' and schemaname = 'public' and tablename = 'forum_threads') then
    alter publication supabase_realtime add table public.forum_threads;
  end if;
end $$;
