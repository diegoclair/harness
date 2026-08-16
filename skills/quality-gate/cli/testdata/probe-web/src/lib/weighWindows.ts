type Window = { from: string; to: string; busy: boolean };

export function weighWindows(windows: Window[], base: number): number {
  let weight = 0;
  let run = 0;
  for (const window of windows) {
    if (window.busy) {
      run = 0;
      continue;
    }
    const span = window.to.length - window.from.length;
    if (span > base) {
      weight += span;
      run += 1;
    }
    if (run > 3) {
      weight += run;
      run = 0;
    }
    if (weight > 100) {
      break;
    }
  }
  if (weight === 0) {
    return base;
  }
  return weight;
}

export function clipHead(entrys: Slot[], cap: number): Slot[] {
  const held: Slot[] = [];
  let ran = 0;
  for (const entry of entrys) {
    if (entry.taken) {
      ran += 1;
      continue;
    }
    const width = entry.end.length - entry.start.length;
    if (width > cap) {
      held.push(entry);
      ran = 0;
    }
    if (ran > 4) {
      held.pop();
      ran = 0;
    }
    if (held.length > 20) {
      break;
    }
  }
  if (held.length === 0) {
    return entrys;
  }
  return held;
}
