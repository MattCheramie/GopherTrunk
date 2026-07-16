-- GopherTrunk — Supabase schema (Phase 2: forum)
--
-- Apply AFTER schema.sql, in the SQL Editor (or `supabase db push`). Idempotent.
-- Same security model as Phase 1: RLS on every table; the browser holds only the
-- public key. Forum is world-readable, posting is authenticated-only, and a user
-- can only edit/delete their own rows.
--
-- author_id references public.profiles(id) (not auth.users) so PostgREST can embed
-- the author on reads (author:profiles(...)). profiles.id === auth.users.id, so
-- `author_id = auth.uid()` in the policies still means "the signed-in user".

create extension if not exists pgcrypto;   -- gen_random_uuid()

-- ---------------------------------------------------------------------------
-- Tables
-- ---------------------------------------------------------------------------
create table if not exists public.forum_categories (
  id          uuid primary key default gen_random_uuid(),
  slug        text unique not null,
  title       text not null,
  description text,
  sort_order  int  not null default 0,
  created_at  timestamptz not null default now()
);

create table if not exists public.forum_threads (
  id          uuid primary key default gen_random_uuid(),
  category_id uuid not null references public.forum_categories(id) on delete cascade,
  author_id   uuid not null references public.profiles(id)         on delete cascade,
  title       text not null check (char_length(title) between 3 and 200),
  is_locked   boolean not null default false,
  created_at  timestamptz not null default now(),
  updated_at  timestamptz not null default now()
);
create index if not exists forum_threads_category_idx on public.forum_threads (category_id, updated_at desc);

create table if not exists public.forum_posts (
  id         uuid primary key default gen_random_uuid(),
  thread_id  uuid not null references public.forum_threads(id) on delete cascade,
  author_id  uuid not null references public.profiles(id)      on delete cascade,
  body       text not null check (char_length(body) between 1 and 10000),
  is_deleted boolean not null default false,
  created_at timestamptz not null default now(),
  edited_at  timestamptz
);
create index if not exists forum_posts_thread_idx on public.forum_posts (thread_id, created_at);

-- ---------------------------------------------------------------------------
-- Row Level Security
-- ---------------------------------------------------------------------------
alter table public.forum_categories enable row level security;
alter table public.forum_threads    enable row level security;
alter table public.forum_posts      enable row level security;

-- Categories: world-readable; writes only via the dashboard / service role
-- (no anon or authenticated write policy exists, so RLS denies them).
drop policy if exists "categories readable" on public.forum_categories;
create policy "categories readable" on public.forum_categories
  for select using (true);

-- Threads: world-readable; a signed-in user creates threads as themselves and
-- edits/deletes only their own.
drop policy if exists "threads readable" on public.forum_threads;
create policy "threads readable" on public.forum_threads
  for select using (true);

drop policy if exists "threads insert own" on public.forum_threads;
create policy "threads insert own" on public.forum_threads
  for insert to authenticated with check (author_id = auth.uid());

drop policy if exists "threads update own" on public.forum_threads;
create policy "threads update own" on public.forum_threads
  for update to authenticated using (author_id = auth.uid()) with check (author_id = auth.uid());

drop policy if exists "threads delete own" on public.forum_threads;
create policy "threads delete own" on public.forum_threads
  for delete to authenticated using (author_id = auth.uid());

-- Posts: world-readable; a signed-in user posts as themselves on an UNLOCKED
-- thread, and edits/deletes only their own (delete is soft, via is_deleted).
drop policy if exists "posts readable" on public.forum_posts;
create policy "posts readable" on public.forum_posts
  for select using (true);

drop policy if exists "posts insert own" on public.forum_posts;
create policy "posts insert own" on public.forum_posts
  for insert to authenticated with check (
    author_id = auth.uid()
    and not exists (
      select 1 from public.forum_threads t where t.id = thread_id and t.is_locked
    )
  );

drop policy if exists "posts update own" on public.forum_posts;
create policy "posts update own" on public.forum_posts
  for update to authenticated using (author_id = auth.uid()) with check (author_id = auth.uid());

drop policy if exists "posts delete own" on public.forum_posts;
create policy "posts delete own" on public.forum_posts
  for delete to authenticated using (author_id = auth.uid());

-- ---------------------------------------------------------------------------
-- Keep thread.updated_at fresh when a reply lands (drives "recent" ordering).
-- ---------------------------------------------------------------------------
create or replace function public.touch_thread_on_post()
returns trigger language plpgsql security definer set search_path = public as $$
begin
  update public.forum_threads set updated_at = now() where id = new.thread_id;
  return new;
end;
$$;

drop trigger if exists on_forum_post_insert on public.forum_posts;
create trigger on_forum_post_insert
  after insert on public.forum_posts
  for each row execute function public.touch_thread_on_post();

-- ---------------------------------------------------------------------------
-- Starter categories — edit freely.
-- ---------------------------------------------------------------------------
insert into public.forum_categories (slug, title, description, sort_order) values
  ('general',  'General',           'Anything GopherTrunk.',                  1),
  ('help',     'Help & Support',    'Setup, tuning, and decoding questions.', 2),
  ('systems',  'Systems & Signals', 'Share systems, talkgroups, and finds.',  3),
  ('dev',      'Development',        'Building, contributing, and the code.',  4)
on conflict (slug) do nothing;
