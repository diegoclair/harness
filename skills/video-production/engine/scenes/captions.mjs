// The caption layer: every line of a film wears it, so it is part of the player, not a scene a line picks.
export const CAPTION_HOLD = 0.3;

export function drawCaption(t, lines, firstChunk, box, wrap, ctxs) {
  // The feed's preview is frame 0, so the first line is already on it.
  const c = captionAt(lines, Math.max(t, firstChunk?.a ?? 0), CAPTION_HOLD);
  const owner = c ? ctxs.find(x => x.parts.some(part => part.line.id === c.line.id)) : null;
  const live = ctxs.find(x => t >= x.a && t < x.b);
  const visible = c && !c.chunk.onScreen && owner?.caption !== false && live?.caption !== false;
  wrap.style.opacity = visible ? 1 : 0;
  if (!visible) return;
  box.textContent = c.chunk.text;
  box.style.fontSize = owner.params.captionSize ? `${owner.params.captionSize}px` : "";
  wrap.style.top = owner.params.captionTop ? `${owner.params.captionTop}px` : "";
}
