-- GopherTrunk — Supabase schema (Phase 1: accounts + learning-path progress)
--
-- Apply in your Supabase project: SQL Editor -> paste -> Run (or `supabase db
-- push`). This file is committed as documentation / version history; the site
-- itself does NOT apply it. Re-running is safe — every statement is idempotent.
--
-- Security model: the site ships the PUBLIC anon key, so every table has Row
-- Level Security ENABLED with explicit policies. RLS — not key secrecy — is what
-- protects user data. Never expose the service_role key to the browser.
--
-- Forum tables (forum_categories / forum_threads / forum_posts) arrive in
-- Phase 2 as docs/supabase/schema-forum.sql.

-- ---------------------------------------------------------------------------
-- profiles — one row per auth user, auto-created on signup.
-- ---------------------------------------------------------------------------
create table if not exists public.profiles (
  id           uuid primary key references auth.users (id) on delete cascade,
  username     text unique,
  display_name text,
  avatar_url   text,
  created_at   timestamptz not null default now()
);

alter table public.profiles enable row level security;

drop policy if exists "profiles are public" on public.profiles;
create policy "profiles are public"
  on public.profiles for select using (true);

drop policy if exists "insert own profile" on public.profiles;
create policy "insert own profile"
  on public.profiles for insert with check (auth.uid() = id);

drop policy if exists "update own profile" on public.profiles;
create policy "update own profile"
  on public.profiles for update using (auth.uid() = id) with check (auth.uid() = id);

-- Auto-create a profile when a new auth user appears, seeding fields from OAuth
-- metadata (GitHub/Google) where present.
create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer set search_path = public
as $$
begin
  insert into public.profiles (id, username, display_name, avatar_url)
  values (
    new.id,
    coalesce(new.raw_user_meta_data->>'user_name',
             new.raw_user_meta_data->>'preferred_username'),
    coalesce(new.raw_user_meta_data->>'name',
             new.raw_user_meta_data->>'full_name'),
    new.raw_user_meta_data->>'avatar_url'
  )
  on conflict (id) do nothing;
  return new;
end;
$$;

drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created
  after insert on auth.users
  for each row execute function public.handle_new_user();

-- ---------------------------------------------------------------------------
-- learn_progress — one row per completed lesson, private to the user.
-- Matches the client store: key is (user_id, path_id, slug).
-- ---------------------------------------------------------------------------
create table if not exists public.learn_progress (
  user_id      uuid not null references auth.users (id) on delete cascade,
  path_id      text not null,
  slug         text not null,
  completed_at timestamptz not null default now(),
  primary key (user_id, path_id, slug)
);

alter table public.learn_progress enable row level security;

drop policy if exists "own progress" on public.learn_progress;
create policy "own progress"
  on public.learn_progress for all
  using (auth.uid() = user_id)
  with check (auth.uid() = user_id);

-- ---------------------------------------------------------------------------
-- delete_own_account() — a signed-in user deletes their own account. The
-- on-delete cascade removes their profile + progress (and, in Phase 2, their
-- forum rows). SECURITY DEFINER so it may touch auth.users; it only ever
-- deletes the caller (auth.uid()), never anyone else.
-- ---------------------------------------------------------------------------
create or replace function public.delete_own_account()
returns void
language plpgsql
security definer set search_path = public, auth
as $$
begin
  if auth.uid() is null then
    raise exception 'not authenticated';
  end if;
  delete from auth.users where id = auth.uid();
end;
$$;

revoke all on function public.delete_own_account() from public, anon;
grant execute on function public.delete_own_account() to authenticated;
