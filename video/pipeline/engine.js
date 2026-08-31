/* GopherTrunk motion-graphics engine — deterministic, clock-driven.
 *
 * Contract: the renderer injects window.TIMELINE (a segment timeline.json) and
 * the scene file calls Engine.init({visualName: {mount, seek}, ...}).
 * window.seek(t) then sets ALL visual state for absolute time t — no CSS
 * animations, no rAF, no wall-clock. See gophertrunk-video-pipeline.md §8.
 *
 * A visual def: {
 *   mount(el, ev)        — build DOM once into layer el; ev = timeline event
 *                          (ev.lines[i].t0 are ABSOLUTE VO line times — use
 *                          Engine.rel(ev, i) for the line start relative to ev)
 *   seek(el, tl, ev)     — set state for local time tl (0 at ev.t0)
 * }
 */
window.Engine = (() => {
  const XFADE = 0.35;
  let events = [], defs = {}, layers = [];

  const clamp01 = x => Math.max(0, Math.min(1, x));
  // "numbers-flying — absurd numbers stream past" → "numbers-flying"
  const vname = v => v.split("—")[0].split(":")[0].trim();
  const easeOut = p => 1 - Math.pow(1 - clamp01(p), 3);
  const easeIO = p => { p = clamp01(p); return p < 0.5 ? 4 * p * p * p : 1 - Math.pow(-2 * p + 2, 3) / 2; };
  // reveal progress for an element appearing at t0, easing over dur
  const rise = (tl, t0, dur = 0.45) => easeOut((tl - t0) / dur);

  // apply standard entrance: fade + 24px lift
  function enter(el, p) {
    el.style.opacity = p;
    // CSS `translate` is independent of `transform`, so centered elements keep their transform
    el.style.translate = `0 ${24 * (1 - easeOut(p))}px`;
    el.style.visibility = p <= 0 ? "hidden" : "visible";
  }

  // staged reveals: items[i] appears at times[i] (local seconds)
  function staged(items, times, dur = 0.45) {
    return (tl) => items.forEach((el, i) => enter(el, rise(tl, times[i], dur)));
  }

  function rel(ev, i) {  // start of VO line i, relative to the event
    if (!ev.lines || !ev.lines[i]) return 0.2;
    return ev.lines[i].t0 - ev.t0;
  }

  function init(sceneDefs) {
    defs = sceneDefs;
    const T = window.TIMELINE;
    const root = document.getElementById("root");
    events = T.events;
    for (const ev of events) {
      const layer = document.createElement("div");
      layer.className = "stage layer";
      layer.style.opacity = 0;
      root.appendChild(layer);
      layers.push(layer);
      const def = defs[vname(ev.visual)];
      if (!def) throw new Error("no visual def: " + ev.visual);
      def.mount(layer, ev);
    }
    window.seek = seek;
    window.ready = true;
    seek(0);
    return true;
  }

  function seek(t) {
    for (let i = 0; i < events.length; i++) {
      const ev = events[i], layer = layers[i];
      const hard = ev.visual === "title" || ev.visual === "recap" ||
                   (i > 0 && events[i - 1].visual === "title");
      const fadeIn = hard ? 0.001 : XFADE;
      const t1 = ev.t1 + (i === events.length - 1 ? 10 : 0);
      if (t < ev.t0 - 0.001 || t >= t1) { layer.style.opacity = 0; layer.style.visibility = "hidden"; continue; }
      layer.style.visibility = "visible";
      layer.style.opacity = clamp01((t - ev.t0) / fadeIn);
      const def = defs[vname(ev.visual)];
      def.seek && def.seek(layer, t - ev.t0, ev);
    }
    return true;
  }

  return { init, seek, clamp01, easeOut, easeIO, rise, enter, staged, rel, XFADE };
})();
