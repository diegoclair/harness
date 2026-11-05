# Frame 0, the cover and the end card

Read this before the first render that will be shown, and whenever the first second or the last card changes.

## Two previews, two jobs

- **Frame 0 is what the feed shows when the video starts on its own.** The hook's words are already on screen and
  readable there, and light up as they are said. No effect (confetti, a flash, a wipe) covers the headline on frame
  0, and any effect stays behind the text afterwards.
- **The cover is an image of its own, not necessarily a frame of the video.** It shows where the video is not
  playing: the profile grid, a share preview. It carries the whole hook, including the turn the video saves for
  second three. Check the platform's current docs for whether a custom thumbnail can be uploaded for that placement.

## Placement and centring

- **The safe zone comes from the platform's current docs, read at render time**, never from memory: it moves.
- **Only what really collides with the platform's overlays leaves room for them.** A card that stands alone on
  screen is centred with equal side margins; shifting everything to one side for a button rail that sits elsewhere
  reads as a crooked layout.
- **A headline longer than its box shrinks to one line** instead of wrapping onto what sits below it.
