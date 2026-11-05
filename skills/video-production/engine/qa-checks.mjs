import { THRESHOLDS as T } from "./qa-thresholds.mjs";

export const fold = s => s.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase();

const UNITS = ["zero", "um", "dois", "tres", "quatro", "cinco", "seis", "sete", "oito", "nove", "dez", "onze", "doze", "treze", "quatorze", "quinze", "dezesseis", "dezessete", "dezoito", "dezenove"];
const TENS = [null, null, "vinte", "trinta", "quarenta", "cinquenta", "sessenta", "setenta", "oitenta", "noventa"];
function numberWords(n) {
  if (n === 100) return ["cem"];
  if (n < 20) return [UNITS[n]];
  if (n < 100) return n % 10 ? [TENS[Math.floor(n / 10)], "e", UNITS[n % 10]] : [TENS[n / 10]];
  return [String(n)];
}

// The transcript writes figures as digits and the script as words, and neither is wrong.
export function tokens(text) {
  const out = [];
  for (const raw of fold(text).replace(/\[|\]|\|/g, " ").split(/\s+/)) {
    const pct = raw.match(/^(\d+)%/);
    const word = raw.replace(/^[^\p{L}\p{N}]+|[^\p{L}\p{N}]+$/gu, "");
    if (pct) out.push(...numberWords(+pct[1]), "por", "cento");
    else if (/^\d+$/.test(word) && +word <= 100) out.push(...numberWords(+word));
    else if (word) out.push(word);
  }
  return out;
}

// Longest common subsequence: the hunks it leaves are exactly the words the two texts disagree on.
export function diffWords(script, heard) {
  const n = script.length, m = heard.length, dp = Array.from({ length: n + 1 }, () => new Uint16Array(m + 1));
  for (let i = n - 1; i >= 0; i--) for (let j = m - 1; j >= 0; j--) dp[i][j] = script[i] === heard[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1]);
  const hunks = [], matched = new Array(n).fill(false);
  let i = 0, j = 0, cur = null;
  const flush = () => { if (cur) hunks.push(cur); cur = null; };
  while (i < n || j < m) {
    if (i < n && j < m && script[i] === heard[j]) { matched[i] = true; flush(); i++; j++; continue; }
    cur ??= { at: i, script: [], heard: [] };
    if (j >= m || (i < n && dp[i + 1][j] >= dp[i][j + 1])) cur.script.push(script[i++]);
    else cur.heard.push(heard[j++]);
  }
  flush();
  return { hunks, matched };
}

function nameIndex(dictionary) {
  return dictionary.entries.map(e => ({ word: e.word, parts: tokens(e.word), heard: [...tokens(e.word).join(" ").split(), ...e.heard_as] }));
}

const inHunk = (hunk, name) => {
  const s = hunk.script.join(" "), h = hunk.heard.join(" ");
  return (s && name.parts.some(p => s.split(" ").includes(p))) || (h && name.heard.some(a => h.includes(a)));
};

export function compareTranscript(lines, transcriptText, dictionary) {
  const lineTokens = lines.map(l => ({ id: l.id, words: tokens(l.text) }));
  const script = lineTokens.flatMap(l => l.words), heard = tokens(transcriptText);
  const { hunks, matched } = diffWords(script, heard);
  const names = nameIndex(dictionary);
  const nameHunks = [], wrong = [];
  for (const h of hunks) (names.some(n => inHunk(h, n)) ? nameHunks : wrong).push(h);
  // A hunk about a name is the ear's call, so it never counts against the script's words.
  const excused = new Set(nameHunks.flatMap(h => h.script.map((_, k) => h.at + k)));
  let offset = 0;
  const coverage = lineTokens.map(l => {
    const idx = l.words.map((_, k) => offset + k);
    offset += l.words.length;
    const hit = idx.filter(i => matched[i] || excused.has(i)).length;
    return { id: l.id, ratio: idx.length ? hit / idx.length : 1 };
  });
  const missed = wrong.reduce((s, h) => s + Math.max(h.script.length, h.heard.length), 0);
  const occurrences = names.filter(n => script.join(" ").includes(n.parts.join(" "))).map(n => n.word);
  return { total: script.length, missed, ratio: script.length ? missed / script.length : 0, wrong, nameHunks, coverage, names: occurrences };
}

