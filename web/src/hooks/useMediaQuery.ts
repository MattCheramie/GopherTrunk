import { useEffect, useState } from "react";

// useMediaQuery tracks a CSS media query and re-renders when it flips.
// Used by the app shell to decide between the desktop sidebar and the
// mobile bottom-nav/drawer. SSR-safe: returns `false` when there's no
// `window` (we never SSR, but jsdom-based tests exercise this path).
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => {
    if (typeof window === "undefined" || !window.matchMedia) return false;
    return window.matchMedia(query).matches;
  });

  useEffect(() => {
    if (typeof window === "undefined" || !window.matchMedia) return;
    const mql = window.matchMedia(query);
    const onChange = () => setMatches(mql.matches);
    onChange();
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, [query]);

  return matches;
}

// Tailwind's `sm` breakpoint (640px). Above it we show the desktop
// sidebar; below it the bottom nav + drawer. Kept in sync with
// tailwind.config.ts manually (Tailwind doesn't expose breakpoints to JS).
export function useIsDesktop(): boolean {
  return useMediaQuery("(min-width: 640px)");
}
