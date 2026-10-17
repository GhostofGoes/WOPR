// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { random, useCurrentFrame } from "remotion";

// WOPR's front panel: rows of amber and red lamps. Each lamp changes at most three times a
// second, wopr's own limit for anything that flashes.
export const Lamps: React.FC<{ cols: number; rows: number; size: number; seed: string; opacity?: number }> = ({
  cols,
  rows,
  size,
  seed,
  opacity = 1,
}) => {
  const frame = useCurrentFrame();
  const tick = Math.floor(frame / 10);
  const gap = size * 0.55;
  return (
    <div style={{ display: "grid", gridTemplateColumns: `repeat(${cols}, ${size}px)`, gap, opacity }}>
      {Array.from({ length: cols * rows }, (_, i) => {
        const red = random(`${seed}-colour-${i}`) < 0.18;
        const on = random(`${seed}-${i}-${tick + Math.floor(random(`${seed}-phase-${i}`) * 3)}`) < 0.45;
        const base = red ? "255,70,40" : "255,176,32";
        return (
          <div
            key={i}
            style={{
              width: size,
              height: size,
              borderRadius: "50%",
              background: on
                ? `radial-gradient(circle at 40% 35%, #fff8e0 0%, rgba(${base},1) 45%, rgba(${base},0.6) 100%)`
                : "radial-gradient(circle at 40% 35%, #4d473f 0%, #2c2925 60%, #1b1a18 100%)",
              boxShadow: on ? `0 0 ${size * 0.8}px rgba(${base},0.7)` : "inset 0 1px 2px rgba(0,0,0,0.8)",
            }}
          />
        );
      })}
    </div>
  );
};