export function parseTranscript(md) {
  return [...md.matchAll(/^`\d+:\d+` (.*?)\s*$/gm)].map(m => m[1]).join(" ");
}

// The hook is the first spoken sentence: a line can run on past it without the hook running on.
export function hookEnd(lines) {
  const first = lines[0];
  const sentence = first.chunks.find(c => /[.?!]$/.test(c.text));
  return (sentence ?? first.chunks[first.chunks.length - 1]).b;
}

export function speechCuts(voice, scenes) {
  const out = [];
  for (const v of voice) {
    const [start, end] = v.speech;
    const scene = scenes.find(([, a, b]) => start >= a && start < b);
    if (scene && end > scene[2] + 0.001) out.push({ id: v.id, over: end - scene[2], scene: scene[0] });
  }
  return out;
}

export function captionSpans(lines, hold) {
  return lines.flatMap(l => l.chunks.map((c, k) => {
    const end = k + 1 < l.chunks.length ? l.chunks[k + 1].a : c.b + hold;
    return { line: l.id, text: c.text, onScreen: c.onScreen, seconds: end - c.a, words: c.text.split(/\s+/).length };
  }));
}

export function shortCaptions(spans) {
  return spans.filter(s => !s.onScreen && (s.seconds < T.captionMinSec || s.words / s.seconds > T.captionMaxWps));
}

// Compared against the frame that opened the stretch, so a slow push adds up to a change and a freeze never does.
export function longestStill(frames, fps, threshold = T.stillDiff) {
  if (!frames.length) return { seconds: 0, at: 0 };
  let anchor = 0, best = { seconds: 0, at: 0 };
  for (let i = 1; i <= frames.length; i++) {
    if (i < frames.length && meanDiff(frames[anchor], frames[i]) < threshold) continue;
    const seconds = (i - anchor) / fps;
    if (seconds > best.seconds) best = { seconds, at: anchor / fps };
    anchor = i;
  }
  return best;
}

export function meanDiff(a, b) {
  let sum = 0;
  for (let i = 0; i < a.length; i++) sum += Math.abs(a[i] - b[i]);
  return sum / a.length;
}

export function lumaStd(frame) {
  let sum = 0, sq = 0;
  for (const v of frame) { sum += v; sq += v * v; }
  const mean = sum / frame.length;
  return Math.sqrt(Math.max(0, sq / frame.length - mean * mean));
}

export const readingSeconds = text => text.trim().split(/\s+/).filter(Boolean).length / T.readingWps + T.readingExtraSec;

const pass = (ok, id, label, measured, limit, detail) => ({ id, label, ok, measured, limit, detail });

