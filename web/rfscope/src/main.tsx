import React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
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

const root = document.getElementById("root");
if (!root) throw new Error("missing #root");

createRoot(root).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
