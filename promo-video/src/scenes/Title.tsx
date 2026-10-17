// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { AbsoluteFill, Img, interpolate, spring, staticFile, useCurrentFrame, useVideoConfig } from "remotion";
import { CrtOverlay } from "../components/Crt";
import { Lamps } from "../components/Lamps";
import { Terminal } from "../components/Terminal";
import { useLayout } from "../layout";
import { C, MONO, glow } from "../style";

const clamp = { extrapolateLeft: "clamp", extrapolateRight: "clamp" } as const;

// 24.7 to 30 s: the name, what it is, where to get it, and WOPR's last word from the film.
export const Title: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const t = frame / fps;
  const { portrait, u } = useLayout();

  const iconIn = spring({ frame: frame - 3, fps, config: { damping: 16, mass: 0.8 } });
  const name = "WOPR".slice(0, Math.max(0, Math.min(4, Math.floor((t - 0.1) / 0.09) + 1)));
  const at = (s: number) => interpolate(t, [s, s + 0.3], [0, 1], clamp);
  const rise = (s: number) => `translateY(${interpolate(at(s), [0, 1], [18 * u, 0])}px)`;

  return (
    <AbsoluteFill style={{ background: "radial-gradient(ellipse at 50% 40%, #0c1626 0%, #03050a 60%, #000 100%)" }}>
      <AbsoluteFill style={{ alignItems: "center", justifyContent: "center", opacity: 0.16, filter: `blur(${3 * u}px)` }}>
        <Lamps cols={portrait ? 14 : 26} rows={portrait ? 26 : 14} size={34 * u} seed="title" />
      </AbsoluteFill>
      <AbsoluteFill style={{ background: "radial-gradient(ellipse at center, rgba(0,0,0,0.7) 20%, rgba(0,0,0,0.2) 70%)" }} />

      <AbsoluteFill style={{ alignItems: "center", justifyContent: "center", flexDirection: "column", fontFamily: MONO, textAlign: "center" }}>
        <Img
          src={staticFile("img/wopr.svg")}
          style={{
            width: 190 * u,
            height: 190 * u,
            opacity: iconIn,
            transform: `scale(${interpolate(iconIn, [0, 1], [0.6, 1])})`,
            filter: "drop-shadow(0 0 30px rgba(255,176,32,0.45))",
            marginBottom: 30 * u,
          }}
        />
        <div
          style={{
            fontSize: (portrait ? 190 : 210) * u,
            fontWeight: 700,
            letterSpacing: "0.22em",
            marginRight: "-0.22em",
            color: C.bright,
            textShadow: glow(C.accent, 2.4),
            lineHeight: 1,
            minHeight: (portrait ? 190 : 210) * u,
          }}
        >
          {name}
        </div>
        <div style={{ fontSize: (portrait ? 28 : 30) * u, letterSpacing: "0.32em", color: C.text, textShadow: glow(C.accent, 0.6), marginTop: 22 * u, opacity: at(0.55) }}>
          WAR OPERATION PLAN RESPONSE
        </div>
        <div
          style={{
            display: "flex",
            flexDirection: portrait ? "column" : "row",
            gap: portrait ? 8 * u : 34 * u,
            fontSize: (portrait ? 46 : 44) * u,
            fontWeight: 700,
            color: C.bright,
            marginTop: 54 * u,
            letterSpacing: "0.06em",
            textShadow: glow(C.accent, 1),
          }}
        >
          <span style={{ opacity: at(1.0), transform: rise(1.0) }}>SIXTEEN GAMES</span>
          {portrait ? null : <span style={{ opacity: at(2.15), color: C.dim }}>·</span>}
          <span style={{ opacity: at(2.15), transform: rise(2.15) }}>FREE AND OPEN SOURCE</span>
        </div>
        <div style={{ fontSize: 30 * u, color: C.text, marginTop: 24 * u, letterSpacing: "0.12em", opacity: at(3.0) }}>
          LINUX · MACOS · WINDOWS
        </div>
        <div
          style={{
            fontSize: (portrait ? 34 : 36) * u,
            marginTop: 30 * u,
            padding: `${12 * u}px ${30 * u}px`,
            border: `${2 * u}px solid ${C.accent}`,
            color: C.bright,
            boxShadow: glow(C.accent, 0.8),
            textShadow: glow(C.accent, 0.6),
            opacity: at(3.15),
            transform: rise(3.15),
          }}
        >
          ghostofgoes.github.io/WOPR
        </div>
        <div style={{ marginTop: 50 * u, minHeight: 60 * u }}>
          <Terminal lines={[{ text: "HOW ABOUT A NICE GAME OF CHESS?", at: 3.7, cps: 31 }]} fontSize={(portrait ? 34 : 38) * u} />
        </div>
      </AbsoluteFill>
      <div
        style={{
          position: "absolute",
          bottom: (portrait ? 60 : 24) * u,
          width: "100%",
          textAlign: "center",
          fontFamily: MONO,
          fontSize: 17 * u,
          color: C.dim,
          opacity: at(0.6),
        }}
      >
        A fan-made homage to WarGames (1983). Not affiliated with or endorsed by its makers or owners.
      </div>
      <CrtOverlay strength={0.6} />
    </AbsoluteFill>
  );
};
