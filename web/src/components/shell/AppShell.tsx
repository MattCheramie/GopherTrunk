import { useCallback, useEffect, useState, type ReactNode } from "react";
import { Sidebar } from "./Sidebar";
import { BottomNav } from "./BottomNav";
import { MobileDrawer } from "./MobileDrawer";
import { TopBar } from "./TopBar";
import { CommandPalette } from "../CommandPalette";
import { useIsDesktop } from "../../hooks/useMediaQuery";
import { prefs } from "../../store/prefs";

interface Props {
  children: ReactNode;
}

// AppShell: the application layout frame. Renders the desktop sidebar,
// the mobile top bar + bottom nav + drawer, and the global command
// palette around the routed page content. All four navigation surfaces
// are driven by the shared nav registry, so every panel is reachable on
// every device.
//
// This component is pure chrome — it owns no connection or data state.
// App.tsx keeps the connection effects (issue #290) and passes the
// <Routes> in as children.
export function AppShell({ children }: Props) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(() => prefs.sidebarCollapsed());
  const isDesktop = useIsDesktop();

  // The drawer is a mobile-only surface; if the viewport grows to
  // desktop while it's open (rotation, window resize), close it so it
  // doesn't linger over the now-visible sidebar.
  useEffect(() => {
    if (isDesktop && drawerOpen) setDrawerOpen(false);
  }, [isDesktop, drawerOpen]);

  const toggleCollapse = useCallback(() => {
    setCollapsed((c) => {
      const next = !c;
      prefs.setSidebarCollapsed(next);
      return next;
    });
  }, []);

  // Global ⌘K / Ctrl-K toggles the command palette.
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setPaletteOpen((o) => !o);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return (
    <div className="min-h-full flex">
      <Sidebar collapsed={collapsed} onToggleCollapse={toggleCollapse} />

      <div className="flex-1 flex flex-col min-w-0">
        <TopBar
          onOpenMore={() => setDrawerOpen(true)}
          onOpenPalette={() => setPaletteOpen(true)}
        />
        <main className="flex-1 p-3 sm:p-4 pb-24 sm:pb-4">{children}</main>
      </div>

      <BottomNav onOpenMore={() => setDrawerOpen(true)} />
      <MobileDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} />
      <CommandPalette
        open={paletteOpen}
        onClose={() => setPaletteOpen(false)}
      />
    </div>
  );
}
