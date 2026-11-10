// Shared by every film: a frame is a pure function of t, so the render, the stills and the score agree.
const cl = (x, a = 0, b = 1) => Math.min(b, Math.max(a, x));
const p = (t, a, b) => cl((t - a) / (b - a));
const ex = x => (x >= 1 ? 1 : 1 - Math.pow(2, -10 * x));
const eio = x => (x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2);
const back = x => { const c1 = 1.9, c3 = c1 + 1; return 1 + c3 * Math.pow(x - 1, 3) + c1 * Math.pow(x - 1, 2); };
const $ = s => document.querySelector(s);
const $$ = s => [...document.querySelectorAll(s)];

function show(el, o, tf = "none", blur = 0) {
  el.style.opacity = o;
  el.style.transform = tf;
  el.style.filter = blur > 0.3 ? `blur(${blur.toFixed(1)}px)` : "none";
}

// Enter by rising, leave by lifting: the one gesture every block shares, with only a trace of blur.
function ee(el, t, a, b, { di = 0.45, dout = 0.25, dy = 40, blur = 6, sc = 0, extra = "" } = {}) {
  if (t < a || t >= b) { el.style.opacity = 0; return 0; }
  const i = ex(p(t, a, a + di)), o = eio(p(t, b - dout, b));
  const y = (1 - i) * dy - o * dy * 0.7;
  const s = 1 + (1 - i) * sc - o * sc * 0.4;
  show(el, i * (1 - o), `${extra} translateY(${y.toFixed(1)}px) scale(${s.toFixed(4)})`, (1 - i) * blur + o * blur);
  return i * (1 - o);
}

// What the seller types keeps a caret, so it never reads as text the system wrote.
function typeInto(el, full, t, a, b) {
  const n = Math.round(cl((t - a) / (b - a)) * full.length);
  el.textContent = full.slice(0, n);
  const idle = t >= b && t < b + 0.9;
  el.classList.toggle("caret", (t >= a && t < b) || idle);
  el.classList.toggle("off", idle && Math.floor(t * 2.4) % 2 === 1);
}

// A film is judged by ear in a browser, where no capture is there to drive the clock.
function mountPlayer(audioSrc) {
  if (new URLSearchParams(location.search).has("capture")) return;
  const audio = new Audio(audioSrc);
  let t0 = null;
  document.addEventListener("click", () => { t0 = performance.now(); audio.currentTime = 0; audio.play().catch(() => {}); });
  const at = new URLSearchParams(location.search).get("t");
  const loop = () => { render(at != null ? +at : t0 == null ? 0 : (performance.now() - t0) / 1000); requestAnimationFrame(loop); };
  window.ready.then(loop);
}

// A pinned narration is measured once for good, so only the pin it was read from has to match.
function measurementHolds(m, line, voice) {
  if (!m) return false;
  if (voice.pin) return m.pin === voice.pin;
  return !m.pin && m.say === line.say && (m.take ?? null) === (line.take ?? null) && m.voice === voice.voice && m.model === voice.model && m.instructions === voice.instructions && m.tempo === (voice.tempo ?? 1);
}

// Until the voice step has measured a line, the film still opens on an estimate, so the lines can be read out of it.
async function loadVoice(voice) {
  let measured = {};
  try { const r = await fetch(".build/voice.json"); if (r.ok) measured = (await r.json()).lines; } catch {}
  const lines = {};
  window.voiceUnmeasured = [];
  for (const l of voice.lines) {
    const m = measured[l.id], fresh = measurementHolds(m, l, voice);
    if (!fresh) window.voiceUnmeasured.push(l.id);
    const dur = fresh ? m.dur : l.say.split(/\s+/).length / 2.6 + 0.3;
    lines[l.id] = { ...l, file: fresh ? m.file : null, dur, speech: fresh ? m.speech : [0.05, dur - 0.1], pauses: fresh ? m.pauses : [], ...(fresh && m.offset != null ? { offset: m.offset } : {}) };
  }
  return lines;
}

