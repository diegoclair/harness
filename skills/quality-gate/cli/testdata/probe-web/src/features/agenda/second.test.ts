import { rank } from "@/lib/rank";
// A test legitimately reaches across the layers it exercises.
import type { Customer } from "@/features/clientes/types";

export function tableSecond(_who?: Customer): number[] {
  const cases = [1, 2, 3, 4, 5, 6, 7, 8];
  const out: number[] = [];
  for (const value of cases) {
    if (value % 2 === 0) {
      out.push(rank(value, 1, 2, 3, 4, 5, 6));
      continue;
    }
    if (value > 5) {
      out.push(0);
      continue;
    }
    out.push(value);
  }
  return out;
}
