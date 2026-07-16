# Supabase setup for gophertrunk.org

The site adds optional learner **accounts**, a community **forum**, richer **profiles**
(with avatar uploads), **public profile pages**, and a **content-moderation** system,
all backed by a hosted Supabase project. Because GitHub Pages is static and keeps no
secrets, everything runs client-side with the **public anon key**; all data safety comes
from **Row Level Security**, defined in the `schema*.sql` files.

You never give anyone the `service_role`/secret key, the database password, or OAuth
client secrets — those live only in the Supabase dashboard. The repo holds just the two
public values (project URL + anon key) in `docs/assets/js/supabase-config.js`.

## One-time setup

1. **Create a project** at <https://supabase.com/dashboard>.
2. **Run the schema.** SQL Editor → paste `schema.sql` → Run. (Re-runnable; idempotent.)
   For the forum (Phase 2), also run `schema-forum.sql` afterwards — it adds the
   `forum_categories` / `forum_threads` / `forum_posts` tables, their RLS policies, and
   a few starter categories.
   For forum moderation + live updates (Phase 3), also run
   `schema-forum-moderation.sql` — it adds admins (`app_admins` + `is_admin()`),
   `forum_reports`, server-side rate limiting, and registers the forum tables for
   Realtime. Grant yourself admin afterwards:
   `insert into public.app_admins (user_id) values ('<your-user-uuid>');` (find the uuid
   under Authentication → Users). Everything is additive and the client feature-detects
   it, so the forum works even before this file is applied.

   For richer profiles + avatar uploads (Phase 4), also run `schema-profiles.sql`, and
   create the avatar storage bucket (step 2b below). For the moderation system —
   report queue, user bans/suspensions, audit log (Phase 5) — also run
   `schema-moderation.sql`. Both are additive and feature-detected: `/profile/`, `/u/`,
   the forum, and `/moderation/` all work before they're applied — the new fields and
   controls simply don't appear yet.
2b. **Create the avatar storage bucket** (needed for Phase 4 avatar uploads) —
   Storage → **New bucket**:
   - **Name:** `avatars`
   - **Public bucket:** ON (avatars must be readable by everyone, including signed-out
     visitors viewing public profiles)
   - **File size limit:** `2 MB`
   - **Allowed MIME types:** `image/png, image/jpeg, image/webp`

   The row-level policies that let a signed-in user write only their *own* avatar
   (`avatars/<uid>/avatar`) are created by `schema-profiles.sql` — no manual policy
   editing needed. If you skip this bucket, everything else still works; avatar uploads
   just show a friendly "not set up yet" message and OAuth avatars keep working.
3. **Enable auth providers** — Authentication → Providers:
   - **GitHub** and **Google** OAuth (create the OAuth apps on GitHub/Google, paste the
     client id + secret into Supabase). Set each provider's callback to the Supabase
     `…/auth/v1/callback` URL shown in the dashboard.
   - **Email** — enable email+password and magic link (email OTP).
4. **URL configuration** — Authentication → URL Configuration:
   - **Site URL:** `https://gophertrunk.org`
   - **Redirect URLs:** add `https://gophertrunk.org/account/` and, for local dev,
     `http://localhost:4000/account/`.
5. **Email/SMTP** — we intentionally use Supabase's **built-in SMTP** for magic link and
   confirmation emails to keep everything on one service. Note its low default send rate
   and weaker deliverability; OAuth is the primary sign-in path. If email volume becomes
   a problem later, add a custom SMTP provider in Authentication → Emails — no site-side
   change is needed.
6. **Wire the site.** Copy the **Project URL** and **anon/publishable key** from Project
   Settings → API into `docs/assets/js/supabase-config.js` (replace the `YOUR-…`
   placeholders). Until you do, auth stays dormant and the site is unchanged.

## SQL files, in apply order

Run each once in the SQL Editor. All are idempotent; re-running is safe.

| File | Adds |
|---|---|
| `schema.sql` | Accounts: `profiles`, `learn_progress`, signup trigger, self-delete. |
| `schema-forum.sql` | Forum: categories / threads / posts + RLS. |
| `schema-forum-moderation.sql` | Admins (`app_admins` / `is_admin()`), `forum_reports`, rate limits, realtime. |
| `schema-profiles.sql` | Profile fields (bio, website, call sign, country/region/city) + avatar storage policies. Needs the `avatars` bucket (step 2b). |
| `schema-moderation.sql` | Bans/suspensions (`user_bans` / `is_banned()`), thread reports + report status, `moderation_actions` audit log. |

## Moderation & bans

- **Add a moderator** (grant admin access): see
  [`adding-a-moderator.md`](adding-a-moderator.md) for the step-by-step.
- **Review reports** on the site at `/moderation/` — a dashboard visible only to admins
  (it calls `is_admin()`; RLS denies the underlying rows to everyone else). From there you
  resolve or dismiss reports, lock threads, remove posts, and ban or suspend users. Every
  action is written to the `moderation_actions` audit log.
- **Bans** (`schema-moderation.sql`): a banned user keeps a valid session but a
  RESTRICTIVE RLS policy stops their posts/threads from being inserted, so the ban can't
  be bypassed from the browser. Set `expires_at` in the future for a temporary
  **suspension** (auto-lifts); leave it null for a permanent ban. You can also manage
  bans directly in SQL:
  ```sql
  insert into public.user_bans (user_id, reason) values ('<uuid>', 'spam');            -- permanent
  insert into public.user_bans (user_id, reason, expires_at)
    values ('<uuid>', 'cooldown', now() + interval '7 days');                          -- 7-day suspension
  delete from public.user_bans where user_id = '<uuid>';                               -- unban
  ```

## Verify

- Sign in on `/account/` with each enabled method; confirm the nav shows your name.
- Complete a lesson signed out (localStorage), then sign in and confirm the completion
  syncs and appears on a second browser/device.
- Confirm a signed-in user can only read/write their own `learn_progress` rows (RLS).
- Try the account-deletion control; confirm the profile + progress rows are gone.
- **Profiles:** on `/profile/`, upload an avatar and save a call sign + location; confirm
  they appear on your public profile at `/u/?u=<username>`. Confirm you cannot write into
  another user's avatar folder (RLS rejects `avatars/<other-uid>/…`).
- **Moderation:** with a second account, file a report; as admin, see it in `/moderation/`,
  resolve it, and ban that user — confirm their next forum post is rejected. Confirm a
  non-admin loading `/moderation/` sees only the "administrator" notice and no report data.

The `schema*.sql` files are documentation/version history — the site does not apply them.
Keep them in sync with what you run in the dashboard.
