# Supabase (database-as-code)

This directory is the Supabase CLI project for GopherTrunk's backend (accounts,
learning-path progress, forum, moderation). The database schema is now managed
as versioned **migrations** instead of the documentation-only SQL that lived
under [`docs/supabase/`](../docs/supabase).

## Layout

| Path | Purpose |
| --- | --- |
| `config.toml` | Local/remote project configuration (ports, auth, storage, realtime). |
| `migrations/` | Ordered, idempotent SQL migrations — the source of truth. |
| `seed.sql` | Local-only fixtures applied after migrations by `supabase db reset`. |
| `.gitignore` | Ignores CLI local state (`.branches`, `.temp`, `.env`). |

`migrations/20260718155556_initial_schema.sql` is the consolidated baseline. It
concatenates the five original phase files **in dependency order**:

1. `schema.sql` — profiles, `learn_progress`, `delete_own_account()`
2. `schema-forum.sql` — forum categories / threads / posts
3. `schema-forum-moderation.sql` — admins, reports, rate limits, realtime
4. `schema-profiles.sql` — richer profile columns + `avatars` Storage RLS
5. `schema-moderation.sql` — bans, report workflow, moderation audit log

The originals under `docs/supabase/` are kept as annotated documentation /
history; new schema changes should be added as **new migration files** here.

## Common commands

```bash
# One-time: install the CLI (https://supabase.com/docs/guides/cli) and link.
supabase link --project-ref <your-project-ref>

# Local development stack (Postgres + Studio + Auth + Storage):
supabase start
supabase db reset            # re-apply all migrations + seed.sql from scratch

# Author a new change:
supabase migration new <name>   # creates migrations/<timestamp>_<name>.sql
#   ...edit the file...
supabase db reset               # verify locally

# Apply pending migrations to the linked remote project:
supabase db push
```

## Manual prerequisite (Storage bucket)

A migration cannot create a Storage bucket. Before the avatar-upload policies in
the initial migration are useful, create a **public** bucket named `avatars`
(2 MB limit; MIME `image/png`, `image/jpeg`, `image/webp`) in the dashboard or
via the Storage API. The bucket's row-level policies are already in the
migration.

## Admin grants

There is intentionally no client path to grant admin. After a user signs up,
grant them via SQL:

```sql
insert into public.app_admins (user_id) values ('<auth-user-uuid>');
```
