import { useCallback, useEffect, useMemo, useRef, useState } from "react";

export function HookHeavy() {
  const [a, setA] = useState(0);
  const [b, setB] = useState(0);
  const [c, setC] = useState(0);
  const [d, setD] = useState(0);
  const [e, setE] = useState(0);
  const box = useRef<HTMLDivElement>(null);
  const sum = useMemo(() => a + b + c + d + e, [a, b, c, d, e]);
  const bump = useCallback(() => setA(a + 1), [a]);
  const drop = useCallback(() => setB(b - 1), [b]);
  useEffect(() => setC(sum), [sum]);
  useEffect(() => setD(sum), [sum]);
  useEffect(() => setE(sum), [sum]);
  return (
    <output ref={box} onClick={bump} onDoubleClick={drop}>
      {sum}
    </output>
  );
}
