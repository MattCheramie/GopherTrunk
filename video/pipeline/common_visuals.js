/* Common visuals shared by all segment scenes: title card, recap card,
 * generic content cards, GopherTrunk dashboard mock, course map card. */
window.CV = (() => {
  const { C, div, svg, el, text } = Fig;
  const E = () => Engine;

  const splitArgs = ev => (ev.args || "").split("|").map(s => s.trim().replace(/^"|"$/g, ""));

  function motif(parent) {
    let d = "M -40 540";
    for (let x = -40; x <= 2000; x += 8)
      d += ` L ${x} ${540 + 90 * Math.sin(x / 55) * Math.sin(x / 700)}`;
    const wrap = div(parent, "motif");
    const s = svg(wrap, "0 0 1920 1080", { preserveAspectRatio: "xMidYMid slice" });
    el("path", { d, stroke: C.accent, "stroke-width": 3, fill: "none" }, s);
  }

  const title = {
    mount(elx, ev) {
      const [term, segid] = splitArgs(ev);
      elx.innerHTML = `
        <div class="title-card">
          <div class="eyebrow">GopherTrunk Field Guide</div>
          <div class="term">${term}</div>
          <div class="rule"></div>
          <div class="segid">${segid ?? ""}</div>
          <div class="stripe"></div>
          <div class="bug"><img src="../../../docs/assets/gophertrunk-logo.png"></div>
        </div>`;
      motif(elx.querySelector(".title-card"));
    },
    seek(elx, tl) {
      const e = E();
      e.enter(elx.querySelector(".term"), e.rise(tl, 0.05, 0.5));
      e.enter(elx.querySelector(".eyebrow"), e.rise(tl, 0.0, 0.4));
      elx.querySelector(".rule").style.opacity = e.rise(tl, 0.3, 0.4);
    },
  };

  const recap = {
    mount(elx, ev) {
      const a = splitArgs(ev);
      const term = a[0], bullets = a.slice(1).map(b => b.replace(/^[①②③]\s*/, ""));
      elx.innerHTML = `
        <div class="recap-card">
          <div class="head">Recap</div>
          <div class="term">${term}</div>
          <div class="bullets">${bullets.map((b, i) =>
            `<div class="bullet"><div class="num">${i + 1}</div><div>${b}</div></div>`).join("")}
          </div>
          <div class="footer">Full write-up → <span class="url">gophertrunk.org/reference/${(window.TIMELINE.meta.slug || "").trim()}/</span></div>
          <div class="bug"><img src="../../../docs/assets/gophertrunk-logo.png"></div>
        </div>`;
      motif(elx.querySelector(".recap-card"));
    },
    seek(elx, tl, ev) {
      const e = E();
      const bl = [...elx.querySelectorAll(".bullet")];
      const dur = Math.max(3, (ev.t1 - ev.t0) * 0.45);
      bl.forEach((b, i) => e.enter(b, e.rise(tl, 0.4 + i * (dur / bl.length), 0.5)));
      e.enter(elx.querySelector(".footer"), e.rise(tl, 0.4 + dur, 0.6));
    },
  };

  // headline + big centered card; scenes pass builder for the card interior
  function centerCard(headline, innerHTML, extra = "") {
    return `
      <div class="safe-headline" style="position:absolute;top:120px;width:100%;text-align:center;
        font-size:44px;font-weight:650;">${headline}</div>
      <div class="card center-card" style="position:absolute;left:50%;top:52%;
        transform:translate(-50%,-50%);${extra}">${innerHTML}</div>`;
  }

  // GopherTrunk dashboard mock: rows of label/meter/value, panel look
  function gtPanel(parent, rows, title = "GopherTrunk — live decode") {
    const p = div(parent, "", `position:absolute;left:50%;top:54%;transform:translate(-50%,-50%);
      width:1100px;background:${C.bg};border:1.5px solid ${C.border};border-radius:14px;
      box-shadow:0 18px 70px rgba(0,0,0,.5);overflow:hidden;`);
    div(p, "", `height:64px;background:${C.elev};border-bottom:1px solid ${C.border};
      display:flex;align-items:center;padding:0 28px;font-size:26px;font-weight:600;
      color:${C.text};gap:14px;`,
      `<span style="width:14px;height:14px;border-radius:50%;background:${C.good};display:inline-block"></span>${title}`);
    const body = div(p, "", "padding:26px 28px;display:flex;flex-direction:column;gap:22px;");
    const made = rows.map(r => {
      const row = div(body, "gt-row", "display:flex;align-items:center;gap:24px;opacity:0;");
      div(row, "", `flex:0 0 300px;font-size:27px;color:${C.muted};`, r.label);
      const meter = div(row, "", `flex:1;height:18px;background:${C.elev};border-radius:9px;overflow:hidden;`);
      const fill = div(meter, "gt-fill", `height:100%;width:0%;border-radius:9px;background:${r.color || C.accent};`);
      const val = div(row, "gt-val", `flex:0 0 300px;font-family:var(--mono);font-size:27px;
        color:${C.text};text-align:right;white-space:nowrap;`, "");
      return { row, fill, val, spec: r };
    });
    return made;
  }

  // course map card (intro / transitions / outro)
  const CHAPTERS = ["Radio wave", "Frequency", "Modulation", "Bandwidth", "Decibels", "SNR"];
  function mapCard(parent, sub = "") {
    const wrap = div(parent, "", "position:absolute;inset:0;");
    motif(wrap);
    div(wrap, "", `position:absolute;top:150px;width:100%;text-align:center;font-size:34px;
      font-weight:600;letter-spacing:.18em;text-transform:uppercase;color:${C.muted};`,
      "Radio Fundamentals — course map");
    const box = div(wrap, "", `position:absolute;top:300px;left:50%;transform:translateX(-50%);
      width:1440px;display:flex;align-items:center;justify-content:center;gap:0;`);
    const nodes = CHAPTERS.map((name, i) => {
      if (i > 0) div(box, "map-link", `width:60px;height:4px;background:${C.border};margin:0 4px;`);
      const n = div(box, "map-node", `width:180px;padding:26px 10px;text-align:center;
        border:2px solid ${C.border};border-radius:14px;background:${C.elev};
        font-size:27px;font-weight:600;color:${C.muted};transition:none;`);
      n.textContent = name;
      return n;
    });
    if (sub) div(wrap, "map-sub", `position:absolute;top:560px;width:100%;text-align:center;
      font-size:38px;color:${C.text};font-weight:550;`, sub);
    return nodes;
  }
  function setNode(n, state, pulseT = 0) {
    if (state === "dim") { n.style.borderColor = C.border; n.style.color = C.muted; n.style.background = C.elev; n.style.boxShadow = "none"; }
    if (state === "lit") { n.style.borderColor = C.accent; n.style.color = C.text; n.style.background = "rgba(88,166,255,.10)"; n.style.boxShadow = "none"; }
    if (state === "pulse") {
      const g = 0.5 + 0.5 * Math.sin(pulseT * 2 * Math.PI * 0.8);
      n.style.borderColor = C.accent; n.style.color = C.text;
      n.style.background = `rgba(88,166,255,${0.08 + 0.10 * g})`;
      n.style.boxShadow = `0 0 ${18 + 22 * g}px rgba(88,166,255,${0.25 + 0.3 * g})`;
    }
  }

  return { splitArgs, motif, title, recap, centerCard, gtPanel, mapCard, setNode, CHAPTERS };
})();
