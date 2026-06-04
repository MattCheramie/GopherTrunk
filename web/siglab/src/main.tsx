import React from "react";
import { createRoot } from "react-dom/client";
import { HashRouter } from "react-router-dom";
import { registerSW } from "virtual:pwa-register";
import { App } from "./App";
import "./styles.css";

document.documentElement.classList.add("dark");
registerSW({ immediate: true });

const root = document.getElementById("root");
if (!root) throw new Error("missing #root");

createRoot(root).render(
  <React.StrictMode>
    <HashRouter>
      <App />
    </HashRouter>
  </React.StrictMode>,
);
