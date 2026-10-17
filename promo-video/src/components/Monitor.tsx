// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { CrtOverlay } from "./Crt";

// A terminal's CRT in a moulded bezel, drawn at the given width (the screen is 16:9.4, like
// wopr's 80x24 screenshots). Callers tilt it with a CSS transform.
export const Monitor: React.FC<{ width: number; children: React.ReactNode; glowColor?: string }> = ({
  width,
  children,
  glowColor = "rgba(158,197,255,0.18)",
}) => {
  const screenH = width * (564 / 960);
  const bezel = width * 0.045;
  return (
    <div
      style={{
        width: width + bezel * 2,
        padding: bezel,
        borderRadius: bezel * 1.1,
        background: "linear-gradient(160deg, #3a3d42 0%, #24262a 45%, #141517 100%)",
        boxShadow: `0 40px 120px rgba(0,0,0,0.85), inset 0 2px 0 rgba(255,255,255,0.08), 0 0 160px ${glowColor}`,
      }}
    >
      <div
        style={{
          position: "relative",
          width,
          height: screenH,
          borderRadius: width * 0.03,
          overflow: "hidden",
          background: "#000",
          boxShadow: "inset 0 0 60px rgba(0,0,0,0.9)",
        }}
      >
        {children}
        <CrtOverlay />
        <div
          style={{
            position: "absolute",
            inset: 0,
            background: "linear-gradient(155deg, rgba(255,255,255,0.07) 0%, rgba(255,255,255,0) 35%)",
          }}
        />
      </div>
    </div>
  );
};
