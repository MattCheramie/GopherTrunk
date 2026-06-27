import React from "react";
import { createRoot } from "react-dom/client";
import { registerSW } from "virtual:pwa-register";
import { App } from "./App";
import { setToken } from "./api/client";
import "./styles.css";

// Honor the theme/density the operator picked in the main console
// (shared via localStorage), applied pre-render so there's no flash.
(() => {
  try {
    const theme = localStorage.getItem("gt.ui.theme");
    document.documentElement.dataset.theme =
      theme === "monochrome" ? "mono" : theme === "light" ? "light" : "dark";
    const density = localStorage.getItem("gt.ui.density");
    document.documentElement.dataset.density =
      density === "compact" ? "compact" : "comfortable";
  } catch {
    document.documentElement.dataset.theme = "dark";
  }
})();
registerSW({ immediate: true });

// Bearer-token bootstrap for non-loopback `cryptolab serve` binds. The
// default loopback bind needs no token; remote deployments can hand out a
// link carrying `#token=...`. Strip the hash immediately so the token isn't
// left in the URL/bookmarks/screenshots.
(() => {
  if (!window.location.hash.includes("=")) return;
  const params = new URLSearchParams(window.location.hash.slice(1));
  const token = params.get("token");
  if (token) {
    setToken(token);
    history.replaceState(null, "", window.location.pathname + window.location.search);
  }
})();

const root = document.getElementById("root");
if (!root) throw new Error("missing #root");

createRoot(root).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
