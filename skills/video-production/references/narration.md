# Voice and narration

Read this before choosing a voice, writing a TTS style, generating a narration or asking for another take.

## Choosing the voice

- **By ear, side by side, in a page the human can open.** Generate the same full line (better: the whole script)
  in 3–4 candidates and put them on one HTML page with players; many people have no audio app installed. A list of
  voice names decides nothing.
- **Filter the voice library by language and persona first.** A library entry's accent field is often empty; the
  accent the human hears is judged by ear, never read from metadata.
- **Timbre and delivery are separate knobs.** When one voice has the timbre and another the delivery, keep the
  timbre's voice and describe the other's delivery in the style — one generation tests it.
- **Compare like with like.** A candidate without music against the approved one with music is not a fair test.

## The style instruction

- **Short, and about who is speaking and how** — a persona and a delivery. Start from an empty style when the
  provider recommends it.
- **No meta-rules inside it** ("say every word exactly as written", "never say X"): they made the voice stall and
  stretch words. A word the voice gets wrong is fixed in the script, scoped to that voice.

## The narration

- **One call per cut, the whole script, played whole.** Each call picks a new delivery, so a line generated alone
  sounds like another person. Separate lines with the provider's pause tag; a cut with two hooks in one narration
  makes the voice read both.
- **Never cut, stretch or squeeze it.** Locate lines by word timestamps of a transcription; a line made only of
  figures transcribes as digits and is placed between its neighbours.
- **A take is for the whole cut.** When a take stumbles on a word, ask another and compare by measure (the
  reviewer's report), then by ear. The provider may add a vocalisation of its own; keep it only if the human likes it.
- **Before a change that regenerates an approved narration, copy the approved render aside** and say so.
