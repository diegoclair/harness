export function rank(a: number, b: number, c: number, d: number, e: number, f: number, g: number): number {
  let total = 0;
  for (let i = 0; i < a; i++) {
    if (i % 2 === 0) {
      for (let j = 0; j < b; j++) {
        if (j % 3 === 0) {
          for (let k = 0; k < c; k++) {
            if (k > d) {
              total += e + f + g;
            }
          }
        }
      }
    }
  }
  return total;
}

export function tier(n: number, label: string): string {
  const low =
    n === 1 || n === 2 || n === 3 || n === 4 || n === 5 ||
    n === 6 || n === 7 || n === 8 || n === 9 || n === 10;
  const mid = n > 10 && n < 21 && label !== "" && label !== "skip";
  const high = n > 20 && (label === "peak" || label === "top" || n % 2 === 0);
  if (low) return "low";
  if (mid) return "mid";
  if (high) return "high";
  for (const ch of label) {
    if (ch === "!" || ch === "?") return "flag";
  }
  return "top";
}
