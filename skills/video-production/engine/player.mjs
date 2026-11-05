import { sceneLibrary } from "./scenes/index.mjs";
import { drawCaption } from "./scenes/captions.mjs";
import { applyVariant, resolveLine, resolveVoice, sceneInstances, lineLead } from "./film-core.mjs";
import { THRESHOLDS as T } from "./qa-thresholds.mjs";

const BEAT = 0.5;
// The reading rule of the engine, which every card holds to when its line alone would be shorter.
const READING = { wps: T.readingWps, extra: T.readingExtraSec };
const DEFAULT_LEAD = 0.05, DEFAULT_TAIL = 0.1;
const stage = document.querySelector("#stage");
const getJson = async url => { const r = await fetch(url); if (!r.ok) throw new Error(`${r.status} ${url}`); return r.json(); };
const getText = async url => { const r = await fetch(url); if (!r.ok) throw new Error(`${r.status} ${url}`); return r.text(); };

function linkStyles(href) {
  const link = Object.assign(document.createElement("link"), { rel: "stylesheet", href });
  return new Promise((resolve, reject) => { link.onload = resolve; link.onerror = () => reject(new Error(`no styles at ${href}`)); document.head.append(link); });
}

// The ad's readable box: Reels covers the top bar, the bottom caption and the right column of buttons.
const SAFE_BOX = { x: 20, y: 118, scale: 0.75 };

// A fitted scene is drawn for the whole frame and scaled into a box, which is how a paid ad keeps clear of the feed's buttons.
function mountDom(inst, safe) {
  const root = document.createElement("div");
  root.className = "scene";
  const html = inst.scene.html(inst.params), box = inst.fit ?? (safe ? inst.scene.safeFit ?? SAFE_BOX : null);
  root.innerHTML = box ? `<div class="fit" style="transform:translate(${box.x}px,${box.y}px) scale(${box.scale})">${html}</div>` : html;
  stage.insertBefore(root, stage.querySelector("#vig"));
  return root;
}

function toContext(inst, root, lines, slots) {
  const parts = inst.lines.map(id => ({ line: lines[id], ...slots[id] }));
  const chunks = parts[0].line.chunks;
  return {
    root, scene: inst.scene, params: inst.params, parts, chunks, line: parts[0].line, T: {}, caption: inst.caption ?? inst.scene.caption,
    a: parts[0].a, b: parts[parts.length - 1].b,
    q: s => root.querySelector(s), qa: s => [...root.querySelectorAll(s)],
    chunk: k => chunks[Math.min(k, chunks.length - 1)],
  };
}

function mergeCues(into, extra) {
  for (const [k, v] of Object.entries(extra ?? {})) into[k] = Array.isArray(v) ? [...(into[k] ?? []), ...v] : v;
}

// The kit names svg files whose markup goes inline, so the drawing takes the colour of the text around it.
async function inlineSvgs(map) {
  for (const [selector, url] of Object.entries(map ?? {})) {
    const svg = await getText(url);
    stage.querySelectorAll(selector).forEach(el => { el.innerHTML = svg; el.querySelector("svg").removeAttribute("color"); });
  }
}

window.ready = (async () => {
  const [base, kit] = await Promise.all([getJson("film.json"), getJson("/__kit/kit.json")]);
  const [dictionary, SCENES] = await Promise.all([getJson(`/__kit/${kit.dictionary}`), sceneLibrary(`/__kit/${kit.scenes}`)]);
  if (kit.styles) await linkStyles(`/__kit/${kit.styles}`);
  if (kit.lang) document.documentElement.lang = kit.lang;
  if (kit.title) document.title = kit.title;
  const film = applyVariant(base, new URLSearchParams(location.search).get("variant"));
  // A paid ad is covered by the feed's own buttons and caption, so its picture is fitted inside what stays visible.
  stage.classList.toggle("safe", !!film.safe);
  const voiceSpec = resolveVoice(kit, film.voice);
  const resolved = film.lines.map(l => resolveLine(l, dictionary, voiceSpec.voice));
  const VOICE = { ...voiceSpec, lines: resolved, pin: film.narration?.spans };
  window.VOICE = VOICE;
  const LINES = await loadVoice(VOICE);
  window.voiceLines = LINES;

  const instances = sceneInstances(film.lines).map(i => ({ ...i, scene: SCENES[i.scene] }));
  document.head.insertAdjacentHTML("beforeend", `<style>${[...new Set(instances.map(i => i.scene.css ?? ""))].join("\n")}</style>`);
  const owner = Object.fromEntries(instances.flatMap(i => i.lines.map(id => [id, i])));
  const slots = film.lines.map((l, i) => {
    const inst = owner[l.id], lastOf = inst.lines[inst.lines.length - 1] === l.id;
    return {
      id: l.id, line: l.id, lead: lineLead(l, LINES[l.id].speech[0], i === 0, DEFAULT_LEAD),
      tail: l.tail ?? (lastOf ? inst.scene.tail : undefined) ?? DEFAULT_TAIL,
      min: l.min ?? inst.scene.minSeconds?.(inst.params, READING),
    };
  });
  const lay = layoutScenes(slots, LINES, { grid: BEAT });

  const roots = instances.map(inst => mountDom(inst, film.safe));
  await inlineSvgs(kit.svg);
  await Promise.all([document.fonts.ready, ...[...stage.querySelectorAll("img")].map(im => im.decode())]);
  const ctxs = instances.map((inst, i) => toContext(inst, roots[i], LINES, lay.scenes));
  ctxs.forEach(c => c.scene.mount?.(c));

  const firstChunk = LINES[film.lines[0].id].chunks[0];
  const cc = stage.querySelector("#cc"), box = cc.querySelector("span"), vig = stage.querySelector("#vig");
  window.render = t => {
    let vignette = true;
    for (const c of ctxs) {
      const live = t >= c.a && t < c.b;
      c.root.classList.toggle("on", live);
      if (!live) continue;
      c.scene.render(t, c);
      if (c.scene.vignette === false) vignette = false;
    }
    vig.style.opacity = vignette ? 1 : 0;
    drawCaption(t, LINES, firstChunk, box, cc, ctxs);
  };

  const cues = {
    duration: lay.duration, beat: BEAT, finale: ctxs[ctxs.length - 1].a, target: film.target,
    scenes: slots.map(s => [s.id, lay.scenes[s.id].a, lay.scenes[s.id].b]),
    cuts: ctxs.slice(1).map(c => c.a),
    voice: voiceCues(film.lines.map(l => LINES[l.id])),
  };
  ctxs.forEach(c => mergeCues(cues, c.scene.cues?.(c)));
  window.CUES = cues;
  await new Promise(r => setTimeout(r, 300));
})();

mountPlayer("mix.wav");
