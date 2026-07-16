-- GopherTrunk — Supabase schema (Phase 4: richer profiles + avatar uploads)
--
-- Apply AFTER schema.sql (and, if you run the forum, after schema-forum.sql /
-- schema-forum-moderation.sql — order between this file and the forum files does
-- not matter, they touch different objects). SQL Editor -> paste -> Run, or
-- `supabase db push`. Idempotent: re-running is safe.
--
-- Everything here is ADDITIVE and the site FEATURE-DETECTS it: /profile/, /u/,
-- and the forum all keep working before this file is applied — the new fields
-- simply don't appear yet. Same security model as the earlier phases: the browser
-- holds only the public anon key, so every object is guarded by Row Level
-- Security, never by key secrecy.
--
-- Two things this file does:
--   1. Adds optional profile columns (bio, website, ham radio call sign, and an
--      optional country / region / city) with light validation.
--   2. Adds Row Level Security policies for a public `avatars` Storage bucket so a
--      signed-in user can upload/replace only their OWN avatar. You must still
--      CREATE the bucket in the dashboard first — see docs/supabase/README.md.

-- ---------------------------------------------------------------------------
-- 1. profiles — new optional columns.
--
-- All nullable, so existing rows and the handle_new_user() signup trigger keep
-- working unchanged. `avatar_url` and `created_at` already exist (from schema.sql)
-- and are reused for the uploaded avatar and the "member since" date — no new
-- column needed for those.
-- ---------------------------------------------------------------------------
alter table public.profiles add column if not exists bio      text;
alter table public.profiles add column if not exists website  text;
alter table public.profiles add column if not exists callsign text;   -- amateur radio call sign
alter table public.profiles add column if not exists country  text;   -- ISO-3166 alpha-2, e.g. 'US'
alter table public.profiles add column if not exists region   text;   -- state / province (free text)
alter table public.profiles add column if not exists city     text;   -- free text

-- Constraints. ADD CONSTRAINT has no IF NOT EXISTS, so each is guarded on
-- pg_constraint. The regexes are deliberately permissive supersets — real
-- validation also happens client-side; these stop obviously bad or oversized
-- values from reaching the table.
do $$ begin
  if not exists (select 1 from pg_constraint where conname = 'profiles_bio_chk') then
    alter table public.profiles add constraint profiles_bio_chk
      check (bio is null or char_length(bio) <= 1000);
  end if;
  if not exists (select 1 from pg_constraint where conname = 'profiles_website_chk') then
    alter table public.profiles add constraint profiles_website_chk
      check (website is null or website ~* '^https?://[^ ]{3,200}$');
  end if;
  if not exists (select 1 from pg_constraint where conname = 'profiles_callsign_chk') then
    -- 3–10 chars, A–Z / 0–9 / '/', e.g. W1AW or W1AW/4. The client uppercases
    -- before saving, so lowercase input is normalised rather than rejected.
    alter table public.profiles add constraint profiles_callsign_chk
      check (callsign is null or callsign ~ '^[A-Z0-9/]{3,10}$');
  end if;
  if not exists (select 1 from pg_constraint where conname = 'profiles_country_chk') then
    alter table public.profiles add constraint profiles_country_chk
      check (country is null or country ~ '^[A-Z]{2}$');
  end if;
  if not exists (select 1 from pg_constraint where conname = 'profiles_region_chk') then
    alter table public.profiles add constraint profiles_region_chk
      check (region is null or char_length(region) <= 80);
  end if;
  if not exists (select 1 from pg_constraint where conname = 'profiles_city_chk') then
    alter table public.profiles add constraint profiles_city_chk
      check (city is null or char_length(city) <= 80);
  end if;
end $$;

-- NOTE on `username`: it is intentionally NOT constrained here. Existing values
-- are seeded from OAuth metadata (GitHub usernames allow hyphens), so a
-- retroactive CHECK could fail this migration. Username format is validated in
-- the client (^[a-zA-Z0-9_-]{3,30}$, lowercased) on top of the existing UNIQUE.
-- The existing profiles RLS policies (public select, insert/update own) already
-- cover every new column, so no policy change is needed.

-- ---------------------------------------------------------------------------
-- 2. Avatar uploads — Storage RLS.
--
-- PREREQUISITE (dashboard, one time): create a PUBLIC bucket named `avatars`
--   Storage -> New bucket -> name `avatars`, Public = on,
--   file size limit 2 MB, allowed MIME types: image/png, image/jpeg, image/webp.
-- See docs/supabase/README.md. This file only adds the row-level policies.
--
-- Path convention: `avatars/<uid>/avatar` — one fixed object key per user,
-- overwritten with upsert. The first path segment must equal the caller's uid,
-- so a signed-in user can only write inside their own folder. Reads are public
-- (the bucket is public, and read is world-open below) so avatars show for
-- everyone, including signed-out visitors on public profiles.
-- ---------------------------------------------------------------------------
drop policy if exists "avatars public read" on storage.objects;
create policy "avatars public read" on storage.objects
  for select using (bucket_id = 'avatars');

drop policy if exists "avatars insert own" on storage.objects;
create policy "avatars insert own" on storage.objects
  for insert to authenticated with check (
    bucket_id = 'avatars' and (storage.foldername(name))[1] = auth.uid()::text
  );

drop policy if exists "avatars update own" on storage.objects;
create policy "avatars update own" on storage.objects
  for update to authenticated using (
    bucket_id = 'avatars' and (storage.foldername(name))[1] = auth.uid()::text
  );

drop policy if exists "avatars delete own" on storage.objects;
create policy "avatars delete own" on storage.objects
  for delete to authenticated using (
    bucket_id = 'avatars' and (storage.foldername(name))[1] = auth.uid()::text
  );
