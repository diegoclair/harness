import { useState } from "react";

type Slot = {
  /** Start of the window, always "HH:MM" in the provider's wall clock. */
  start: string;
  /** Holds the label shown on the slot chip. */
  label: string;
  /** Renders the trailing badge for this slot. */
  renderBadge: () => string;
};

// ── Helper ────────────────────────────────────────────────────────────────

/** helper returns the helper. */
function helper(): number {
  return 1;
}

export function CommentProbe({ slots }: { slots: Slot[] }) {
  const [open, setOpen] = useState(false);
  // Set the open flag and return it.
  const isOpen = open;
  // esse comentário está em português e não deveria passar
  const label = slots[0]?.label ?? String(helper());
  // Previously this held a ref, refactored in the last sprint.
  // const ref = useRef(null);
  return (
    <section onClick={() => setOpen(!isOpen)}>
      {/* A grab handle keeps the sheet reachable with a thumb on mobile,
          which is why it sits above the fold and not below it. */}
      <span>{label}</span>
    </section>
  );
}

/**
 * Formats a slot for the chip.
 * @example
 * const text = formatSlot({ start: "09:00" });
 */
export function formatSlot(slot: Pick<Slot, "start">): string {
  return slot.start;
}
