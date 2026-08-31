/* Shared figure library — site-style SVG line art on the dark theme.
 * Mirrors the Field Guide's currentColor figures (diagram style sheet in brand.css). */
window.Fig = (() => {
  const NS = "http://www.w3.org/2000/svg";
  const C = { text: "#e6edf3", muted: "#8b96a3", accent: "#58a6ff", deep: "#155799",
              good: "#3fb950", warn: "#d29922", bad: "#f85149", violet: "#bc8cff",
              bg: "#0d1117", elev: "#161b22", border: "#30363d" };

  function el(tag, attrs = {}, parent = null) {
    const e = document.createElementNS(NS, tag);
    for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v);
    if (parent) parent.appendChild(e);
    return e;
  }
  function svg(parent, vb, attrs = {}) {
    const s = el("svg", { viewBox: vb, ...attrs });
    (parent instanceof Element ? parent : document.body).appendChild(s);
    return s;
  }
  function div(parent, cls = "", style = "", html = "") {
    const d = document.createElement("div");
    d.className = cls; d.style.cssText = style; d.innerHTML = html;
    parent.appendChild(d);
    return d;
  }
  function text(parent, x, y, str, size = 28, fill = C.muted, anchor = "middle", family = "Inter", weight = 500) {
    const t = el("text", { x, y, "font-size": size, fill, "text-anchor": anchor,
                           "font-family": family, "font-weight": weight }, parent);
    t.textContent = str;
    return t;
  }

  // sine path: x0..x0+w, centered on y0, amp px, cycles over the width, phase rad
  function sinePath(x0, y0, w, amp, cycles, phase = 0, step = 4) {
    let d = "";
    for (let x = 0; x <= w; x += step) {
      const y = y0 - amp * Math.sin(2 * Math.PI * cycles * (x / w) + phase);
      d += (x === 0 ? "M" : "L") + (x0 + x).toFixed(1) + " " + y.toFixed(1);
    }
    return d;
  }
  // sine with per-x amplitude/frequency functions (for AM/FM/chirp)
  function wavePath(x0, y0, w, ampFn, phaseFn, step = 3) {
    let d = "";
    for (let x = 0; x <= w; x += step) {
      const y = y0 - ampFn(x / w) * Math.sin(phaseFn(x / w));
      d += (x === 0 ? "M" : "L") + (x0 + x).toFixed(1) + " " + y.toFixed(1);
    }
    return d;
  }
  // gaussian-ish spectrum bump
  function bumpPath(cx, base, halfw, height, floorY) {
    let d = `M${cx - halfw * 2.2} ${floorY}`;
    for (let x = -halfw * 2.2; x <= halfw * 2.2; x += 4) {
      const y = floorY - height * Math.exp(-(x * x) / (2 * halfw * halfw / 4));
      d += `L${(cx + x).toFixed(1)} ${y.toFixed(1)}`;
    }
    return d;
  }

  // stroke draw-in: set once at mount, drive with setDraw(path, p)
  function prepDraw(path) {
    const len = path.getTotalLength();
    path.style.strokeDasharray = len;
    path.dataset.len = len;
    path.style.strokeDashoffset = len;
  }
  function setDraw(path, p) {
    path.style.strokeDashoffset = path.dataset.len * (1 - Math.max(0, Math.min(1, p)));
  }

  function arrowDefs(s, id = "arr", color = C.muted) {
    const defs = el("defs", {}, s);
    const m = el("marker", { id, markerWidth: 10, markerHeight: 10, refX: 7, refY: 3.5, orient: "auto" }, defs);
    el("path", { d: "M0 0 L7 3.5 L0 7 z", fill: color }, m);
    return defs;
  }

  const fmt1 = n => (Math.round(n * 10) / 10).toString();
  return { NS, C, el, svg, div, text, sinePath, wavePath: wavePath, bumpPath, prepDraw, setDraw, arrowDefs, fmt1 };
})();
