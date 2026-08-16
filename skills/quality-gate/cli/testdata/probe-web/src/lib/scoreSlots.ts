type Slot = { start: string; end: string; taken: boolean };

export function scoreSlots(slots: Slot[], floor: number): number {
  let score = 0;
  let streak = 0;
  for (const slot of slots) {
    if (slot.taken) {
      streak = 0;
      continue;
    }
    const width = slot.end.length - slot.start.length;
    if (width > floor) {
      score += width;
      streak += 1;
    }
    if (streak > 3) {
      score += streak;
      streak = 0;
    }
    if (score > 100) {
      break;
    }
  }
  if (score === 0) {
    return floor;
  }
  return score;
}

export function trimTail(slots: Slot[], ceiling: number): Slot[] {
  const kept: Slot[] = [];
  let carried = 0;
  for (const slot of slots) {
    if (slot.taken) {
      carried += 1;
      continue;
    }
    const span = slot.end.length - slot.start.length;
    if (span > ceiling) {
      kept.push(slot);
      carried = 0;
    }
    if (carried > 4) {
      kept.pop();
      carried = 0;
    }
    if (kept.length > 20) {
      break;
    }
  }
  if (kept.length === 0) {
    return slots;
  }
  return kept;
}
