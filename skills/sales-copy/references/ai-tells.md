# The marks of machine-written Portuguese

Read this before showing the human any caption, speech, comment or post. A short text read out loud or in a feed shows
a written-by-formula line louder than a long one does: the reader scrolls past something that sounds like a brand, and
the voice stumbles on a sentence nobody would say.

## Run the check

```bash
python3 scripts/tells.py draft.txt
```

It reads `scripts/tells.json`. **FAIL is fixed every time**: invisible characters, dashes and curly quotes, which no
keyboard types and which survive copy-paste. **WARN is decided out loud**: a stock phrase or a structure that sounds
generated. The lexicon is meant to grow: a phrase the human rejects as "cara de IA" enters it in the same session.

## What the script cannot see, and the writer fixes by hand

- **"Não é só X, é Y"** and every staged reveal ("O resultado?", "Sabe o melhor?").
- **Triads by reflex**: three adjectives or three benefits where one carries the line.
- **Every sentence the same length.** People mix a long sentence with a three-word one.
- **No number, no name, no concrete thing** in a paragraph: abstract nouns are the strongest tell.
- **Written connectors in speech**: "além disso", "dessa forma", "portanto". Speech says "e", "aí", "só que".
- **Bait asks**: "segue pra mais", "marca alguém", "salva esse post" with no reason to.

The score is not the goal and nothing here promises a detector's verdict. The goal is a text a seller would say to
another seller.

---

The idea of a lexicon plus a structural checklist is adapted from the humanizer of instagram-agent-skill (MIT, Copyright
(c) 2026 Jake Schincariol); the full notice is at the end of `hooks.md`. The Portuguese lexicon is our own.
