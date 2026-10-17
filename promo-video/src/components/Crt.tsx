// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { AbsoluteFill, useCurrentFrame } from "remotion";

// Scanlines, a vignette and a slow roll bar over whatever is on screen.
export const CrtOverlay: React.FC<{ strength?: number }> = ({ strength = 1 }) => {
  const frame = useCurrentFrame();
  const roll = ((frame * 6) % 1400) - 200;
  return (
    <AbsoluteFill style={{ pointerEvents: "none" }}>
      <AbsoluteFill
        style={{
          background: "repeating-linear-gradient(to bottom, rgba(0,0,0,0) 0px, rgba(0,0,0,0) 2px, rgba(0,0,0,0.28) 3px, rgba(0,0,0,0) 4px)",
          opacity: strength,
        }}
      />
      <AbsoluteFill
        style={{
          background: `linear-gradient(to bottom, transparent ${roll - 120}px, rgba(158,197,255,0.035) ${roll}px, transparent ${roll + 120}px)`,
          opacity: strength,
        }}
      />
      <AbsoluteFill
        style={{ background: "radial-gradient(ellipse at center, rgba(0,0,0,0) 55%, rgba(0,0,0,0.75) 100%)", opacity: strength }}
      />
    </AbsoluteFill>
  );
};
