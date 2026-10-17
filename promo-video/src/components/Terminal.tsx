// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { useCurrentFrame, useVideoConfig } from "remotion";
import { C, MONO, glow } from "../style";

// One line on the terminal. It types itself from `at` (seconds into the scene) at `cps`
// characters a second. `user` lines are the player's typing, in mixed case and plain colour.
// `then` is the player's answer on the same line, typed from its own time, as after `LOGON:  `.
export type Line = {
  text: string;
  at: number;
  cps?: number;
  user?: boolean;
  bright?: boolean;
  color?: string;
  then?: { text: string; at: number; cps: number };
};

const count = (text: string, at: number, cps: number, t: number) =>
  t < at ? -1 : Math.min(text.length, Math.floor((t - at) * cps));

// How many characters of the line show at time t, or -1 before it starts.
export const typed = (line: Line, t: number) => {
  const n = count(line.text, line.at, line.cps ?? 40, t);
  if (!line.then || n < line.text.length) return n;
  return n + Math.max(0, count(line.then.text, line.then.at, line.then.cps, t));
};

const full = (line: Line) => line.text + (line.then?.text ?? "");

// Lines type in order. The block cursor sits after the last character shown, and blinks at
// under 3 Hz, as wopr's own cursor does, only while nothing types.
export const Terminal: React.FC<{
  lines: Line[];
  fontSize: number;
  lineHeight?: number;
  showCursor?: boolean;
  style?: React.CSSProperties;
}> = ({ lines, fontSize, lineHeight = 1.45, showCursor = true, style }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const t = frame / fps;
  let cursorLine = -1;
  let typing = false;
  lines.forEach((l, i) => {
    const n = typed(l, t);
    if (n >= 0) {
      cursorLine = i;
      typing = n < full(l).length && (n < l.text.length || t >= (l.then?.at ?? 0));
    }
  });
  const blinkOn = typing || Math.floor(t * 2.4) % 2 === 0;
  return (
    <div style={{ fontFamily: MONO, fontSize, lineHeight, whiteSpace: "pre", ...style }}>
      {lines.map((l, i) => {
        const n = typed(l, t);
        const color = l.color ?? (l.user ? C.text : l.bright ? C.bright : C.text);
        return (
          <div key={i} style={{ minHeight: fontSize * lineHeight, color, textShadow: glow(l.color ?? C.accent, l.user ? 0.5 : 0.8) }}>
            {n > 0 ? l.text.slice(0, n) : ""}
            {n > l.text.length ? <span style={{ textShadow: glow(C.accent, 0.5) }}>{full(l).slice(l.text.length, n)}</span> : null}
            {showCursor && i === cursorLine && blinkOn ? (
              <span style={{ display: "inline-block", width: fontSize * 0.6, height: fontSize * 1.05, verticalAlign: "text-bottom", background: C.text, boxShadow: glow(C.accent, 0.8) }} />
            ) : null}
          </div>
        );
      })}
    </div>
  );
};
