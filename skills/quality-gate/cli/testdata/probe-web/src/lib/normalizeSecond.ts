export function normalizeSecond(values: string[]): string[] {
  const out: string[] = [];
  for (const value of values) {
    const trimmed = value.trim();
    if (trimmed === "") {
      continue;
    }
    if (trimmed.startsWith("#")) {
      continue;
    }
    let lower = trimmed.toLowerCase();
    if (lower.length > 64) {
      lower = lower.slice(0, 64);
    }
    out.push(lower);
  }
  return out;
}
