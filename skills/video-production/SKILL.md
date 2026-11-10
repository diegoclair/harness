---
name: video-production
version: 0.2.2
description: >-
  Produces short vertical videos — paid ads, feature videos, institutional pieces — from an approved script to a
  render a human only has to hear once: the script reviewed per variant before any voice exists, one synthetic
  narration per cut that plays whole while the scenes follow it, frame 0 and the cover treated as the feed preview,
  and two independent reviewers (script-reviewer before the voice, video-reviewer after the render) before anyone
  is shown the film. Use it WHENEVER a video, Reel, Short, anúncio, criativo, vídeo de função or vídeo institucional
  is scripted, voiced (TTS, locução, voz, narração), rendered, re-cut or fixed, or when a human says the audio
  "começou péssimo", a word "trava", the voice "mudou", a scene "passou rápido" or the end card is crooked — even
  when "video" is never said. Not for the words of the hook or the offer (that is sales-copy) nor for a web page
  (that is ui-ux).
---

# video-production

Born from one ad campaign (Nov 2026): six films were rejected by ear, one defect at a time, before all six were
approved in a day. Every defect came from the pipeline, not from the voice model: a narration cut into lines broke
words ("tx… t", "ap… p") and pushed the end of one sentence into the next; pauses squeezed to fit a fixed length
ran sentences together; lines generated apart sounded like different people; a body written for one hook was
pasted under another that never named the pain. What is here is the order that stops those.

## The rule under everything: the video adapts to the voice

A synthetic voice reads a whole script better than any edit of it. So the narration is **generated once per cut,
in one call, and played whole**; nothing cuts, stretches or trims its pauses. The pipeline only *locates* each line
inside it (from word timestamps of a transcription) and changes scene there. When a scene needs more time than the
voice gives it, the fix goes into the **script** — punctuation and the TTS's own pause tags — or into the scene's
own timing, never into the audio. A fixed target duration is the same mistake in disguise: the film lasts what its
speech needs.

## The order

1. **Words first, from sales-copy.** The hook, the claim and the offer come from the sales-copy skill and the
   project's brand and message docs. Read sales-copy's short-video reference for the shape of the piece.
2. **Script per variant, as speech.** Write each cut as the transcript a person would say, then read each variant
   **whole, from its own hook to its close**: a body written under one hook rarely survives under another. Read
   `references/script.md` while writing it.
3. **script-reviewer before any voice.** Dispatch the `script-reviewer` agent with every variant's full spoken
   text and the project's brand rules. A rejected line costs nothing now and a regeneration plus a review later.
4. **Voice by ear, then narration.** Pick the voice with the human from samples heard side by side, then generate
   one narration per cut. Read `references/narration.md` before choosing a voice, writing a TTS style or asking
   for another take.
5. **Scenes follow the narration.** Cuts land where the voice changes line; captions follow its words; anything a
   scene animates after a line (a tap, a confirmation) happens inside the time the voice leaves, and is seen long
   enough to read.
6. **Frame 0 and the cover.** Read `references/frame-and-cover.md` before the first render that will be shown.
7. **video-reviewer before the human.** Dispatch the `video-reviewer` agent on each render with its script. It runs
   `scripts/review.py` and looks at the frames itself. Fix what it rejects, render, and send it again with its
   context kept. Only an approved render goes to the human — and the human's ear is still the last word on tone.
8. **Keep what was approved.** Before regenerating anything an approved film depends on (voice, style, a shared
   line), copy the approved render aside: a new take changes every line, including the ones nobody asked to change.

## What a change costs

Any change to the spoken text, the voice or its style is a new narration for every cut that shares it. Say that to
the human before the change, with the price (TTS for a 20 s spot costs well under a cent), and re-review every
affected cut. A change to a scene, a caption chunk or the music touches no voice and costs a render.

## The engine and the kit

The engine ships with this skill, in `engine/`: run it as `node ~/.claude/skills/video-production/engine/film.mjs`.
`film setup` fetches once per machine what it needs outside the skill folder; `film run <film folder>` renders a film,
which is a `film.json` and its approved narration in `voice/`. What belongs to the project (scenes, brand, voices,
pronunciation) is the repo's kit, `video-kit/kit.json` in the nearest folder above the film; read the kit's README and
voice doc before touching a narration, and `engine/README.md` for the film and kit contract. `scripts/review.py` reads
only the rendered file and its script.

Revalidate when: a TTS model changes how it reads tags or styles, the engine stops narrating whole, or a defect
reaches the human that neither reviewer would have caught.
