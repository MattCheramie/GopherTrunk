import React from "react";
import { createRoot } from "react-dom/client";
import { HashRouter } from "react-router-dom";
import { registerSW } from "virtual:pwa-register";
import { App } from "./App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { prefs, themeAttr } from "./store/prefs";
import "./styles.css";

// Apply the stored theme + density before the first render so the UI
// never flashes the default palette or spacing.
document.documentElement.dataset.theme = themeAttr(prefs.theme());
document.documentElement.dataset.density = prefs.density();

// Register the service worker. autoUpdate strategy: if a new bundle
// is deployed, Workbox swaps it in on the next reload — no user
// prompt needed for a small operator tool.
registerSW({ immediate: true });

const root = document.getElementById("root");
if (!root) throw new Error("missing #root");

createRoot(root).render(
  <React.StrictMode>
    <ErrorBoundary>
      <HashRouter>
        <App />
      </HashRouter>
    </ErrorBoundary>
  </React.StrictMode>,
);
