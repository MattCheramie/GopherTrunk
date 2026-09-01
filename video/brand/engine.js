/* GopherTrunk video motion-graphics engine.
 *
 * Contract (video/gophertrunk-video-pipeline.md §8): the page exposes
 * window.seek(t_seconds) which synchronously sets ALL visual state for time t —
 * a pure function of t. No CSS animations, no rAF, no wall-clock.
 *
 * The renderer injects window.RENDER = {
 *   mode: "wide"|"vert",          // 1920x1080 or 1080x1920 stage
 *   timeline: {...},              // per-segment timeline JSON (from tts.py)
 *   logo: "data:image/png;...",   // gopher logo
 *   captions: bool,               // burn captions (verticals)
 * }
 * Timeline events: {b, t, dur, visual, variant, arg, text}
 * Consecutive blocks with the same visual name share one component instance,
 * so multi-beat diagrams animate continuously across variants.
 */
(function () {
  "use strict";
  const R = window.RENDER;
  const vert = R.mode === "vert";
  const W = vert ? 1080 : 1920, H = vert ? 1920 : 1080;
  const C = {
    bg: "#0d1117", elev: "#161b22", text: "#e6edf3", mut: "#8b96a3",
    acc: "#58a6ff", deep: "#155799", soft: "#e8f0fa", bor: "#30363d",
    good: "#3fb950", warn: "#d29922", bad: "#f85149",
  };
  const SANS = '"DejaVu Sans", system-ui, sans-serif';
  const MONO = '"DejaVu Sans Mono", monospace';

  // ---------- DOM helpers ----------
  const stage = document.getElementById("stage");
  stage.style.width = W + "px"; stage.style.height = H + "px";
  function S(tag, attrs, parent) {
    const el = document.createElementNS("http://www.w3.org/2000/svg", tag);
    for (const k in attrs) el.setAttribute(k, attrs[k]);
    (parent || stage).appendChild(el); return el;
  }
  function svgRoot() {
    const el = S("svg", { width: W, height: H, viewBox: `0 0 ${W} ${H}` });
    el.style.position = "absolute"; el.style.left = 0; el.style.top = 0;
    return el;
  }
  function T(parent, x, y, str, size, opts) {
    opts = opts || {};
    const el = S("text", {
      x, y, "font-size": size, fill: opts.fill || C.text,
      "text-anchor": opts.anchor || "middle",
      "font-family": opts.mono ? MONO : SANS,
      "font-weight": opts.w || 400, opacity: opts.o == null ? 1 : opts.o,
      "letter-spacing": opts.ls || 0,
    }, parent);
    el.textContent = str; return el;
  }
  function RB(parent, x, y, w, h, opts) { // rounded box
    opts = opts || {};
    return S("rect", {
      x, y, width: w, height: h, rx: opts.rx == null ? 10 : opts.rx,
      fill: opts.fill || "none", stroke: opts.stroke || "none",
      "stroke-width": opts.sw || 2, opacity: opts.o == null ? 1 : opts.o,
      "stroke-dasharray": opts.dash || "none",
      "fill-opacity": opts.fo == null ? 1 : opts.fo,
    }, parent);
  }
  function L(parent, x1, y1, x2, y2, opts) {
    opts = opts || {};
    return S("line", {
      x1, y1, x2, y2, stroke: opts.stroke || C.mut,
      "stroke-width": opts.sw || 2, opacity: opts.o == null ? 1 : opts.o,
      "stroke-dasharray": opts.dash || "none",
    }, parent);
  }
  const clamp = (x, a, b) => Math.max(a, Math.min(b, x));
  const ez = (p) => { p = clamp(p, 0, 1); return 1 - Math.pow(1 - p, 3); };
  const fi = (vt, t0, d) => ez((vt - t0) / (d || 0.4));       // fade-in 0..1
  const rnd = (i) => { const x = Math.sin(i * 127.1 + 0.37) * 43758.5453; return x - Math.floor(x); };
  const lerp = (a, b, p) => a + (b - a) * p;
  function op(el, v) { el.setAttribute("opacity", clamp(v, 0, 1)); }

  // Content geometry. Wide diagrams draw on a virtual 1200x760 canvas centered;
  // vert mode stacks a header above and scales the canvas into 1000px width.
  function canvasGroup(root, opts) {
    opts = opts || {};
    const vw = opts.vw || 1200, vh = opts.vh || 760;
    let sc, tx, ty;
    if (!vert) { sc = opts.scW || 1.18; tx = W / 2 - (vw * sc) / 2; ty = opts.tyW == null ? 170 : opts.tyW; }
    else { sc = 1000 / vw; ty = opts.tyV == null ? 430 : opts.tyV; tx = W / 2 - (vw * sc) / 2; }
    const g = S("g", { transform: `translate(${tx},${ty}) scale(${sc})` }, root);
    return g;
  }
  function heading(root, str, sub) {
    const y = vert ? 250 : 105;
    T(root, W / 2, y, str, vert ? 58 : 50, { w: 700 });
    if (sub) T(root, W / 2, y + (vert ? 64 : 52), sub, vert ? 34 : 30, { fill: C.mut });
  }

  // ---------- shared scene bits ----------
  function radioIcon(g, x, y, s, color) { // small handheld radio glyph
    const gg = S("g", { transform: `translate(${x},${y}) scale(${s || 1})` }, g);
    RB(gg, -14, -18, 28, 40, { rx: 6, fill: color || C.mut, fo: 0.9 });
    L(gg, 8, -18, 8, -34, { stroke: color || C.mut, sw: 4 });
    return gg;
  }
  function towerIcon(g, x, y, s, color) {
    const c = color || C.acc;
    const gg = S("g", { transform: `translate(${x},${y}) scale(${s || 1})` }, g);
    L(gg, -22, 40, 0, -40, { stroke: c, sw: 4 }); L(gg, 22, 40, 0, -40, { stroke: c, sw: 4 });
    L(gg, -13, 8, 13, 8, { stroke: c, sw: 3 }); L(gg, -8, -14, 8, -14, { stroke: c, sw: 3 });
    S("circle", { cx: 0, cy: -44, r: 5, fill: c }, gg);
    return gg;
  }

  // ============================================================
  // Components. factory(inst, ctx) -> {update(vt, st)}
  //   vt: seconds since instance mount; st: {variant, vv (s since variant), arg}
  // ============================================================
  const F = {};

  // ---- title card (hard cut) ----
  F.title = (inst) => {
    const root = svgRoot();
    const term = inst.variants[0].arg || "";
    S("rect", { x: 0, y: 0, width: W, height: 6, fill: C.acc }, root);
    T(root, W / 2, H / 2 - (vert ? 130 : 110), "GOPHERTRUNK  FIELD  GUIDE", vert ? 34 : 30, { fill: C.mut, ls: 8 });
    const sz = term.length > 14 ? (vert ? 84 : 104) : (vert ? 104 : 128);
    T(root, W / 2, H / 2 + 10, term, sz, { w: 700 });
    L(root, W / 2 - 140, H / 2 + 70, W / 2 + 140, H / 2 + 70, { stroke: C.acc, sw: 5 });
    T(root, W - 60, H - 44, R.timeline.seg, 26, { fill: C.mut, anchor: "end", mono: true });
    if (R.logo) {
      const im = S("image", { x: 60, y: H - 118, width: 74, height: 74, href: R.logo }, root);
      im.setAttribute("opacity", 0.85);
    }
    return { update() {} };
  };

  // ---- vertical hook card ----
  F.hook = (inst) => {
    const root = svgRoot();
    const txt = inst.variants[0].arg || "";
    S("rect", { x: 0, y: 0, width: W, height: 6, fill: C.acc }, root);
    T(root, W / 2, 320, "GOPHERTRUNK FIELD GUIDE", 30, { fill: C.mut, ls: 6 });
    const words = txt.split(" "); const lines = []; let cur = "";
    for (const w2 of words) { if ((cur + " " + w2).trim().length > 18) { lines.push(cur.trim()); cur = w2; } else cur += " " + w2; }
    if (cur.trim()) lines.push(cur.trim());
    const g = S("g", {}, root);
    lines.forEach((ln, i) => T(g, W / 2, 560 + i * 110, ln, 88, { w: 700 }));
    L(root, W / 2 - 150, 560 + lines.length * 110, W / 2 + 150, 560 + lines.length * 110, { stroke: C.acc, sw: 6 });
    return { update(vt) { op(g, fi(vt, 0.05, 0.3)); } };
  };

  // ---- recap card ----
  F.recap = (inst) => {
    const root = svgRoot();
    const [slug, rest] = (inst.variants[0].arg || "|").split("|");
    const bullets = rest.split(/[①②③]/).map(s => s.trim()).filter(Boolean);
    const term = R.timeline.title || "";
    T(root, W / 2, vert ? 280 : 130, term.toUpperCase(), vert ? 40 : 36, { fill: C.mut, ls: 5 });
    T(root, W / 2, vert ? 350 : 195, "Recap", vert ? 64 : 56, { w: 700 });
    const bw = vert ? 940 : 1100, bx = W / 2 - bw / 2;
    const items = bullets.map((b, i) => {
      const y = (vert ? 500 : 300) + i * (vert ? 240 : 170);
      const g = S("g", {}, root);
      RB(g, bx, y, bw, vert ? 190 : 130, { fill: C.elev, stroke: C.bor, rx: 12 });
      S("circle", { cx: bx + (vert ? 70 : 60), cy: y + (vert ? 95 : 65), r: 30, fill: "none", stroke: C.acc, "stroke-width": 3 }, g);
      T(g, bx + (vert ? 70 : 60), y + (vert ? 107 : 77), String(i + 1), 34, { fill: C.acc, w: 700 });
      // wrap bullet text
      const maxc = vert ? 34 : 52; const words = b.split(" "); const lines = []; let cur = "";
      for (const w2 of words) { if ((cur + " " + w2).trim().length > maxc) { lines.push(cur.trim()); cur = w2; } else cur += " " + w2; }
      if (cur.trim()) lines.push(cur.trim());
      lines.forEach((ln, j) => T(g, bx + (vert ? 130 : 120), y + (vert ? 80 : 55) + j * (vert ? 46 : 42) + (lines.length === 1 ? (vert ? 30 : 22) : 0), ln, vert ? 36 : 34, { anchor: "start" }));
      op(g, 0); return g;
    });
    const foot = T(root, W / 2, vert ? 1560 : (H - 90), "Full write-up → gophertrunk.org/reference/" + slug + "/", vert ? 34 : 32, { fill: C.acc, mono: true, o: 0 });
    const bdur = inst.t1 - inst.t0;
    const step = Math.min(2.2, Math.max(0.9, (bdur - 3) / Math.max(1, bullets.length)));
    return { update(vt) {
      items.forEach((g, i) => op(g, fi(vt, 0.3 + i * step, 0.5)));
      op(foot, fi(vt, 0.3 + bullets.length * step, 0.6));
    } };
  };

  // ---- definition / variants card ----
  F.defcard = (inst) => {
    const root = svgRoot();
    let g, title, body;
    function build(arg) {
      if (g) g.remove();
      [title, body] = (arg || "|").split("|");
      g = S("g", {}, root);
      const cw = vert ? 940 : 1160, ch = vert ? 560 : 420, cx = W / 2 - cw / 2, cy = vert ? H / 2 - 380 : H / 2 - ch / 2 - 20;
      RB(g, cx, cy, cw, ch, { fill: C.elev, stroke: C.bor, rx: 14 });
      S("rect", { x: cx, y: cy, width: 8, height: ch, fill: C.acc, rx: 3 }, g);
      T(g, cx + 60, cy + (vert ? 110 : 100), title, vert ? 52 : 54, { anchor: "start", w: 700 });
      L(g, cx + 60, cy + (vert ? 140 : 130), cx + 60 + Math.min(cw - 120, title.length * (vert ? 26 : 27)), cy + (vert ? 140 : 130), { stroke: C.acc, sw: 4 });
      const maxc = vert ? 40 : 56; const words = (body || "").split(" "); const lines = []; let cur = "";
      for (const w2 of words) { if ((cur + " " + w2).trim().length > maxc) { lines.push(cur.trim()); cur = w2; } else cur += " " + w2; }
      if (cur.trim()) lines.push(cur.trim());
      lines.forEach((ln, i) => T(g, cx + 60, cy + (vert ? 230 : 210) + i * (vert ? 58 : 56), ln, vert ? 38 : 38, { anchor: "start", fill: ln.includes("·") ? C.acc : C.text }));
    }
    let lastArg = null;
    return { update(vt, st) {
      if (st.arg !== lastArg) { lastArg = st.arg; build(st.arg); }
      op(g, fi(st.vv, 0.05, 0.4));
      g.setAttribute("transform", `translate(0,${(1 - fi(st.vv, 0.05, 0.5)) * 40})`);
    } };
  };

  // ---- transition bridge ----
  F.transit = (inst) => {
    const root = svgRoot();
    const [a, b] = (inst.variants[0].arg || "→").split("→").map(s => s.trim());
    const cwv = vert ? 420 : 560;
    const y = H / 2 - 60;
    const ga = S("g", {}, root), gb = S("g", {}, root);
    RB(ga, W / 2 - cwv - 100, y, cwv, 120, { fill: C.elev, stroke: C.bor, rx: 12 });
    T(ga, W / 2 - cwv / 2 - 100, y + 72, a, 40, { w: 700 });
    RB(gb, W / 2 + 100, y, cwv, 120, { fill: C.elev, stroke: C.acc, rx: 12 });
    T(gb, W / 2 + cwv / 2 + 100, y + 72, b, 40, { w: 700, fill: C.acc });
    const ar = L(root, W / 2 - 80, y + 60, W / 2 + 80, y + 60, { stroke: C.acc, sw: 5 });
    S("path", { d: `M ${W / 2 + 66} ${y + 46} L ${W / 2 + 92} ${y + 60} L ${W / 2 + 66} ${y + 74} Z`, fill: C.acc }, root);
    T(root, W / 2, y - 90, "UP NEXT", 30, { fill: C.mut, ls: 8 });
    return { update(vt) {
      op(ga, lerp(1, 0.45, fi(vt, 1.2, 1.5)));
      op(gb, fi(vt, 0.3, 0.8));
      op(ar, fi(vt, 0.15, 0.6));
    } };
  };

  // ---- pool: control channel + voice-channel pool (article SVG, animated) ----
  F.pool = (inst) => {
    const root = svgRoot();
    heading(root, "The channel pool", vert ? "one coordinator, channels on demand" : null);
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 230, tyV: 520 });
    // control channel bar
    RB(g, 100, 40, 1000, 90, { fill: C.acc, fo: 0.13, stroke: C.acc, rx: 12 });
    T(g, 600, 82, "control channel (data)", 34, { w: 700 });
    const ccMsg = T(g, 600, 118, '"TG 101 → channel 3"', 26, { fill: C.mut, mono: true, o: 0 });
    // voice channels (five — the script's "pool of five")
    const chs = [];
    for (let i = 0; i < 5; i++) {
      const x = 60 + i * 220;
      const gg = S("g", {}, g);
      const box = RB(gg, x, 420, 190, 110, { fill: C.text, fo: 0.0, stroke: C.bor, rx: 12, sw: 2.5 });
      T(gg, x + 95, 468, "voice " + (i + 1), 28, { fill: C.mut });
      const tg = T(gg, x + 95, 506, "", 25, { fill: C.acc, mono: true });
      chs.push({ gg, box, tg, x });
    }
    // grant arrow CC -> voice 3 (index 2)
    const arrow = L(g, 620, 140, 590, 400, { stroke: C.acc, sw: 4, dash: "10 8", o: 0 });
    const ah = S("path", { d: "M 573 386 L 588 414 L 604 384 Z", fill: C.acc, opacity: 0 }, g);
    // requesting radio
    const rq = radioIcon(g, 40, 480, 1.2, C.mut);
    const ping = S("circle", { cx: 52, cy: 440, r: 8, fill: C.acc, opacity: 0 }, g);
    const capt = T(g, 600, 640, "", 30, { fill: C.mut });
    const counter = T(g, 600, 690, "", 30, { fill: C.acc, mono: true, o: 0 });
    function setCh(i, on, label) {
      chs[i].box.setAttribute("fill-opacity", on ? 0.16 : 0);
      chs[i].box.setAttribute("fill", on ? C.acc : C.text);
      chs[i].box.setAttribute("stroke", on ? C.acc : C.bor);
      chs[i].tg.textContent = on ? (label || "TG 101") : "";
    }
    return { update(vt, st) {
      const v = st.variant, vv = st.vv;
      if (v === "grant") {
        capt.textContent = "a free channel is assigned per call";
        // ping rises to CC
        const p1 = fi(vv, 0.8, 1.0);
        op(ping, p1 > 0 && p1 < 1 ? 1 : 0);
        ping.setAttribute("cy", lerp(440, 140, p1));
        op(ccMsg, fi(vv, 2.0, 0.5));
        const p2 = fi(vv, 2.6, 0.9);
        op(arrow, p2); arrow.setAttribute("stroke-dashoffset", String((1 - p2) * 300));
        op(ah, fi(vv, 3.4, 0.3));
        setCh(2, vv > 3.5);
      } else if (v === "release") {
        setCh(2, vv < 1.2); op(arrow, clamp(1 - vv, 0, 1)); op(ah, clamp(1 - vv, 0, 1)); op(ccMsg, 0);
        capt.textContent = "…then released back to the pool";
      } else if (v === "stats") {
        capt.textContent = "5 channels · dozens of talkgroups";
        op(counter, fi(vv, 0.4));
        // deterministic random traffic
        let served = 0;
        for (let i = 0; i < 5; i++) {
          const cyc = 1.1 + rnd(i * 7 + 1) * 0.9;
          const ph = (vv + rnd(i + 3) * cyc * 2) / cyc;
          const on = (ph % 1) < 0.55;
          const tgn = 100 + Math.floor(rnd(i * 13 + Math.floor(ph)) * 80);
          setCh(i, on && vv > 0.3, "TG " + tgn);
        }
        served = Math.floor(clamp(vv, 0, 30) * 1.7);
        counter.textContent = "calls served: " + served;
      }
    } };
  };

  // ---- conventional vs trunked compare ----
  F["conv-vs-trunk"] = (inst) => {
    const root = svgRoot();
    heading(root, "Conventional vs. trunked");
    const g = canvasGroup(root, { vw: 1200, vh: 720, tyW: 220, tyV: 480 });
    const rows = ["Police disp.  → ch A", "Fire         → ch B", "Public works → ch C", "Transit      → ch D", "Events       → ch E", "…30 groups   → 30 ch"];
    const left = [], flick = [];
    const lx = vert ? 150 : 60, lw = vert ? 900 : 520;
    T(g, lx + lw / 2, 30, "conventional", 32, { w: 700, fill: C.mut });
    rows.forEach((r2, i) => {
      const y = 70 + i * 92;
      const gg = S("g", { opacity: 0 }, g);
      RB(gg, lx, y, lw, 70, { fill: C.elev, stroke: C.bor, rx: 10 });
      T(gg, lx + 30, y + 46, r2, 28, { anchor: "start", mono: true });
      const dot = S("circle", { cx: lx + lw - 40, cy: y + 35, r: 10, fill: C.good, opacity: 0.15 }, gg);
      left.push(gg); flick.push(dot);
    });
    const rx2 = vert ? 150 : 660, rw = vert ? 900 : 480;
    const ry = vert ? 660 : 70;
    const gr = S("g", { opacity: 0 }, g);
    if (vert) { /* stacked: pool goes below, drawn small */ }
    T(gr, rx2 + rw / 2, ry - 40 + (vert ? 20 : 0), "trunked", 32, { w: 700, fill: C.acc });
    RB(gr, rx2, ry, rw, 90, { fill: C.acc, fo: 0.12, stroke: C.acc, rx: 10 });
    T(gr, rx2 + rw / 2, ry + 55, "shared pool of 5 channels", 30, {});
    RB(gr, rx2, ry + 120, rw / 2 - 15, 80, { stroke: C.bor, rx: 10 });
    T(gr, rx2 + rw / 4 - 8, ry + 168, "any group", 26, { fill: C.mut });
    RB(gr, rx2 + rw / 2 + 15, ry + 120, rw / 2 - 15, 80, { stroke: C.bor, rx: 10 });
    T(gr, rx2 + 3 * rw / 4 + 8, ry + 168, "on demand", 26, { fill: C.mut });
    const note = T(g, vert ? 600 : 320, vert ? 620 : 660, "", 32, { fill: C.warn, mono: true, o: 0 });
    return { update(vt, st) {
      const v = st.variant, vv = st.vv;
      left.forEach((gg, i) => op(gg, fi(vt, 0.4 + i * 0.5, 0.5)));
      op(gr, v === "intro" ? fi(vt, 3.6, 0.8) * 0.35 : 0.35 + 0.65 * fi(vv, 0, 0.8));
      if (v === "idle") {
        note.textContent = "30 channels · ~4 busy right now"; op(note, fi(vv, 3.0, 0.8));
        left.forEach((gg, i) => {
          const cyc = 2.2 + rnd(i * 5) * 2.0;
          const on = ((vv + rnd(i * 11) * cyc) % cyc) < 0.5;
          gg.setAttribute("opacity", on ? 1 : 0.35);
          flick[i].setAttribute("opacity", on ? 0.9 : 0.12);
        });
      }
    } };
  };

  // ---- ccstream: the always-on control channel stripe ----
  F.ccstream = (inst) => {
    const root = svgRoot();
    heading(root, "The control channel", vert ? "continuous data — never voice" : null);
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 240, tyV: 500 });
    RB(g, 40, 120, 1120, 110, { fill: C.acc, fo: 0.1, stroke: C.acc, rx: 14 });
    if (!vert) T(g, 600, 90, "continuous data — never voice", 30, { fill: C.mut });
    // scrolling message blocks inside stripe (deterministic from t)
    const clip = S("clipPath", { id: "ccclip" }, g);
    S("rect", { x: 44, y: 124, width: 1112, height: 102, rx: 12 }, clip);
    const mg = S("g", { "clip-path": "url(#ccclip)" }, g);
    const kinds = [
      { k: "GRANT", c: C.acc, d: "TG 101 → CH 3" }, { k: "REG", c: C.good, d: "radio 4571" },
      { k: "AFFIL", c: C.warn, d: "4571 ⇒ TG 101" }, { k: "SYSINFO", c: C.mut, d: "site 1-2" },
      { k: "GRANT", c: C.acc, d: "TG 205 → CH 7" }, { k: "NEIGHBOR", c: C.mut, d: "3 sites" },
      { k: "UPDATE", c: C.deep, d: "TG 101 @ CH 3" }, { k: "GRANT", c: C.acc, d: "TG 310 → CH 2" },
    ];
    const blocks = kinds.map((kk, i) => {
      const gg = S("g", {}, mg);
      const bw = 200;
      RB(gg, 0, 138, bw, 74, { fill: C.elev, stroke: kk.c, rx: 8, sw: 2.5 });
      T(gg, bw / 2, 170, kk.k, 24, { fill: kk.c, w: 700, mono: true });
      T(gg, bw / 2, 200, kk.d, 20, { fill: C.mut, mono: true });
      return { gg, bw: 224 };
    });
    const lbl = T(g, 600, 300, "", 30, { fill: C.mut });
    // below-stripe area for roles / late entry
    const below = S("g", { opacity: 0 }, g);
    const callBar = RB(below, 140, 380, 0, 70, { fill: C.good, fo: 0.18, stroke: C.good, rx: 10 });
    T(below, 100, 425, "ch 3", 26, { fill: C.mut, mono: true });
    const lateR = radioIcon(below, 700, 560, 1.3, C.warn);
    const lateA = L(below, 700, 300, 700, 500, { stroke: C.warn, sw: 3, dash: "8 6", o: 0 });
    const lateT = T(below, 700, 640, "late entry — joins mid-call", 28, { fill: C.warn, o: 0 });
    const legend = T(g, 600, 660, "TSBK (P25) · CSBK (DMR) — short fixed-format blocks, back to back", 27, { fill: C.mut, mono: true, o: 0 });
    return { update(vt, st) {
      const v = st.variant, vv = st.vv;
      const speed = 120; // px/s scroll
      blocks.forEach((b, i) => {
        const total = blocks.reduce((s, x) => s + x.bw, 0);
        let x = 1160 - ((vt * speed + i * b.bw) % total);
        b.gg.setAttribute("transform", `translate(${x},0)`);
        op(b.gg, v === "intro" && vt < 1.5 ? fi(vt, 0.8, 0.7) : 1);
      });
      if (v === "intro") { lbl.textContent = "always on · never speaks"; op(lbl, fi(vv, 1.2, 0.8)); op(below, 0); op(legend, 0); }
      if (v === "messages") { lbl.textContent = "every message answers a radio's question"; op(lbl, 1); }
      if (v === "grantflow") { lbl.textContent = "identity · neighbors · status — between the grants"; op(lbl, 1); op(legend, fi(vv, 0.5, 0.8)); op(below, 0); }
      if (v === "lateentry") {
        op(legend, 0); lbl.textContent = "updates repeat while the call runs"; op(lbl, 1);
        op(below, fi(vv, 0.2, 0.6));
        callBar.setAttribute("width", clamp(vv * 260, 0, 920));
        const pj = fi(vv, 2.2, 0.8);
        op(lateA, pj); op(lateT, fi(vv, 2.8, 0.6));
        lateR.setAttribute("transform", `translate(700,${lerp(600, 540, pj)}) scale(1.3)`);
      }
    } };
  };

  // ---- ccpair: control vs voice channel roles ----
  F.ccpair = (inst) => {
    const root = svgRoot();
    heading(root, "Two kinds of channel");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 240, tyV: 500 });
    T(g, 60, 120, "control", 30, { anchor: "start", fill: C.acc, w: 700 });
    RB(g, 60, 150, 1080, 80, { fill: C.acc, fo: 0.12, stroke: C.acc, rx: 10 });
    T(g, 600, 200, "data · continuous · coordinates everything", 28, { fill: C.mut });
    T(g, 60, 330, "voice (traffic)", 30, { anchor: "start", fill: C.good, w: 700 });
    const vLine = L(g, 60, 440, 1140, 440, { stroke: C.bor, sw: 2 });
    const burst = RB(g, 260, 370, 0, 80, { fill: C.good, fo: 0.2, stroke: C.good, rx: 10 });
    const hang = RB(g, 0, 370, 0, 80, { fill: "none", stroke: C.warn, rx: 10, dash: "8 6" });
    const bl = T(g, 0, 420, "call", 26, { o: 0 });
    const hl = T(g, 0, 350, "hang time", 24, { fill: C.warn, o: 0 });
    const idle = T(g, 900, 420, "idle → back to pool", 26, { fill: C.mut, o: 0 });
    const cap = T(g, 600, 620, "borrowed for one call, then returned", 30, { fill: C.mut, o: 0 });
    return { update(vt, st) {
      const vv = st.vv;
      const w1 = clamp((vv - 1.0) * 180, 0, 420);
      burst.setAttribute("width", w1);
      bl.setAttribute("x", 260 + w1 / 2); op(bl, w1 > 60 ? 1 : 0);
      const hw = clamp((vv - 3.6) * 150, 0, 150);
      hang.setAttribute("x", 260 + w1); hang.setAttribute("width", hw);
      hl.setAttribute("x", 260 + w1 + hw / 2); op(hl, hw > 40 ? 1 : 0);
      op(idle, fi(vv, 5.2, 0.8)); op(cap, fi(vv, 5.6, 0.8));
    } };
  };

  // ---- tghop: talkgroup hopping across channels ----
  F.tghop = (inst) => {
    const root = svgRoot();
    heading(root, "Talkgroup 101 — one virtual channel");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 250, tyV: 500 });
    const rows = [{ n: "ch 2", y: 120 }, { n: "ch 3", y: 280 }, { n: "ch 7", y: 440 }];
    rows.forEach(r2 => { T(g, 50, r2.y + 55, r2.n, 28, { fill: C.mut, mono: true }); L(g, 110, r2.y + 45, 1150, r2.y + 45, { stroke: C.bor, sw: 2 }); });
    const calls = [
      { label: "call 1", row: 1, x: 160 }, { label: "call 2", row: 2, x: 520 }, { label: "call 3", row: 0, x: 880 },
    ].map((cc, i) => {
      const gg = S("g", { opacity: 0 }, g);
      RB(gg, cc.x, rows[cc.row].y, 260, 90, { fill: C.acc, fo: 0.16, stroke: C.acc, rx: 12 });
      T(gg, cc.x + 130, rows[cc.row].y + 40, cc.label, 28, { w: 700 });
      T(gg, cc.x + 130, rows[cc.row].y + 74, "TG 101", 24, { fill: C.acc, mono: true });
      return { gg, cc };
    });
    const ring = RB(g, 150, rows[1].y - 10, 280, 110, { stroke: C.warn, rx: 14, sw: 4, o: 0 });
    const foot = T(g, 600, 640, "you follow the group — the system moves it", 30, { fill: C.mut, o: 0 });
    return { update(vt, st) {
      const appear = [0.8, 2.0, 3.6]; // relative to instance for intro; follow keeps going
      const v = st.variant;
      const base = v === "intro" ? vt : 100; // in follow, all visible
      calls.forEach((c2, i) => {
        const on = v === "intro" ? (i === 0 ? fi(vt, 1.0, 0.6) : 0) : fi(st.vv, i === 0 ? 0 : (i === 1 ? 0.6 : 2.6), 0.6);
        op(c2.gg, on);
      });
      let target = 0;
      if (v === "follow") target = st.vv < 2.6 ? 1 : 2; else target = 0;
      const tc = calls[target].cc;
      ring.setAttribute("x", tc.x - 10); ring.setAttribute("y", rows[tc.row].y - 10);
      op(ring, v === "follow" ? 1 : fi(vt, 1.4, 0.6));
      op(foot, v === "follow" ? fi(st.vv, 3.4, 0.8) : 0);
    } };
  };

  // ---- tgprog: programming + affiliation ----
  F.tgprog = (inst) => {
    const root = svgRoot();
    heading(root, "Membership, not frequency");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 240, tyV: 500 });
    // radio with programmed list
    radioIcon(g, 120, 200, 2.2, C.mut);
    T(g, 120, 300, "your radio", 26, { fill: C.mut });
    const list = ["TG 101  Dispatch", "TG 205  Tac-2", "TG 310  Events"];
    const lg = S("g", { opacity: 0 }, g);
    RB(lg, 230, 110, 380, 200, { fill: C.elev, stroke: C.bor, rx: 12 });
    list.forEach((s2, i) => T(lg, 260, 160 + i * 55, s2, 26, { anchor: "start", mono: true, fill: i === 0 ? C.acc : C.text }));
    // keyup -> CC -> group retunes
    const cc = S("g", { opacity: 0 }, g);
    RB(cc, 700, 110, 440, 80, { fill: C.acc, fo: 0.12, stroke: C.acc, rx: 10 });
    T(cc, 920, 160, "control channel", 28, {});
    const arrowUp = L(g, 610, 150, 690, 150, { stroke: C.acc, sw: 4, dash: "8 6", o: 0 });
    const grp = S("g", { opacity: 0 }, g);
    for (let i = 0; i < 5; i++) radioIcon(grp, 720 + i * 90, 360, 1.1, C.acc);
    T(grp, 900, 440, "every affiliated radio retunes together", 26, { fill: C.mut });
    const down = L(g, 920, 200, 920, 320, { stroke: C.acc, sw: 4, dash: "8 6", o: 0 });
    return { update(vt) {
      op(lg, fi(vt, 0.5, 0.7)); op(arrowUp, fi(vt, 3.2, 0.6)); op(cc, fi(vt, 3.6, 0.6));
      op(down, fi(vt, 5.2, 0.6)); op(grp, fi(vt, 5.6, 0.8));
    } };
  };

  // ---- rid: talkgroup + radio id pair ----
  F.rid = (inst) => {
    const root = svgRoot();
    heading(root, "Two numbers per call");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 260, tyV: 520 });
    function chip(x, y, big, small, color) {
      const gg = S("g", { opacity: 0 }, g);
      RB(gg, x, y, vert ? 900 : 520, 220, { fill: C.elev, stroke: color, rx: 16, sw: 3 });
      T(gg, x + (vert ? 450 : 260), y + 90, big, 54, { w: 700, fill: color, mono: true });
      T(gg, x + (vert ? 450 : 260), y + 160, small, 30, { fill: C.mut });
      return gg;
    }
    const a = chip(vert ? 150 : 40, vert ? 60 : 160, "TG 101", "which group — the many", C.acc);
    const b = chip(vert ? 150 : 640, vert ? 340 : 160, "RID 4571", "which unit is talking — the one", C.good);
    const cap = T(g, 600, vert ? 660 : 520, "group identity + unit identity = every call, explained", 30, { fill: C.mut, o: 0 });
    return { update(vt) { op(a, fi(vt, 0.3, 0.6)); op(b, fi(vt, 1.6, 0.6)); op(cap, fi(vt, 3.2, 0.8)); } };
  };

  // ---- grantflow: CC -> grant capsule -> radios retune ----
  F.grantflow = (inst) => {
    const root = svgRoot();
    heading(root, "The grant");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 240, tyV: 500 });
    RB(g, 40, 140, 300, 100, { fill: C.acc, fo: 0.12, stroke: C.acc, rx: 12 });
    T(g, 190, 200, "control channel", 26, {});
    RB(g, 860, 140, 300, 100, { stroke: C.bor, rx: 12 });
    T(g, 1010, 188, "radios", 26, {});
    T(g, 1010, 220, "+ monitors", 22, { fill: C.mut });
    L(g, 350, 190, 850, 190, { stroke: C.bor, sw: 2 });
    const cap = S("g", { opacity: 0 }, g);
    RB(cap, -140, 152, 280, 76, { fill: C.elev, stroke: C.acc, rx: 38, sw: 2.5 });
    T(cap, 0, 185, "GRANT", 22, { fill: C.acc, w: 700, mono: true });
    T(cap, 0, 212, "TG 101 → CH 3", 20, { mono: true, fill: C.mut });
    // retune scene below
    const rg = S("g", { opacity: 0 }, g);
    const chbar = RB(rg, 300, 480, 600, 80, { fill: C.good, fo: 0.14, stroke: C.good, rx: 12 });
    T(rg, 600, 528, "voice channel 3", 26, {});
    const dots = [];
    for (let i = 0; i < 6; i++) {
      const d = S("circle", { cx: 150 + i * 180, cy: 330, r: 14, fill: i === 5 ? C.warn : C.acc }, rg);
      dots.push(d);
    }
    const monLbl = T(rg, 1120, 336, "← monitor", 22, { fill: C.warn, anchor: "start", o: vert ? 0 : 1 });
    const cap2 = T(g, 600, 660, "one message — the whole group moves at once", 30, { fill: C.mut, o: 0 });
    return { update(vt, st) {
      const v = st.variant, vv = st.vv;
      const p = fi(vt, 1.2, 1.6);
      op(cap, v === "retune" ? (1 - fi(vv, 0.4, 0.6)) : fi(vt, 0.8, 0.4));
      cap.setAttribute("transform", `translate(${lerp(220, 990, p)},0)`);
      if (v === "retune") {
        op(monLbl, vert ? 0 : 1 - fi(vv, 0.4, 0.6));
        op(rg, fi(vv, 0.1, 0.6));
        dots.forEach((d, i) => {
          const pp = fi(vv, 0.5 + i * 0.12, 0.7);
          d.setAttribute("cy", lerp(330, 465, pp));
          d.setAttribute("cx", lerp(150 + i * 180, 380 + i * 90, pp));
        });
        op(cap2, fi(vv, 2.2, 0.8));
      }
    } };
  };

  // ---- grantfields: anatomy of the message ----
  F.grantfields = (inst) => {
    const root = svgRoot();
    heading(root, "Inside a grant");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 240, tyV: 480 });
    RB(g, 300, 40, 600, 70, { fill: C.elev, stroke: C.acc, rx: 35 });
    T(g, 600, 84, "GRANT: TG 101 → CH 0x0A3 · slot 2", 26, { mono: true });
    const fields = [
      { t: "TARGET", d: ["talkgroup 101 (group call)", "or src+dst pair (private)"], c: C.acc },
      { t: "CHANNEL", d: ["number 0x0A3 → channel plan", "→ 853.9875 MHz"], c: C.good },
      { t: "TIMESLOT", d: ["slot 2 — TDMA only", "(two calls per frequency)"], c: C.warn },
    ].map((f, i) => {
      const x = vert ? 140 : 40 + i * 400, y = vert ? 180 + i * 175 : 220, w2 = vert ? 920 : 360;
      const gg = S("g", { opacity: 0 }, g);
      L(gg, 600, 115, vert ? 600 : x + w2 / 2, y - 10, { stroke: f.c, sw: 2, dash: "6 5" });
      RB(gg, x, y, w2, 150, { fill: C.elev, stroke: f.c, rx: 12, sw: 3 });
      T(gg, x + w2 / 2, y + 45, f.t, 28, { fill: f.c, w: 700, mono: true });
      f.d.forEach((dd, j) => T(gg, x + w2 / 2, y + 85 + j * 34, dd, 23, { fill: C.mut }));
      return gg;
    });
    const cap = T(g, 600, vert ? 480 : 470, "", 30, { fill: C.mut, o: 0 });
    return { update(vt) { fields.forEach((f, i) => op(f, fi(vt, 1.0 + i * 4.0, 0.7))); } };
  };

  // ---- grantupdate: repeats + late entry ----
  F.grantupdate = (inst) => {
    const root = svgRoot();
    heading(root, "Grant updates & late entry");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 250, tyV: 500 });
    L(g, 60, 400, 1150, 400, { stroke: C.bor, sw: 2 });
    T(g, 1120, 440, "time →", 24, { fill: C.mut });
    const call = RB(g, 160, 300, 0, 70, { fill: C.good, fo: 0.15, stroke: C.good, rx: 10 });
    T(g, 90, 345, "call", 26, { fill: C.mut });
    const gr = S("g", { opacity: 0 }, g);
    RB(gr, 120, 150, 120, 60, { fill: C.elev, stroke: C.acc, rx: 8, sw: 2.5 });
    T(gr, 180, 188, "GRANT", 22, { fill: C.acc, mono: true });
    const ups = [];
    for (let i = 0; i < 4; i++) {
      const x = 330 + i * 210;
      const gg = S("g", { opacity: 0 }, g);
      RB(gg, x, 150, 140, 60, { fill: C.elev, stroke: C.deep, rx: 8, sw: 2.5 });
      T(gg, x + 70, 188, "UPDATE", 22, { fill: C.acc, mono: true, o: 0.8 });
      L(gg, x + 70, 212, x + 70, 290, { stroke: C.deep, sw: 2, dash: "6 5" });
      ups.push({ gg, x });
    }
    const late = radioIcon(g, 760, 520, 1.3, C.warn);
    op(late, 0);
    const lateT = T(g, 760, 600, "late arrival — reads an update, joins mid-call", 27, { fill: C.warn, o: 0 });
    return { update(vt) {
      op(gr, fi(vt, 0.4, 0.5));
      call.setAttribute("width", clamp((vt - 0.8) * 130, 0, 930));
      ups.forEach((u, i) => op(u.gg, fi(vt, 1.6 + i * 1.5, 0.5)));
      const pj = fi(vt, 6.5, 0.8);
      op(late, pj); late.setAttribute("transform", `translate(760,${lerp(560, 480, pj)}) scale(1.3)`);
      op(lateT, fi(vt, 7.2, 0.7));
    } };
  };

  // ---- grantqueue: busy system ----
  F.grantqueue = (inst) => {
    const root = svgRoot();
    heading(root, "When every channel is busy");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 250, tyV: 520 });
    const chs = [];
    for (let i = 0; i < 4; i++) {
      const x = 100 + i * 260;
      const box = RB(g, x, 200, 220, 110, { fill: C.bad, fo: 0.14, stroke: C.bad, rx: 12 });
      T(g, x + 110, 248, "voice " + (i + 1), 28, { fill: C.mut });
      const s2 = T(g, x + 110, 286, "busy", 24, { fill: C.bad, mono: true });
      chs.push({ box, s2 });
    }
    const q = S("circle", { cx: 600, cy: 480, r: 16, fill: C.warn }, g);
    const ql = T(g, 600, 540, "request queued…", 28, { fill: C.warn });
    const done = T(g, 600, 620, "channel frees → grant arrives seconds later", 28, { fill: C.mut, o: 0 });
    return { update(vt) {
      const freeAt = 4.0;
      const blink = (Math.sin(vt * 5) + 1) / 2;
      q.setAttribute("opacity", vt < freeAt ? 0.4 + 0.6 * blink : 1);
      if (vt > freeAt) {
        chs[1].box.setAttribute("stroke", C.good); chs[1].box.setAttribute("fill", C.good);
        chs[1].s2.textContent = "TG 101"; chs[1].s2.setAttribute("fill", C.good);
        const p = fi(vt, freeAt + 0.4, 1.0);
        q.setAttribute("cx", lerp(600, 470, p)); q.setAttribute("cy", lerp(480, 255, p));
        ql.textContent = "granted!"; ql.setAttribute("fill", C.good);
        op(done, fi(vt, freeAt + 1.4, 0.7));
      }
    } };
  };

  // ---- fdmastack ----
  F.fdmastack = (inst) => {
    const root = svgRoot();
    heading(root, "FDMA — one call per frequency");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 240, tyV: 500 });
    L(g, 80, 620, 80, 60, { stroke: C.mut, sw: 3 });
    S("path", { d: "M 70 74 L 80 50 L 90 74 Z", fill: C.mut }, g);
    const fl = T(g, 30, 340, "frequency", 26, { fill: C.mut, o: 1 });
    fl.setAttribute("transform", "rotate(-90 30 340)");
    const bars = [], labels = ["call A", "call B", "call C", "call D"];
    for (let i = 0; i < 4; i++) {
      const gg = S("g", { opacity: 0 }, g);
      const y = 90 + i * 135;
      const b = RB(gg, 130, y, 900, 105, { fill: C.acc, fo: 0.15, stroke: C.acc, rx: 8 });
      const t2 = T(gg, 580, y + 62, labels[i], 30, {});
      bars.push({ gg, b, t2, y });
    }
    const guard = T(g, 1090, 240, "guard band", 22, { fill: C.warn, o: 0 });
    const nb = S("g", { opacity: 0 }, g);
    const nbl = T(nb, 580, 660, "25 kHz FM → 12.5 kHz P25 Phase 1 → 6.25 kHz NXDN", 28, { fill: C.acc, mono: true });
    return { update(vt, st) {
      const v = st.variant, vv = st.vv;
      bars.forEach((b, i) => op(b.gg, fi(vt, 0.6 + i * 0.8, 0.6)));
      op(guard, fi(vt, 4.2, 0.8));
      if (v === "narrowband") {
        op(nb, fi(vv, 0.3, 0.6));
        const shrink = ez(clamp((vv - 1.0) / 2.5, 0, 1)); // 105 -> 48
        bars.forEach((b, i) => {
          const h = lerp(105, 46, shrink);
          b.b.setAttribute("height", h);
          b.b.setAttribute("y", 90 + i * lerp(135, 62, shrink));
          b.t2.setAttribute("y", 90 + i * lerp(135, 62, shrink) + h / 2 + 10);
          b.t2.setAttribute("font-size", lerp(30, 22, shrink));
        });
        // extra channels appear as they narrow
        for (let i = 4; i < 8; i++) {
          if (!bars[i] && vv > 2.6) {
            const gg = S("g", {}, g);
            const y = 90 + i * 62;
            const b = RB(gg, 130, y, 900, 46, { fill: C.good, fo: 0.14, stroke: C.good, rx: 6 });
            const t2 = T(gg, 580, y + 32, "call " + "EFGH"[i - 4], 22, {});
            bars.push({ gg, b, t2, y });
          }
          if (bars[i]) op(bars[i].gg, fi(vv, 2.6 + (i - 4) * 0.3, 0.5));
        }
      }
    } };
  };

  // ---- tdmaslots ----
  F.tdmaslots = (inst) => {
    const root = svgRoot();
    heading(root, "TDMA — calls take turns in time");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 240, tyV: 500 });
    L(g, 60, 420, 1150, 420, { stroke: C.mut, sw: 3 });
    S("path", { d: "M 1126 410 L 1150 420 L 1126 430 Z", fill: C.mut }, g);
    T(g, 600, 460, "time →", 26, { fill: C.mut });
    T(g, 600, 90, "one frequency — two calls share the slots", 30, { fill: C.mut });
    const slots = [];
    for (let i = 0; i < 8; i++) {
      const x = 100 + i * 128;
      const one = i % 2 === 0;
      const gg = S("g", { opacity: 0 }, g);
      RB(gg, x, 180, 112, 200, { fill: one ? C.acc : C.good, fo: one ? 0.28 : 0.16, stroke: one ? C.acc : C.good, rx: 8 });
      T(gg, x + 56, 292, one ? "1" : "2", 44, { w: 700, fill: one ? C.acc : C.good });
      slots.push({ gg, x });
    }
    const guard = S("g", { opacity: 0 }, g);
    for (let i = 1; i < 8; i++) RB(guard, 100 + i * 128 - 16, 180, 16, 200, { fill: C.warn, fo: 0.25, rx: 2 });
    T(guard, 600, 160, "guard time between bursts", 24, { fill: C.warn });
    const sync = S("g", { opacity: 0 }, g);
    for (let i = 0; i < 8; i += 2) L(sync, 100 + i * 128 + 56, 390, 100 + i * 128 + 56, 410, { stroke: C.soft, sw: 4 });
    T(sync, 600, 560, "radios lock to a frame-sync pattern to find the slot boundaries", 26, { fill: C.mut });
    const chips = S("g", { opacity: 0 }, g);
    [["DMR ×2", C.acc], ["P25 Phase 2 ×2", C.acc], ["TETRA ×4", C.good]].forEach((cc, i) => {
      const x = vert ? 210 + i * 280 : 260 + i * 260;
      RB(chips, x, 610, vert ? 260 : 230, 64, { fill: C.elev, stroke: cc[1], rx: 32 });
      T(chips, x + (vert ? 130 : 115), 650, cc[0], 24, { fill: cc[1], mono: true });
    });
    return { update(vt, st) {
      const v = st.variant, vv = st.vv;
      slots.forEach((s2, i) => op(s2.gg, fi(vt, 0.6 + i * 0.35, 0.4)));
      if (v === "timing") {
        op(guard, fi(vv, 0.4, 0.7)); op(sync, fi(vv, 2.2, 0.7)); op(chips, fi(vv, 5.0, 0.7));
      }
    } };
  };

  // ---- accesscompare ----
  F.accesscompare = (inst) => {
    const root = svgRoot();
    heading(root, "FDMA + TDMA, together");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 250, tyV: 500 });
    function mini(x, y, w2, tdma) {
      const gg = S("g", { opacity: 0 }, g);
      for (let i = 0; i < 3; i++) {
        RB(gg, x, y + i * 64, w2, 52, { fill: C.acc, fo: 0.12, stroke: C.acc, rx: 6 });
        if (tdma) for (let j2 = 0; j2 < 4; j2++)
          RB(gg, x + 8 + j2 * (w2 / 4 - 6), y + i * 64 + 8, w2 / 4 - 14, 36, { fill: j2 % 2 ? C.good : C.acc, fo: 0.3, rx: 4 });
      }
      return gg;
    }
    const lx = vert ? 90 : 80, rx2 = vert ? 90 : 660, w2 = vert ? 900 : 460;
    const ly = vert ? 60 : 100, ry = vert ? 400 : 100;
    T(g, lx + w2 / 2, ly - 20, "FDMA channelizes the band", 28, { fill: C.mut });
    const a = mini(lx, ly, w2, false);
    T(g, rx2 + w2 / 2, ry - 20, "TDMA time-shares each channel", 28, { fill: C.mut });
    const b = mini(rx2, ry, w2, true);
    const ga = S("g", { opacity: 0 }, g), gb = S("g", { opacity: 0 }, g);
    RB(ga, lx + w2 / 2 - 210, ly + 220, 420, 62, { fill: C.elev, stroke: C.acc, rx: 31 });
    T(ga, lx + w2 / 2, ly + 260, "GRANT: frequency", 24, { mono: true, fill: C.acc });
    RB(gb, rx2 + w2 / 2 - 250, ry + 220, 500, 62, { fill: C.elev, stroke: C.good, rx: 31 });
    T(gb, rx2 + w2 / 2, ry + 260, "GRANT: frequency + slot", 24, { mono: true, fill: C.good });
    return { update(vt) { op(a, fi(vt, 0.3, 0.6)); op(b, fi(vt, 1.4, 0.6)); op(ga, fi(vt, 4.4, 0.6)); op(gb, fi(vt, 5.2, 0.6)); } };
  };

  // ---- gt-tiein: GopherTrunk UI vignette ----
  F["gt-tiein"] = (inst) => {
    const root = svgRoot();
    heading(root, "In GopherTrunk");
    const g = canvasGroup(root, { vw: 1200, vh: 700, tyW: 230, tyV: 480 });
    RB(g, 60, 40, 1080, 620, { fill: C.elev, stroke: C.bor, rx: 16 });
    if (R.logo) S("image", { x: 90, y: 66, width: 44, height: 44, href: R.logo }, g);
    T(g, 150, 96, "GopherTrunk — live", 28, { anchor: "start", w: 700 });
    L(g, 60, 130, 1140, 130, { stroke: C.bor, sw: 2 });
    const cc = S("g", { opacity: 0 }, g);
    RB(cc, 90, 155, 1020, 64, { fill: C.acc, fo: 0.1, stroke: C.acc, rx: 8 });
    T(cc, 110, 196, "CONTROL  853.0125 MHz", 25, { anchor: "start", mono: true });
    T(cc, 1090, 196, "LOCKED ●", 25, { anchor: "end", mono: true, fill: C.good });
    const rowsDef = [
      ["TG 101  Police Dispatch", "RID 4571", "CH 3"],
      ["TG 310  Events", "RID 8823", "CH 7 · s2"],
      ["TG 205  Fire Tac-2", "RID 1054", "CH 2"],
    ];
    const rows = rowsDef.map((r2, i) => {
      const y = 245 + i * 92;
      const gg = S("g", { opacity: 0 }, g);
      RB(gg, 90, y, 1020, 74, { fill: C.bg, stroke: C.bor, rx: 8 });
      S("circle", { cx: 122, cy: y + 37, r: 9, fill: C.good }, gg);
      T(gg, 150, y + 46, r2[0], 25, { anchor: "start", mono: true });
      T(gg, 800, y + 46, r2[1], 24, { anchor: "start", mono: true, fill: C.mut });
      T(gg, 1090, y + 46, r2[2], 24, { anchor: "end", mono: true, fill: C.acc });
      // little VU bars
      for (let j2 = 0; j2 < 5; j2++) {
        const rb = S("rect", { x: 700 + j2 * 12, y: y + 50, width: 8, height: 6, fill: C.acc, opacity: 0.8 }, gg);
        rb.dataset.j = j2; rb.dataset.i = i;
      }
      return gg;
    });
    const foot = T(g, 600, 610, "one SDR · every conversation, tagged by talkgroup", 27, { fill: C.mut, o: 0 });
    return { update(vt) {
      op(cc, fi(vt, 0.4, 0.6));
      rows.forEach((r2, i) => {
        op(r2, fi(vt, 1.4 + i * 1.6, 0.6));
        r2.querySelectorAll("rect[data-j]").forEach(rb => {
          const j2 = +rb.dataset.j, i2 = +rb.dataset.i;
          const h = 5 + 22 * Math.abs(Math.sin(vt * (2.1 + i2 * 0.7) + j2 * 1.1)) * rnd(i2 * 9 + j2);
          rb.setAttribute("height", h); rb.setAttribute("y", 245 + i2 * 92 + 56 - h);
        });
      });
      op(foot, fi(vt, 6.0, 0.8));
    } };
  };

  // ---- cold open ----
  F.coldopen = (inst) => {
    const root = svgRoot();
    let g = null, mode = null;
    function build(v) {
      if (g) g.remove(); g = S("g", {}, root); mode = v;
      if (v === "city") {
        towerIcon(g, W / 2, H / 2 - 40, 3.2, C.acc);
        for (let i = 0; i < 3; i++) S("circle", { cx: W / 2, cy: H / 2 - 180, r: 60, fill: "none", stroke: C.acc, "stroke-width": 3, opacity: 0, "data-ring": i }, g);
        for (let i = 0; i < 8; i++) {
          const a = (i / 8) * Math.PI * 2 + 0.4;
          const r2 = vert ? 380 : 520;
          radioIcon(g, W / 2 + Math.cos(a) * r2, H / 2 + 60 + Math.sin(a) * (vert ? 420 : 260), 1.0, C.mut);
        }
        T(g, W / 2, H - (vert ? 420 : 200), "one dispatcher · one hundred radios", 40, { fill: C.mut });
      } else { // title
        S("rect", { x: 0, y: 0, width: W, height: 6, fill: C.acc }, g);
        T(g, W / 2, H / 2 - 150, "THE GOPHERTRUNK FIELD GUIDE", 32, { fill: C.mut, ls: 8 });
        T(g, W / 2, H / 2 - 30, "TRUNKED RADIO", vert ? 96 : 130, { w: 700 });
        T(g, W / 2, H / 2 + 60, "five ideas · one course", 40, { fill: C.acc });
        if (R.logo) S("image", { x: W / 2 - 40, y: H / 2 + 120, width: 80, height: 80, href: R.logo }, g);
      }
    }
    return { update(vt, st) {
      if (st.variant !== mode) build(st.variant);
      if (mode === "city") {
        g.querySelectorAll("circle[data-ring]").forEach((c2) => {
          const i = +c2.dataset.ring;
          const p = ((st.vv * 0.5 + i / 3) % 1);
          c2.setAttribute("r", 60 + p * 380);
          c2.setAttribute("opacity", (1 - p) * 0.5);
        });
      } else op(g, fi(st.vv, 0.05, 0.6));
    } };
  };

  // ---- intro cards + agenda ----
  F.introcard = (inst) => {
    const root = svgRoot();
    let g = null, last = null;
    const msgs = {
      welcome: ["GOPHERTRUNK FIELD GUIDE — VIDEO EDITION", "The trunked radio pillar", "many users · a small pool of channels"],
      standalone: ["EVERY SEGMENT STANDS ALONE", "Jump to the idea you need", "full articles at gophertrunk.org/reference/"],
      start: ["FIRST STOP", "Too many groups, too few frequencies", "let's fix that"],
    };
    return { update(vt, st) {
      if (st.variant !== last) {
        last = st.variant; if (g) g.remove(); g = S("g", {}, root);
        const m = msgs[st.variant] || msgs.welcome;
        T(g, W / 2, H / 2 - 120, m[0], 30, { fill: C.mut, ls: 6 });
        T(g, W / 2, H / 2 - 10, m[1], vert ? 60 : 72, { w: 700 });
        T(g, W / 2, H / 2 + 80, m[2], 36, { fill: C.acc });
        L(g, W / 2 - 130, H / 2 + 130, W / 2 + 130, H / 2 + 130, { stroke: C.acc, sw: 4 });
      }
      op(g, fi(st.vv, 0.05, 0.5));
    } };
  };
  F.agenda = (inst) => {
    const root = svgRoot();
    T(root, W / 2, vert ? 300 : 140, "THE ROUTE", 32, { fill: C.mut, ls: 8 });
    const items = ["Trunked radio", "Control channel", "Talkgroup", "Channel grant", "FDMA & TDMA"].map((s2, i) => {
      const y = (vert ? 420 : 230) + i * (vert ? 220 : 150);
      const g = S("g", { opacity: 0 }, root);
      const bw = vert ? 820 : 900;
      RB(g, W / 2 - bw / 2, y, bw, vert ? 160 : 110, { fill: C.elev, stroke: C.bor, rx: 12 });
      T(g, W / 2 - bw / 2 + 70, y + (vert ? 100 : 72), String(i + 1), 44, { fill: C.acc, w: 700, mono: true });
      T(g, W / 2 - bw / 2 + 130, y + (vert ? 100 : 72), s2, vert ? 48 : 44, { anchor: "start", w: 700 });
      return g;
    });
    return { update(vt) { items.forEach((g, i) => op(g, fi(vt, 1.0 + i * 3.4, 0.6))); } };
  };

  // ---- outro ----
  F.outrosum = (inst) => {
    const root = svgRoot();
    const ideas = ["pool", "control channel", "talkgroup", "grant", "FDMA/TDMA"];
    const chips = ideas.map((s2, i) => {
      const n = ideas.length;
      const x = vert ? W / 2 : W / 2 + (i - 2) * 330;
      const y = vert ? 420 + i * 230 : H / 2 - 80;
      const g = S("g", { opacity: 0 }, root);
      RB(g, x - 140, y - 45, 280, 90, { fill: C.elev, stroke: C.acc, rx: 45, sw: 2.5 });
      T(g, x, y + 10, s2, 28, { w: 700 });
      return { g, x, y };
    });
    const links = S("g", { opacity: 0 }, root);
    for (let i = 0; i < chips.length - 1; i++)
      L(links, vert ? chips[i].x : chips[i].x + 140, vert ? chips[i].y + 45 : chips[i].y,
        vert ? chips[i + 1].x : chips[i + 1].x - 140, vert ? chips[i + 1].y - 45 : chips[i + 1].y,
        { stroke: C.acc, sw: 3, dash: "8 6" });
    const cta = S("g", { opacity: 0 }, root);
    T(cta, W / 2, H / 2 + (vert ? 640 : 200), "Field Guide → gophertrunk.org", 40, { fill: C.acc, mono: true });
    T(cta, W / 2, H / 2 + (vert ? 700 : 260), "GopherTrunk (open source) → GitHub", 34, { fill: C.mut, mono: true });
    return { update(vt, st) {
      const v = st.variant;
      chips.forEach((c2, i) => op(c2.g, fi(vt, 0.6 + i * 1.6, 0.6)));
      op(links, v !== "recap" ? fi(st.vv, v === "apply" ? 0.5 : 0, 0.8) : 0);
      op(cta, v === "cta" ? fi(st.vv, 0.4, 0.7) : 0);
    } };
  };
  F.endslate = (inst) => {
    const root = svgRoot();
    if (R.logo) S("image", { x: W / 2 - 70, y: H / 2 - 300, width: 140, height: 140, href: R.logo }, root);
    T(root, W / 2, H / 2 - 60, "Full course on YouTube", 44, { fill: C.mut });
    T(root, W / 2, H / 2 + 30, "GopherTrunk", 84, { w: 700 });
    T(root, W / 2, H / 2 + 120, "gophertrunk.org", 40, { fill: C.acc, mono: true });
    S("rect", { x: 0, y: H - 6, width: W, height: 6, fill: C.acc }, root);
    return { update() {} };
  };

  // ============================================================
  // Timeline plumbing
  // ============================================================
  const TL = R.timeline;
  // group events into blocks, blocks into instances
  const blocks = [];
  for (const ev of TL.events) {
    let blk = blocks[blocks.length - 1];
    if (!blk || blk.b !== ev.b) {
      blk = { b: ev.b, visual: ev.visual, variant: ev.variant || "", arg: ev.arg || "", t0: ev.t, t1: ev.t + ev.dur, events: [] };
      blocks.push(blk);
    }
    blk.t1 = ev.t + ev.dur; blk.events.push(ev);
  }
  const instances = [];
  for (const blk of blocks) {
    let inst = instances[instances.length - 1];
    if (!inst || inst.name !== blk.visual) {
      inst = { name: blk.visual, t0: blk.t0, t1: blk.t1, variants: [] };
      instances.push(inst);
    }
    inst.t1 = blk.t1;
    inst.variants.push({ variant: blk.variant, arg: blk.arg, t0: blk.t0 });
  }

  // chrome: corner bug (wide) + captions (vert)
  let bug = null;
  if (!vert && R.logo) {
    bug = document.createElement("img");
    bug.src = R.logo;
    Object.assign(bug.style, { position: "absolute", right: "34px", top: "28px", width: "64px", opacity: 0.3, zIndex: 50 });
    stage.appendChild(bug);
  }
  let capBox = null, capLine1 = null, capLine2 = null;
  if (R.captions) {
    capBox = document.createElement("div");
    Object.assign(capBox.style, {
      position: "absolute", left: "50%", transform: "translateX(-50%)",
      top: Math.round(H * 0.70) + "px", width: "920px", padding: "16px 26px",
      background: "rgba(13,17,23,.85)", borderRadius: "16px", textAlign: "center",
      font: `600 34px ${SANS}`, color: "#fff", lineHeight: "1.35", zIndex: 60,
      border: "1px solid rgba(48,54,61,.8)",
    });
    stage.appendChild(capBox);
  }
  // caption chunking: <=76-char chunks; the browser wraps them to <=2 lines
  function chunks(text) {
    const words = text.split(" "); const out = []; let cur = "";
    for (const w2 of words) { if ((cur + " " + w2).trim().length > 76) { out.push(cur.trim()); cur = w2; } else cur += " " + w2; }
    if (cur.trim()) out.push(cur.trim());
    return out.map(c => [c]);
  }

  let mounted = null, comp = null;
  window.seek = function (t) {
    // instance lookup
    let inst = instances[0];
    for (const it of instances) if (it.t0 <= t + 1e-6) inst = it; else break;
    if (inst !== mounted) {
      stage.querySelectorAll("svg").forEach(e => e.remove());
      mounted = inst;
      comp = (F[inst.name] || F.introcard)(inst, {});
      if (bug) bug.style.display = (inst.name === "title" || inst.name === "endslate" || inst.name === "hook") ? "none" : "block";
    }
    let va = inst.variants[0];
    for (const v of inst.variants) if (v.t0 <= t + 1e-6) va = v; else break;
    comp.update(t - inst.t0, { variant: va.variant, arg: va.arg, vv: t - va.t0, t });
    // captions
    if (capBox) {
      let ev = null;
      for (const e of TL.events) if (e.text && e.t <= t && t < e.t + e.dur + 0.15) { ev = e; break; }
      if (ev) {
        const ch = chunks(ev.text);
        const idx = Math.min(ch.length - 1, Math.floor(((t - ev.t) / ev.dur) * ch.length));
        capBox.innerHTML = ch[idx].map(l2 => l2.replace(/&/g, "&amp;").replace(/</g, "&lt;")).join("<br>");
        capBox.style.display = "block";
      } else capBox.style.display = "none";
    }
  };
  window.seek(0);
  window.ready = true;
})();
