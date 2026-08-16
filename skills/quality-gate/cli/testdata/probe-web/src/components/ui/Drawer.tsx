import type { ReactNode } from "react";

export function Drawer({ children }: { children: ReactNode }) {
  return <dialog>{children}</dialog>;
}
