// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import { CrtOverlay } from "../components/Crt";
import { Terminal, type Line } from "../components/Terminal";
import { useLayout } from "../layout";

// 20.2 to 24.7 s: silence, then WOPR's verdict, typed with its voice. The film's lines and line
// breaks, exactly.
const LINES: Line[] = [
  { text: "A STRANGE GAME.", at: 0.22, cps: 17 },
  { text: "THE ONLY WINNING MOVE IS", at: 1.66, cps: 20 },
  { text: "NOT TO PLAY.", at: 3.24, cps: 18, bright: true },
];

export const Strange: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const t = frame / fps;
  const { portrait, u } = useLayout();
  const scale = interpolate(t, [0, 4.5], [1.04, 1], { extrapolateRight: "clamp" });
  const fade = interpolate(t, [4.2, 4.5], [1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
  return (
    <AbsoluteFill style={{ background: "#000" }}>
      <AbsoluteFill style={{ alignItems: "center", justifyContent: "center", opacity: fade, transform: `scale(${scale})` }}>
        <Terminal lines={LINES} fontSize={(portrait ? 50 : 72) * u} lineHeight={1.5} />
      </AbsoluteFill>
      <CrtOverlay strength={0.8} />
    </AbsoluteFill>
  );
};
