/*
 * labcharts.js — self-contained canvas charts for the Lab Bench blog series
 * (Signal Lab, RF Scope, Crypto Lab). No CDN, no dependencies, no build step —
 * the same bespoke, vendored style as site.js / learn.js / autolink.js.
 *
 * Usage in a post (kramdown passes raw HTML through when it starts at column 0):
 *
 *   <figure class="lab-figure">
 *     <canvas class="lab-chart" data-chart="constellation" width="480" height="360"
 *             role="img" aria-label="C4FM constellation, four rails"></canvas>
 *     <script type="application/json" class="lab-chart-data">
 *       { "points": [[1,0.02],[0.98,-0.3], ...], "rails": [-3,-1,1,3] }
 *     </script>
 *     <figcaption>A clean C4FM constellation: four tight vertical rails.</figcaption>
 *   </figure>
 *
 * Each <canvas class="lab-chart"> is paired with the JSON in its *next sibling*
 * <script class="lab-chart-data">. data-chart selects the renderer. The module
 * no-ops on any page with no .lab-chart elements, is high-DPI aware, and redraws
 * when the color theme (:root.dark) is toggled or the canvas is resized.
 *
 * Chart types: constellation | eye | heatmap | line | bars | timeline
 */
(function () {
  "use strict";

  function cssVar(name, fallback) {
    var v = getComputedStyle(document.documentElement).getPropertyValue(name);
    return (v && v.trim()) || fallback;
  }

  // Theme-derived palette, re-read on every draw so dark-mode toggles are live.
  function palette() {
    return {
      fg: cssVar("--fg", "#1a202c"),
      muted: cssVar("--fg-muted", "#5b6470"),
      border: cssVar("--border", "#e2e8f0"),
      accent: cssVar("--accent", "#155799"),
      bg: cssVar("--bg", "#ffffff"),
      elev: cssVar("--bg-elev", "#f6f8fa"),
      // A small categorical ramp for multi-series charts (color-blind safe-ish).
      series: ["#155799", "#2f9e44", "#e8590c", "#9c36b5", "#0c8599", "#c92a2a"],
      good: "#2f9e44",
      warn: "#e8590c",
      bad: "#c92a2a"
    };
  }

  function setupHiDPI(canvas) {
    var ctx = canvas.getContext("2d");
    var ratio = window.devicePixelRatio || 1;
    // The author-declared width/height attributes are the CSS logical size.
    var cssW = canvas.getAttribute("width") ? parseInt(canvas.getAttribute("width"), 10) : canvas.clientWidth;
    var cssH = canvas.getAttribute("height") ? parseInt(canvas.getAttribute("height"), 10) : canvas.clientHeight;
    canvas.style.width = cssW + "px";
    canvas.style.height = cssH + "px";
    canvas.width = Math.round(cssW * ratio);
    canvas.height = Math.round(cssH * ratio);
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    return { ctx: ctx, w: cssW, h: cssH };
  }

  // ---- shared axis helper ------------------------------------------------
  function axes(ctx, box, p, opts) {
    opts = opts || {};
    ctx.strokeStyle = p.border;
    ctx.fillStyle = p.muted;
    ctx.lineWidth = 1;
    ctx.font = '11px ' + cssVar("--font-sans", "sans-serif");
    // frame
    ctx.strokeRect(box.x, box.y, box.w, box.h);
    if (opts.xlabel) {
      ctx.textAlign = "center";
      ctx.fillText(opts.xlabel, box.x + box.w / 2, box.y + box.h + 26);
    }
    if (opts.ylabel) {
      ctx.save();
      ctx.translate(box.x - 34, box.y + box.h / 2);
      ctx.rotate(-Math.PI / 2);
      ctx.textAlign = "center";
      ctx.fillText(opts.ylabel, 0, 0);
      ctx.restore();
    }
  }

  function plotBox(w, h) {
    return { x: 46, y: 14, w: w - 62, h: h - 44 };
  }

  // ---- renderers ---------------------------------------------------------
  var renderers = {
    constellation: function (ctx, w, h, d, p) {
      var box = { x: 20, y: 14, w: w - 40, h: h - 40 };
      // scale so ±rails fit with margin
      var lim = 1.6;
      if (d.rails && d.rails.length) lim = Math.max.apply(null, d.rails.map(Math.abs)) * 1.35;
      function X(v) { return box.x + (v / lim / 2 + 0.5) * box.w; }
      function Y(v) { return box.y + (0.5 - v / lim / 2) * box.h; }
      // crosshair axes
      ctx.strokeStyle = p.border;
      ctx.lineWidth = 1;
      ctx.beginPath(); ctx.moveTo(box.x, Y(0)); ctx.lineTo(box.x + box.w, Y(0)); ctx.stroke();
      ctx.beginPath(); ctx.moveTo(X(0), box.y); ctx.lineTo(X(0), box.y + box.h); ctx.stroke();
      // rail guides (vertical, for C4FM 4-level style)
      if (d.rails) {
        ctx.strokeStyle = p.accent;
        ctx.globalAlpha = 0.18;
        d.rails.forEach(function (r) {
          ctx.beginPath(); ctx.moveTo(X(r), box.y); ctx.lineTo(X(r), box.y + box.h); ctx.stroke();
        });
        ctx.globalAlpha = 1;
      }
      // points
      ctx.fillStyle = d.color || p.accent;
      (d.points || []).forEach(function (pt) {
        ctx.beginPath();
        ctx.arc(X(pt[0]), Y(pt[1]), d.dot || 2.2, 0, Math.PI * 2);
        ctx.fill();
      });
    },

    eye: function (ctx, w, h, d, p) {
      var box = plotBox(w, h);
      axes(ctx, box, p, { xlabel: d.xlabel || "symbol time (2 UI)", ylabel: d.ylabel || "amplitude" });
      var traces = d.traces || [];
      var span = d.span || 2;
      ctx.lineWidth = 1;
      ctx.strokeStyle = d.color || p.accent;
      ctx.globalAlpha = d.alpha || 0.5;
      traces.forEach(function (tr) {
        ctx.beginPath();
        tr.forEach(function (v, i) {
          var x = box.x + (i / (tr.length - 1)) * box.w;
          var y = box.y + (0.5 - v / 2) * box.h;
          if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
        });
        ctx.stroke();
      });
      ctx.globalAlpha = 1;
    },

    heatmap: function (ctx, w, h, d, p) {
      var box = plotBox(w, h);
      var rows = d.matrix || [];
      if (!rows.length) return;
      var cols = rows[0].length;
      var cw = box.w / cols, ch = box.h / rows.length;
      var lo = d.min, hi = d.max;
      if (lo === undefined || hi === undefined) {
        lo = Infinity; hi = -Infinity;
        rows.forEach(function (r) { r.forEach(function (v) { if (v < lo) lo = v; if (v > hi) hi = v; }); });
      }
      function ramp(t) {
        // viridis-ish dark→teal→yellow, readable in both themes
        t = Math.max(0, Math.min(1, t));
        var stops = [[13, 8, 74], [33, 106, 130], [53, 183, 121], [253, 231, 37]];
        var s = t * (stops.length - 1), i = Math.floor(s), f = s - i;
        var a = stops[i], b = stops[Math.min(i + 1, stops.length - 1)];
        return "rgb(" + Math.round(a[0] + (b[0] - a[0]) * f) + "," +
          Math.round(a[1] + (b[1] - a[1]) * f) + "," +
          Math.round(a[2] + (b[2] - a[2]) * f) + ")";
      }
      rows.forEach(function (r, ri) {
        r.forEach(function (v, ci) {
          ctx.fillStyle = ramp((v - lo) / (hi - lo || 1));
          ctx.fillRect(box.x + ci * cw, box.y + ri * ch, cw + 0.5, ch + 0.5);
        });
      });
      axes(ctx, box, p, { xlabel: d.xlabel || "frequency bin", ylabel: d.ylabel || "time →" });
    },

    line: function (ctx, w, h, d, p) {
      var box = plotBox(w, h);
      var series = d.series || [];
      var allX = [], allY = [];
      series.forEach(function (s) { s.points.forEach(function (pt) { allX.push(pt[0]); allY.push(pt[1]); }); });
      var xmin = d.xmin !== undefined ? d.xmin : Math.min.apply(null, allX);
      var xmax = d.xmax !== undefined ? d.xmax : Math.max.apply(null, allX);
      var ymin = d.ymin !== undefined ? d.ymin : Math.min.apply(null, allY);
      var ymax = d.ymax !== undefined ? d.ymax : Math.max.apply(null, allY);
      function X(v) { return box.x + ((v - xmin) / (xmax - xmin || 1)) * box.w; }
      function Y(v) { return box.y + (1 - (v - ymin) / (ymax - ymin || 1)) * box.h; }
      // gridlines + y ticks
      ctx.strokeStyle = p.border; ctx.fillStyle = p.muted; ctx.lineWidth = 1;
      ctx.font = '10px ' + cssVar("--font-sans", "sans-serif");
      ctx.textAlign = "right"; ctx.textBaseline = "middle";
      for (var g = 0; g <= 4; g++) {
        var yy = box.y + (g / 4) * box.h;
        ctx.globalAlpha = 0.5;
        ctx.beginPath(); ctx.moveTo(box.x, yy); ctx.lineTo(box.x + box.w, yy); ctx.stroke();
        ctx.globalAlpha = 1;
        var val = ymax - (g / 4) * (ymax - ymin);
        ctx.fillText(val.toFixed(d.yprec !== undefined ? d.yprec : 0), box.x - 6, yy);
      }
      axes(ctx, box, p, { xlabel: d.xlabel, ylabel: d.ylabel });
      // series
      series.forEach(function (s, si) {
        var col = s.color || p.series[si % p.series.length];
        ctx.strokeStyle = col; ctx.fillStyle = col; ctx.lineWidth = s.width || 2;
        if (s.dashed) ctx.setLineDash([5, 4]); else ctx.setLineDash([]);
        ctx.beginPath();
        s.points.forEach(function (pt, i) {
          if (i === 0) ctx.moveTo(X(pt[0]), Y(pt[1])); else ctx.lineTo(X(pt[0]), Y(pt[1]));
        });
        ctx.stroke();
        ctx.setLineDash([]);
        if (s.dots) s.points.forEach(function (pt) {
          ctx.beginPath(); ctx.arc(X(pt[0]), Y(pt[1]), 2.6, 0, Math.PI * 2); ctx.fill();
        });
      });
      // legend
      if (d.legend !== false && series.length > 1) {
        ctx.textAlign = "left"; ctx.textBaseline = "middle";
        ctx.font = '11px ' + cssVar("--font-sans", "sans-serif");
        var ly = box.y + 6;
        series.forEach(function (s, si) {
          var col = s.color || p.series[si % p.series.length];
          ctx.fillStyle = col; ctx.fillRect(box.x + 8, ly - 4, 14, 3);
          ctx.fillStyle = p.fg; ctx.fillText(s.label || ("series " + (si + 1)), box.x + 26, ly);
          ly += 16;
        });
      }
    },

    bars: function (ctx, w, h, d, p) {
      var box = plotBox(w, h);
      var cats = d.categories || [];
      var vals = d.values || [];
      var ymax = d.ymax !== undefined ? d.ymax : Math.max.apply(null, vals.concat([1]));
      var ymin = d.ymin !== undefined ? d.ymin : 0;
      function Y(v) { return box.y + (1 - (v - ymin) / (ymax - ymin || 1)) * box.h; }
      axes(ctx, box, p, { ylabel: d.ylabel, xlabel: d.xlabel });
      // threshold line (e.g. NIST p=0.01 pass/fail, or an entropy bound)
      if (d.threshold !== undefined) {
        ctx.strokeStyle = p.bad; ctx.setLineDash([5, 4]); ctx.lineWidth = 1.5;
        ctx.beginPath(); ctx.moveTo(box.x, Y(d.threshold)); ctx.lineTo(box.x + box.w, Y(d.threshold)); ctx.stroke();
        ctx.setLineDash([]);
      }
      var n = cats.length, bw = box.w / n * 0.62, gap = box.w / n;
      ctx.font = '10px ' + cssVar("--font-sans", "sans-serif");
      ctx.textAlign = "center";
      vals.forEach(function (v, i) {
        var x = box.x + gap * i + (gap - bw) / 2;
        var pass = d.pass ? d.pass[i] : (d.threshold !== undefined ? v >= d.threshold : true);
        ctx.fillStyle = d.colors ? d.colors[i] : (d.pass ? (pass ? p.good : p.bad) : p.accent);
        ctx.fillRect(x, Y(v), bw, box.y + box.h - Y(v));
        ctx.fillStyle = p.muted;
        ctx.fillText(cats[i], x + bw / 2, box.y + box.h + 14);
      });
    },

    timeline: function (ctx, w, h, d, p) {
      // Per-channel occupancy strips over time (RF Scope I/O graph style).
      var chans = d.channels || [];
      var labelW = 92, top = 10, rowH = Math.min(26, (h - 24) / Math.max(chans.length, 1));
      ctx.font = '11px ' + cssVar("--font-mono", "monospace");
      ctx.textBaseline = "middle";
      chans.forEach(function (c, ri) {
        var y = top + ri * rowH;
        ctx.fillStyle = p.muted; ctx.textAlign = "right";
        ctx.fillText(c.label, labelW - 8, y + rowH / 2);
        var strip = c.occ || [];
        var cw = (w - labelW - 8) / strip.length;
        strip.forEach(function (v, ci) {
          // v in 0..1 -> intensity of accent
          if (v <= 0) { ctx.fillStyle = p.elev; }
          else {
            ctx.fillStyle = p.accent;
            ctx.globalAlpha = 0.25 + 0.75 * Math.min(1, v);
          }
          ctx.fillRect(labelW + ci * cw, y + 2, cw + 0.5, rowH - 4);
          ctx.globalAlpha = 1;
        });
      });
      // time axis label
      ctx.fillStyle = p.muted; ctx.textAlign = "left";
      ctx.font = '10px ' + cssVar("--font-sans", "sans-serif");
      ctx.fillText(d.xlabel || "time →", labelW, top + chans.length * rowH + 12);
    }
  };

  function draw(canvas) {
    var type = canvas.getAttribute("data-chart");
    var renderer = renderers[type];
    if (!renderer) return;
    var holder = canvas.nextElementSibling;
    while (holder && !(holder.tagName === "SCRIPT" && holder.className.indexOf("lab-chart-data") !== -1)) {
      holder = holder.nextElementSibling;
    }
    var data = {};
    if (holder) {
      try { data = JSON.parse(holder.textContent); }
      catch (e) { if (window.console) console.warn("labcharts: bad JSON for", type, e); return; }
    }
    var dim = setupHiDPI(canvas);
    dim.ctx.clearRect(0, 0, dim.w, dim.h);
    renderer(dim.ctx, dim.w, dim.h, data, palette());
  }

  function drawAll() {
    var charts = document.querySelectorAll("canvas.lab-chart");
    for (var i = 0; i < charts.length; i++) draw(charts[i]);
  }

  function init() {
    if (!document.querySelector("canvas.lab-chart")) return; // no-op off-topic pages
    drawAll();
    // Redraw on theme toggle (site.js flips :root.dark) …
    var mo = new MutationObserver(function () { drawAll(); });
    mo.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
    // … and on resize (debounced).
    var t;
    window.addEventListener("resize", function () {
      clearTimeout(t);
      t = setTimeout(drawAll, 150);
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
