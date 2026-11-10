# The video engine

**A film is a page with a deterministic timeline: `render(t)` draws the frame of second `t`, and the same cue object
that marks the cuts drives the score and the mix.** Moving a time moves picture, music and voice together, and the same
command gives the same file, so nothing is edited in a video editor.

Three layers, each with one owner:

| Layer | Lives in | Holds |
|---|---|---|
| engine | this folder, shipped with the skill | runner, server, player, voice, alignment, mix, instruments, default score, QA, footage tools, captions |
| kit | `video-kit/` in the repo that makes films | `kit.json`, brand styles, scenes, pronunciation dictionary, approved voices, prepare script, film template |
| film | a folder with `film.json` | the lines, scene params, variants, the approved narration in `voice/`, optionally a `music.py` |

**The only override rule: a kit scene with the same name as an engine scene wins.** The engine ships no branded scene.

## Commands

```sh
F=~/.claude/skills/video-production/engine/film.mjs
node $F setup                        # once per machine: Node deps, Chromium, the Python venv and the speech model
node $F new <folder> --title "..."   # a film from the kit's template
node $F scenes <folder>              # what the kit offers, its params and where its words come from
node $F run <folder>                 # voice measure render music mix mux transcribe sheet qa
node $F run <folder> --from mix      # resume from a step; --to stops after one
node $F run <folder> --query variant=v2
node $F run <folder> --no-new-voice  # same as FILM_NO_NEW_VOICE=1: a missing narration fails naming its file
node $F serve <folder>               # watch and hear it in a browser
node $F test <kit folder>            # the kit's tests against this engine
```

`film setup` is idempotent and writes only under `~/.cache/video-production/` (a reinstall wipes the skill folder) and
the tools' own caches (Chromium, the speech model). It needs `node`, `npm`, `python3` and `ffmpeg` on the machine.

## The kit

The engine finds `video-kit/kit.json` in the nearest folder above the film. Paths in it are relative to the kit.

| Key | Meaning |
|---|---|
| `root` | the folder the server serves films from; without it, the film's git repository; `--root` wins over both |
| `mounts` | extra URL prefixes served from other folders, `{ "/ui/": "../../ui" }` |
| `env`, `keys` | an untracked file with API keys, and the name it keeps each provider key under (`OPENAI_API_KEY`, `GEMINI_API_KEY` by default); the environment wins |
| `prepare` | a Node script run in the film folder before any page opens, for assets the scenes read from it |
| `scenes` | the module exporting `SCENES`, by name |
| `styles` | the brand stylesheet: fonts, tokens, `--caption-font`, utility classes |
| `svg` | `{ selector: url }`: svg markup put inline into every matching element, so it takes the text colour |
| `dictionary` | spellings per voice and the takes that worked, for the text sent to the voice |
| `voice`, `voices` | the default voice and every approved one: `model`, `instructions` |
| `template` | the `film.json` that `film new` starts from |
| `output`, `lang`, `title`, `music` | output file prefix, page language, page title, default score script |

The page serves the engine at `/__engine/` and the kit at `/__kit/`; scenes import by those paths, and in Node the same
paths resolve through a module hook, so a scene runs unchanged in the page and in the kit's tests.

## The film

```json
{
  "title": "…", "safe": true, "target": 20, "voice": { "voice": "…", "take": 3 },
  "narration": { "file": "voice/<key>.wav", "spans": "voice/<key>.spans.json" },
  "lines": [{ "id": "hook", "text": "[Shown]|said", "scene": "…", "params": {}, "say": "…", "lead": 0, "tail": 0.2, "min": 3, "fit": { "x": 108, "y": 120, "scale": 0.8 }, "caption": false }],
  "variants": { "v2": { "narration": { "file": "…", "spans": "…" }, "hook": { "text": "…", "scene": "…" }, "also": null } }
}
```

- **One line, one scene.** `text` is the caption: `|` splits chunks, `[]` marks what the screen already shows. `continues`
  extends the scene before, so one phone can span three lines. `fit` scales one scene into a box; `safe` does it for every
  scene, inside what Reels leaves visible (a scene's `safeFit` is its own box there). `"caption": false` silences the
  scene's lines and hides a caption held from the line before while it is on screen.
- **A hook test is a variant.** `variants.<name>.<id>` replaces that line, `null` drops it, and `narration` names the
  variant's own voice; a variant never inherits the base film's narration.
- **A scene spanning two visuals inside one line handles both**, keyed to the line's chunks, and emits in `cues()` every
  key its film's music reads. Cue arrays merge in scene order.

## The narration

**The video adapts to the voice.** A narration is played whole, never cut, stretched or pause-trimmed; the engine only
locates each line in it, and the scene changes there.

- **A pinned narration (`narration` in `film.json`) is used as is, with its spans,** whatever the kit's voices say:
  changing a voice's style never re-voices an approved film. No voice call, no transcription.
- **Without a pin,** the engine looks for `voice/<key>.wav` and `voice/<key>.spans.json` in the film, then
  `~/.cache/video-production/voice/<key>.wav`, and only then calls the voice, keeping the audio in the cache and both files
  in the film. The key hashes model, voice, instructions, spoken text and take, so a new word pays for its narration only.
- **A voice that changes its delivery on every call is read in one call for the whole film;** a voice that does not is
  read line by line and may be paced up to 1.05x (`tempo`, `maxPause`).
- **The film's `.gitignore` versions `voice/*.wav` and `voice/*.json`** and nothing else it renders.
- **Once `narration` is pinned, `voice/` holds only the pinned files** (base and variants): every other take leaves the
  film on the next run and its audio stays in the cache, so a take search never reaches git.

## Rhythm

- **An event falls on the 0.25 s grid and a scene change on a beat** (0.5 s at 120 bpm).
- **Text stays on screen long enough to read**: about 3 words a second, plus 1 s.
- **A narrated film follows its narration:** each scene starts where its line starts in the voice; a scene that needs
  more time asks for a pause in the script, never a moved word.
- **Frame 0 already shows the piece**, because the feed uses it as the preview.
- **A page that loses an asset fails the step:** any failed request, 4xx/5xx answer or page error during capture.

## Mix and loudness

- **The music steps back by the timeline, not by a voice detector**: down a little before the first syllable, back after
  the last, the same on every render.
- **The mix lands at -14 LUFS, true peak under -1 dBTP**, with the peaks held before the one flat gain.
- **Chromium does not draw every frame identically twice:** compare versions by their audio and still frames, not by the file.

## QA

- **Every limit lives in `qa-thresholds.mjs`.** A check that disagrees with a film is discussed there, never inside the check.
- **A proper name never passes or fails by the transcript**; it goes to the list for the ear. Passing every check approves
  nothing: tone and pronunciation are a human's call.
- The report is `.build/qa.md`; the contact sheet is `.build/sheet.jpg`.

## Real footage

**`footage.py` keeps only what every `clips.py` repeats: cut a shot, blur regions, sample every 0.5 s.** A raw source inside
a repository and an output git would track are refused.
