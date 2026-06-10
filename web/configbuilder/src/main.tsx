import React from "react";
import { createRoot } from "react-dom/client";
import { registerSW } from "virtual:pwa-register";
import { App } from "./App";
import { setToken } from "./api/client";
import "./styles.css";

document.documentElement.classList.add("dark");
registerSW({ immediate: true });

// Bearer-token bootstrap for non-loopback `config serve` binds. The default
// loopback bind needs no token, so the top bar has no token field; remote
// deployments hand out a one-click link carrying `#token=...`. Strip the hash
// immediately so the token isn't left in the URL/bookmarks/screenshots.
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
