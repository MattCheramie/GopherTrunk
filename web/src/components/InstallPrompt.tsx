import { useEffect, useState } from "react";
import { prefs } from "../store/prefs";

// PWA install prompt. Browsers that implement `beforeinstallprompt`
// (Chrome on Android, Edge on desktop) fire it once; we capture the
// event so the user can install on demand from the Settings panel.
// iOS doesn't fire the event — Safari requires Share → Add to Home
// Screen — so this banner only appears where the API is supported.

interface BeforeInstallPromptEvent extends Event {
  readonly platforms: string[];
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

export function InstallPrompt() {
  const [deferred, setDeferred] = useState<BeforeInstallPromptEvent | null>(
    null,
  );
  const [dismissed, setDismissed] = useState(prefs.installPromptDismissed());

  useEffect(() => {
    const onPrompt = (e: Event) => {
      e.preventDefault();
      setDeferred(e as BeforeInstallPromptEvent);
    };
    window.addEventListener("beforeinstallprompt", onPrompt);
    return () => window.removeEventListener("beforeinstallprompt", onPrompt);
  }, []);

  if (!deferred || dismissed) return null;

  return (
    <div className="fixed top-3 inset-x-3 sm:left-auto sm:max-w-sm z-40 panel p-3 flex items-center gap-3">
      <div className="flex-1 text-sm">
        <p className="font-medium">Install GopherTrunk?</p>
        <p className="text-muted text-xs">
          Add the app to your home screen for a full-screen, offline-
          capable launch.
        </p>
      </div>
      <button
        className="btn-primary text-xs"
        onClick={async () => {
          await deferred.prompt();
          const choice = await deferred.userChoice;
          if (choice.outcome === "accepted") {
            setDeferred(null);
          } else {
            prefs.setInstallPromptDismissed(true);
            setDismissed(true);
          }
        }}
      >
        Install
      </button>
      <button
        className="btn-ghost text-xs"
        aria-label="Dismiss"
        onClick={() => {
          prefs.setInstallPromptDismissed(true);
          setDismissed(true);
        }}
      >
        ✕
      </button>
    </div>
  );
}
