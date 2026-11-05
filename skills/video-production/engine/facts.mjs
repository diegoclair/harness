import { writeFileSync } from "node:fs";
import { openFilm } from "./lib.mjs";

// What QA reads off the page, so a film.html and a declarative film are judged by the same numbers.
const { page, close } = await openFilm();
const facts = await page.evaluate(() => {
  const cues = window.CUES;
  const lines = cues.voice.map(v => {
    const l = window.voiceLines[v.id];
    return { id: l.id, text: l.text, chunks: l.chunks ?? [] };
  });
  window.render(cues.duration - 0.05);
  const visible = el => { for (let e = el; e && e !== document.body; e = e.parentElement) if (+getComputedStyle(e).opacity === 0 || getComputedStyle(e).display === "none") return false; return true; };
  const last = [...document.querySelectorAll(".scene.on")].pop();
  const walk = last ? document.createTreeWalker(last, NodeFilter.SHOW_TEXT) : null;
  const words = [];
  while (walk?.nextNode()) if (walk.currentNode.textContent.trim() && visible(walk.currentNode.parentElement)) words.push(walk.currentNode.textContent.trim());
  const end = cues.scenes[cues.scenes.length - 1];
  return { lines, endCard: { text: cues.endCard?.text ?? words.join(" "), readyAt: cues.endCard?.readyAt ?? end[1], end: end[2] }, target: cues.target ?? null };
});
await close();
writeFileSync(".build/facts.json", JSON.stringify(facts));