export function runChecks(m) {
  const { cues, facts, compare, lufs, truePeak, fileDuration, still, frame0Std, cuts, spans } = m;
  const short = shortCaptions(spans), firstSpeech = Math.min(...cues.voice.map(v => v.speech[0])), hook = hookEnd(facts.lines);
  const endHeld = facts.endCard.end - facts.endCard.readyAt, endNeed = readingSeconds(facts.endCard.text);
  const offTarget = facts.target != null && Math.abs(fileDuration - facts.target) > T.targetToleranceSec;
  const durationOff = Math.abs(fileDuration - cues.duration);
  const garbled = compare.coverage.filter(c => c.ratio < T.lineMinCoverage);
  return [
    pass(durationOff <= T.durationToleranceSec && !offTarget, "duration", "Duração", `${fileDuration.toFixed(2)} s`, `${cues.duration} s ± ${T.durationToleranceSec}${facts.target != null ? `, alvo ${facts.target} ± ${T.targetToleranceSec}` : ""}`),
    pass(Math.abs(lufs - T.loudnessLufs) <= T.loudnessToleranceLu, "loudness", "Loudness", `${lufs.toFixed(1)} LUFS`, `${T.loudnessLufs} ± ${T.loudnessToleranceLu}`),
    pass(truePeak < T.truePeakDbtp, "peak", "Pico real", `${truePeak.toFixed(1)} dBTP`, `abaixo de ${T.truePeakDbtp}`),
    pass(firstSpeech <= T.firstSpeechSec, "first-speech", "Primeira fala", `${firstSpeech.toFixed(2)} s`, `até ${T.firstSpeechSec} s`),
    pass(hook <= T.hookEndSec, "hook", "Fim do gancho", `${hook.toFixed(2)} s`, `até ${T.hookEndSec} s`),
    pass(!cuts.length, "speech-cut", "Fala cortada por cena", `${cuts.length}`, "0", cuts.map(c => `${c.id} passa ${c.over.toFixed(2)} s do fim da cena ${c.scene}`)),
    pass(compare.ratio <= T.transcriptMaxMismatch && !garbled.length, "transcript", "Transcrição contra o roteiro", `${compare.missed} de ${compare.total} palavras (${(compare.ratio * 100).toFixed(0)}%)`, `até ${T.transcriptMaxMismatch * 100}%, e cada fala com ${T.lineMinCoverage * 100}% das palavras`,
      [...garbled.map(g => `fala ${g.id}: só ${(g.ratio * 100).toFixed(0)}% das palavras ouvidas`), ...compare.wrong.map(h => `roteiro "${h.script.join(" ")}", ouvido "${h.heard.join(" ")}"`)]),
    pass(!short.length, "captions", "Legenda na tela", `${short.length} curtas de ${spans.filter(s => !s.onScreen).length}`, `pelo menos ${T.captionMinSec} s e até ${T.captionMaxWps} palavras por segundo`, short.map(s => `"${s.text}" fica ${s.seconds.toFixed(2)} s`)),
    pass(still.seconds <= T.stillMaxSec, "still", "Maior trecho parado", `${still.seconds.toFixed(1)} s a partir de ${still.at.toFixed(1)} s`, `até ${T.stillMaxSec} s`),
    pass(frame0Std >= T.frame0MinStd, "frame0", "Quadro 0 não vazio", `desvio ${frame0Std.toFixed(1)}`, `pelo menos ${T.frame0MinStd}`),
    pass(endHeld >= endNeed, "end-card", "Tempo de leitura do cartão final", `${endHeld.toFixed(2)} s`, `pelo menos ${endNeed.toFixed(2)} s para "${facts.endCard.text}"`),
  ];
}

export function reportMarkdown(file, checks, compare) {
  const failed = checks.filter(c => !c.ok);
  const rows = checks.map(c => `| ${c.ok ? "passou" : "**FALHOU**"} | ${c.label} | ${c.measured} | ${c.limit} |`);
  const details = checks.filter(c => c.detail?.length).flatMap(c => [`\n### ${c.label}\n`, ...c.detail.map(d => `- ${d}`)]);
  return [
    `# QA de ${file}`,
    "",
    `**${failed.length ? `${failed.length} de ${checks.length} checagens falharam` : `As ${checks.length} checagens passaram`}.** Nada aqui substitui um ouvido humano: pronúncia, tom e naturalidade só o Diego aprova, e passar nesta lista não aprova o filme.`,
    "",
    "| Resultado | Checagem | Medido | Limite |",
    "|---|---|---|---|",
    ...rows,
    ...details,
    "",
    "## Para o ouvido",
    "",
    "A transcrição erra nome próprio com frequência, então estes trechos se conferem ouvindo, nunca por ela:",
    "",
    ...compare.names.map(n => `- ${n}`),
    ...compare.nameHunks.map(h => `- roteiro "${h.script.join(" ")}", ouvido "${h.heard.join(" ")}"`),
    "",
  ].join("\n");
}
