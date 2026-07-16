# Adding a moderator (admin) to GopherTrunk

Moderators — the people who can open the `/moderation/` dashboard, lock threads,
remove posts, review reports, and ban/suspend users — are just users whose id is
listed in the `public.app_admins` table. Nobody is a moderator by default; you
grant it deliberately, and only from inside Supabase (SQL or dashboard), never
from the browser. This guide walks through granting it to someone.

> **Do this once per person.** Repeat the steps below with each new moderator's
> UUID. There is no self-service — being able to run SQL in the Supabase
> dashboard *is* the gate that keeps moderator access controlled.

## How moderator access works

The `/moderation/` page and every admin power are gated by the `is_admin()`
function, which simply checks: *"is my user id in `public.app_admins`?"*

- `is_admin()` and the `app_admins` table are created by
  `schema-forum-moderation.sql` (Phase 3).
- The ban/report/audit tooling those admins use is created by
  `schema-moderation.sql` (Phase 5).
- Row Level Security enforces it on the server, so a non-admin who opens
  `/moderation/` just sees "You must be an administrator" and gets no data —
  the gate can't be bypassed from the client.

## Prerequisites

1. **`schema-forum-moderation.sql` has been run** in your Supabase project (it
   creates `app_admins` + `is_admin()`). If you've run the full setup in
   `README.md`, this is already done.
2. **The person has signed into the live site at least once** with the account
   you want to promote — sign in at
   <https://gophertrunk.org/account/> using GitHub, Google, email, or a magic
   link. Until they sign in once, no user record exists to promote.

## Step 1 — Find the new moderator's user UUID

Every account has a UUID like `a1b2c3d4-5678-90ab-cdef-1234567890ab`. Get it one
of two ways.

### Option A — Dashboard (easiest)

1. Supabase dashboard → **Authentication** → **Users**.
2. Find the person's row (search by email or provider username).
3. Click the row — copy the **User UID**.

### Option B — SQL

Supabase dashboard → **SQL Editor** → **New query** → run whichever you have:

```sql
-- by email:
select id, email from auth.users where email = 'them@example.com';

-- or, if they've already set a profile username on /profile/:
select id, username, display_name from public.profiles where username = 'their-username';
```

Copy the `id` value.

## Step 2 — Grant moderator access

Supabase dashboard → **SQL Editor** → **New query** → paste, replace the
placeholder UUID with the one from Step 1, and **Run**:

```sql
insert into public.app_admins (user_id)
values ('a1b2c3d4-5678-90ab-cdef-1234567890ab')
on conflict (user_id) do nothing;
```

Prefer not to copy a UUID around? Grant by email in a single statement instead:

```sql
insert into public.app_admins (user_id)
select id from auth.users where email = 'them@example.com'
on conflict (user_id) do nothing;
```

`on conflict (user_id) do nothing` makes it safe to run more than once — a person
who is already a moderator stays one, with no error.

## Step 3 — Confirm it worked

```sql
select p.username, p.display_name, a.user_id, a.created_at
from public.app_admins a
join public.profiles p on p.id = a.user_id
order by a.created_at;
```

You should see the new moderator in the list. Then have them (while **signed in**)
open <https://gophertrunk.org/moderation/> — they'll get the dashboard instead of
the "administrator" notice. If they already had the page open, a refresh is
needed; the admin check runs on page load.

## Removing a moderator

```sql
delete from public.app_admins where user_id = '<uuid>';
```

Or by email:

```sql
delete from public.app_admins
where user_id = (select id from auth.users where email = 'them@example.com');
```

They lose dashboard access immediately (on their next page load / RLS check).

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `insert or update on table "app_admins" violates foreign key constraint` | The UUID has no matching `profiles` row — the person hasn't signed into the site yet, or you used the wrong id. Have them sign in once, then re-run. |
| `relation "public.app_admins" does not exist` | `schema-forum-moderation.sql` hasn't been run. Run it first (see `README.md`). |
| They still see "You must be an administrator" | They're on the page from before the grant (refresh), not signed in, or signed in as a *different* account than the UUID you granted. Confirm with the Step 3 query. |
| Want to check who is a moderator | Run the Step 3 confirmation query any time. |

## Quick reference

```sql
-- Grant (by email):
insert into public.app_admins (user_id)
select id from auth.users where email = 'them@example.com'
on conflict (user_id) do nothing;

-- List moderators:
select p.username, a.user_id, a.created_at
from public.app_admins a join public.profiles p on p.id = a.user_id;

-- Revoke:
delete from public.app_admins where user_id = '<uuid>';
```

> **Moderator ≠ ban.** `app_admins` *grants* moderation power. `user_bans`
> *removes* a user's ability to post. They're separate tables with opposite
> purposes — adding someone to `app_admins` never bans them, and banning someone
> never touches their moderator status.