// Chunks share the voiced span by length, punctuation weighing like a few letters; a measured pause near a
// boundary then wins, because that is where the voice really turns the phrase.
function timeChunks(line) {
  const parts = line.text.split("|").map(s => s.trim());
  const w = parts.map(s => s.replace(/[[\]\s]/g, "").length + (/[.,:?!]\]?$/.test(s) ? 4 : 0));
  const total = w.reduce((a, b) => a + b, 0);
  const [s0, s1] = line.speech;
  let acc = 0;
  const edges = parts.map((_, k) => { const e = { a: s0 + (acc / total) * (s1 - s0) }; acc += w[k]; e.b = s0 + (acc / total) * (s1 - s0); return e; });
  const free = [...line.pauses];
  for (let k = 1; k < edges.length; k++) {
    const at = edges[k].a;
    // A pause before the previous chunk's start would end that chunk before it begins.
    const best = free.map((g, i) => [Math.abs((g[0] + g[1]) / 2 - at), i]).filter(([, i]) => free[i][0] > edges[k - 1].a && free[i][1] < edges[k].b).sort((x, y) => x[0] - y[0])[0];
    if (best && best[0] < 0.6) { const [g] = free.splice(best[1], 1); edges[k - 1].b = g[0]; edges[k].a = g[1]; }
  }
  return parts.map((s, k) => ({ text: s.replace(/[[\]]/g, ""), onScreen: /^\[.*\]$/.test(s), a: line.at + edges[k].a, b: line.at + edges[k].b }));
}

// Each scene holds until its line has been said, rounded up to the beat so every cut stays on the grid.
function layoutScenes(scenes, lines, opts = {}) {
  if (scenes.some(s => s.line) && scenes.every(s => !s.line || lines[s.line].offset != null)) return layoutNarrated(scenes, lines, opts);
  const { grid = 0.5, lead = 0.2, tail = 0.5 } = opts;
  let t = 0;
  const out = {};
  for (const s of scenes) {
    const line = s.line ? lines[s.line] : null;
    const ld = s.lead ?? lead;
    const need = line ? ld + line.speech[1] + (s.tail ?? tail) : 0;
    const len = Math.ceil(Math.max(s.min ?? 0, need) / grid - 1e-9) * grid;
    out[s.id] = { a: t, b: t + len };
    if (line) { line.at = t + ld; line.chunks = timeChunks(line); }
    t += len;
  }
  return { scenes: out, duration: t };
}

// The mix places each line where the timeline put it; a line located inside a narration says where it sits in it.
const voiceCues = lines => lines.map(l => ({ id: l.id, at: l.at, file: l.file, speech: l.speech.map(x => l.at + x), ...(l.offset != null ? { offset: l.offset } : {}) }));

// One narration plays whole, so the scenes follow where the voice changes line instead of each line being placed in
// its scene; a scene that needs more time than the voice gives it asks for a pause in the script, never a moved word.
function layoutNarrated(scenes, lines, { lead = 0.2, tail = 0.5 } = {}) {
  const first = scenes.find(s => s.line);
  const shift = (first.lead ?? lead) - lines[first.line].speech[0];
  const out = {};
  window.layoutWarnings = [];
  scenes.forEach((s, i) => {
    const line = lines[s.line], next = scenes[i + 1];
    const a = i === 0 ? 0 : shift + line.offset;
    const b = next ? shift + lines[next.line].offset : Math.max(shift + line.offset + line.speech[1] + (s.tail ?? tail), a + (s.min ?? 0));
    if (next && s.min && b - a < s.min) window.layoutWarnings.push(`${s.id} lasts ${(b - a).toFixed(2)} s for ${s.min} s it needs: add a pause to the script`);
    out[s.id] = { a, b };
    line.at = shift + line.offset;
    line.chunks = timeChunks(line);
  });
  const last = out[scenes.at(-1).id];
  return { scenes: out, duration: Math.ceil(last.b * 4 - 1e-9) / 4 };
}

// A caption never blinks out between chunks of one line: the eye loses its place when it does.
// The hold lets a caption be read after its line, but never covers the next line's words.
function captionAt(lines, t, hold = 0.35) {
  const starts = Object.values(lines).flatMap(l => (l.chunks ?? []).map(c => c.a));
  for (const l of Object.values(lines)) {
    if (!l.chunks) continue;
    const c = l.chunks;
    for (let k = 0; k < c.length; k++) {
      const end = k + 1 < c.length ? c[k + 1].a : Math.min(c[k].b + hold, ...starts.filter(a => a > c[k].a));
      if (t >= c[k].a && t < end) return { line: l, k, chunk: c[k], a: c[k].a, b: end };
    }
  }
  return null;
}
