// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { AbsoluteFill, interpolate, random, useCurrentFrame, useVideoConfig } from "remotion";
import { Lamps } from "../components/Lamps";
import { useLayout } from "../layout";
import { C, MONO, glow } from "../style";

const CODE = "CPE1704TKS"; // the film's launch code
const CHARS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

// A tic-tac-toe board that fills one move at a time, from a seeded order.
const Board: React.FC<{ seed: number; t: number; size: number }> = ({ seed, t, size }) => {
  const order = Array.from({ length: 9 }, (_, i) => i).sort((a, b) => random(`ttt-${seed}-${a}`) - random(`ttt-${seed}-${b}`));
  const moves = Math.max(0, Math.min(9, Math.floor(t * 9)));
  const cells: string[] = Array(9).fill(" ");
  order.slice(0, moves).forEach((c, i) => {
    cells[c] = i % 2 === 0 ? "X" : "O";
  });
  const row = (r: number) => ` ${cells[r * 3]} | ${cells[r * 3 + 1]} | ${cells[r * 3 + 2]} `;
  return (
    <div style={{ fontFamily: MONO, fontSize: size, lineHeight: 1.15, whiteSpace: "pre", color: C.text, textShadow: glow(C.accent, 0.5) }}>
      {row(0)}
      {"\n---+---+---\n"}
      {row(1)}
      {"\n---+---+---\n"}
      {row(2)}
    </div>
  );
};

// 17.2 to 20.2 s: DEFCON 1. WOPR plays tic-tac-toe against itself while the launch code cracks,
// one character after another. The cut to black comes before the last one.
export const Climax: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const t = frame / fps;
  const { portrait, u } = useLayout();

  const cols = portrait ? 4 : 7;
  const rows = portrait ? 8 : 4;
  const boardSize = (portrait ? 24 : 22) * u;
  const pulse = 0.6 + 0.4 * Math.cos(t * Math.PI * 2);
  const codeSize = (portrait ? 62 : 96) * u;

  return (
    <AbsoluteFill style={{ background: "#000" }}>
      <AbsoluteFill
        style={{
          display: "grid",
          gridTemplateColumns: `repeat(${cols}, 1fr)`,
          gridTemplateRows: `repeat(${rows}, 1fr)`,
          placeItems: "center",
          padding: 40 * u,
          opacity: 0.42,
        }}
      >
        {Array.from({ length: cols * rows }, (_, i) => (
          <Board key={i} seed={i} size={boardSize} t={t * 1.4 - random(`start-${i}`) * 1.2} />
        ))}
      </AbsoluteFill>
      <AbsoluteFill style={{ background: `radial-gradient(ellipse at center, rgba(0,0,0,0.92) 25%, rgba(0,0,0,0.4) 60%, rgba(192,0,26,${0.5 * pulse}) 100%)` }} />
      <AbsoluteFill style={{ alignItems: "center", justifyContent: "center", flexDirection: "column", gap: 40 * u }}>
        <div style={{ display: "flex", gap: 30 * u, alignItems: "center" }}>
          {portrait ? null : <Lamps cols={4} rows={3} size={20 * u} seed="climax-l" />}
          <div
            style={{
              fontFamily: MONO,
              fontWeight: 700,
              fontSize: 40 * u,
              padding: `${6 * u}px ${24 * u}px`,
              background: C.redBG,
              color: C.bright,
              letterSpacing: "0.1em",
              boxShadow: `0 0 ${40 * u * pulse}px ${C.red}`,
            }}
          >
            DEFCON 1
          </div>
          {portrait ? null : <Lamps cols={4} rows={3} size={20 * u} seed="climax-r" />}
        </div>
        <div style={{ fontFamily: MONO, fontSize: 36 * u, color: C.text, letterSpacing: "0.2em", textShadow: glow(C.accent, 0.8) }}>LAUNCH CODE</div>
        <div style={{ display: "flex", gap: 10 * u }}>
          {CODE.split("").map((ch, i) => {
            const lockAt = 0.2 + i * 0.3;
            const locked = i < 9 && t >= lockAt;
            const shown = locked ? ch : CHARS[Math.floor(random(`code-${i}-${Math.floor(frame / 2)}`) * CHARS.length)];
            const justLocked = locked ? interpolate(t - lockAt, [0, 0.25], [1, 0], { extrapolateRight: "clamp" }) : 0;
            return (
              <div
                key={i}
                style={{
                  fontFamily: MONO,
                  fontWeight: 700,
                  fontSize: codeSize,
                  width: codeSize * 0.72,
                  textAlign: "center",
                  color: locked ? C.bright : C.dim,
                  borderBottom: `${4 * u}px solid ${locked ? C.bright : C.dim}`,
                  textShadow: locked ? glow(justLocked > 0 ? C.red : C.accent, 1 + justLocked * 1.5) : "none",
                }}
              >
                {shown}
              </div>
            );
          })}
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
