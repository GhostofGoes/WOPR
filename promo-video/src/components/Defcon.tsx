// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { C, MONO, glow } from "../style";

// The DEFCON ladder from wopr's big board: `>` and reverse video mark the level, so it reads
// without colour. Level 1 turns red.
export const Defcon: React.FC<{ level: number; size: number }> = ({ level, size }) => (
  <div style={{ fontFamily: MONO, fontSize: size, color: C.text, textShadow: glow(C.accent, 0.6), lineHeight: 1.1 }}>
    <div style={{ fontWeight: 700, letterSpacing: "0.08em", marginBottom: size * 0.4, textAlign: "center" }}>DEFCON</div>
    {[5, 4, 3, 2, 1].map((n) => {
      const on = n === level;
      const hot = on && n === 1;
      return (
        <div key={n} style={{ display: "flex", alignItems: "center", gap: size * 0.4, marginBottom: size * 0.32 }}>
          <span style={{ width: size * 0.7, color: hot ? C.red : C.bright, opacity: on ? 1 : 0 }}>&gt;</span>
          <span
            style={{
              width: size * 2.2,
              textAlign: "center",
              border: `${Math.max(2, size * 0.06)}px solid ${hot ? C.red : on ? C.bright : C.dim}`,
              background: hot ? C.redBG : on ? C.bright : "transparent",
              color: hot ? C.bright : on ? "#000" : C.dim,
              fontWeight: 700,
              boxShadow: hot ? `0 0 ${size}px ${C.red}` : on ? glow(C.accent, 1) : "none",
              textShadow: on ? "none" : undefined,
            }}
          >
            {n}
          </span>
        </div>
      );
    })}
  </div>
);
