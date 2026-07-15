import { describe, it, expect } from "vitest";
import { SW_NAVIGATE_FALLBACK_DENYLIST } from "./swDenylist";
import { NAV_ITEMS } from "../nav/registry";

// Every `external` nav item is a daemon-served sibling console living at a
// subpath under the main console's service-worker "/" scope, opened in a new
// tab via a plain <a href> navigation. If such a path is not on the SW's
// navigateFallbackDenylist, the SW answers the navigation with our own cached
// index.html and the console opens blank (the RFScope/SigLab blank-page bug;
// previously seen and fixed for /config/). This pins that invariant so adding
// a new external console without denylisting it fails CI.
describe("SW navigateFallbackDenylist", () => {
  it("denylists every external console path so the SW doesn't shadow it", () => {
    const external = NAV_ITEMS.filter((i) => i.external);
    expect(external.length).toBeGreaterThan(0);
    for (const item of external) {
      const matched = SW_NAVIGATE_FALLBACK_DENYLIST.some((re) => re.test(item.to));
      expect(matched, `${item.to} must be on the SW navigateFallbackDenylist`).toBe(true);
    }
  });
});
