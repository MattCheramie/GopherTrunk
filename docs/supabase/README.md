# Supabase setup for gophertrunk.org

The site adds optional learner **accounts** (and, in Phase 2, a **forum**) backed by a
hosted Supabase project. Because GitHub Pages is static and keeps no secrets, everything
runs client-side with the **public anon key**; all data safety comes from **Row Level
Security**, defined in `schema.sql`.

You never give anyone the `service_role`/secret key, the database password, or OAuth
client secrets — those live only in the Supabase dashboard. The repo holds just the two
public values (project URL + anon key) in `docs/assets/js/supabase-config.js`.

## One-time setup

1. **Create a project** at <https://supabase.com/dashboard>.
2. **Run the schema.** SQL Editor → paste `schema.sql` → Run. (Re-runnable; idempotent.)
   For the forum (Phase 2), also run `schema-forum.sql` afterwards — it adds the
   `forum_categories` / `forum_threads` / `forum_posts` tables, their RLS policies, and
   a few starter categories.
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

## Verify

- Sign in on `/account/` with each enabled method; confirm the nav shows your name.
- Complete a lesson signed out (localStorage), then sign in and confirm the completion
  syncs and appears on a second browser/device.
- Confirm a signed-in user can only read/write their own `learn_progress` rows (RLS).
- Try the account-deletion control; confirm the profile + progress rows are gone.

`schema.sql` is documentation/version history — the site does not apply it. Keep it in
sync with what you run in the dashboard.
