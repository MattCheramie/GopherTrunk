/*
 * Public Supabase connection config for gophertrunk.org.
 *
 * BOTH values below are PUBLIC and safe to commit. The anon / publishable key is
 * designed to ship in the browser; all data safety comes from Row Level Security
 * policies in the database (docs/supabase/schema.sql), NOT from hiding these.
 *
 * NEVER put the service_role / secret key, the database password, or any OAuth
 * client secret in this file — those live only in the Supabase dashboard.
 *
 * Fill in your project's values from:
 *   Supabase Dashboard -> Project Settings -> API
 *     url     = "Project URL"        (https://<ref>.supabase.co)
 *     anonKey = "anon"/"publishable" key (browser-safe; NOT service_role)
 *
 * Until both are real (placeholders replaced), auth.js stays dormant and the
 * site behaves exactly as before: learning-path progress lives in localStorage
 * only, and every page is fully usable. Nothing here is required for the site
 * to build or render.
 */
window.GT_SUPABASE = {
  url: 'https://fzundthahevgjrwoesjz.supabase.co',
  anonKey: 'sb_publishable_Qc0e9q7BHp2gGirMKM5Ftg_issUBvk0'
};
