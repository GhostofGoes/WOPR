// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { AbsoluteFill, Img, interpolate, spring, staticFile, useCurrentFrame, useVideoConfig } from "remotion";
import { Monitor } from "../components/Monitor";
import { useLayout } from "../layout";
import { C, MONO, glow } from "../style";
import { MONTAGE, SCENES } from "../timeline";

// The film's LIST GAMES, as WOPR prints it.
const GAMES = [
  "FALKEN'S MAZE", "BLACK JACK", "GIN RUMMY", "HEARTS", "BRIDGE", "CHECKERS", "CHESS", "POKER",
  "FIGHTER COMBAT", "GUERRILLA ENGAGEMENT", "DESERT WARFARE", "AIR-TO-GROUND ACTIONS",
  "THEATERWIDE TACTICAL WARFARE", "THEATERWIDE BIOTOXIC AND CHEMICAL WARFARE", "", "GLOBAL THERMONUCLEAR WAR",
];

// 7.6 to 12 s: the list of games on one side, and each game cutting in on the terminal as the
// narrator names it. The screenshots are wopr's own, at 80x24.
export const Montage: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const t = frame / fps + SCENES.montage[0];
  const { width, height, portrait, u } = useLayout();

  let cur = 0;
  MONTAGE.forEach((m, i) => {
    if (t >= m.at) cur = i;
  });
  const shot = MONTAGE[cur];
  const local = t - shot.at;
  const pop = spring({ frame: Math.round(local * fps), fps, config: { damping: 18, mass: 0.6, stiffness: 160 } });
  const side = cur % 2 === 0 ? 1 : -1;
  const monitorW = portrait ? width * 0.86 : width * 0.54;
  const listFont = portrait ? 25 * u : 27 * u;
  const listShown = Math.floor(interpolate(t - SCENES.montage[0], [0, 0.55], [0, GAMES.length], { extrapolateRight: "clamp" }));
  const nameSize = Math.min((portrait ? 78 : 92) * u, (monitorW * 1.55) / Math.max(shot.name.length, 8));

  const list = (
    <div style={{ fontFamily: MONO, fontSize: listFont, lineHeight: 1.42, whiteSpace: "pre", color: C.text }}>
      <div style={{ color: C.text, textShadow: glow(C.accent, 0.5), marginBottom: listFont * 0.6 }}>LIST GAMES</div>
      {GAMES.slice(0, listShown).map((g, i) => {
        const on = g !== "" && g === shot.name;
        return (
          <div
            key={i}
            style={{
              minHeight: listFont * 1.42,
              padding: `0 ${listFont * 0.3}px`,
              color: on ? "#000" : C.text,
              background: on ? C.bright : "transparent",
              textShadow: on ? "none" : glow(C.accent, 0.6),
              boxShadow: on ? glow(C.accent, 1.2) : "none",
              opacity: on ? 1 : 0.72,
            }}
          >
            {g}
          </div>
        );
      })}
    </div>
  );

  const screen = (
    <div style={{ transform: `rotateY(${side * interpolate(pop, [0, 1], [14, 7])}deg) rotateX(3deg) scale(${interpolate(pop, [0, 1], [0.9, 1])})` }}>
      <Monitor width={monitorW}>
        <Img
          src={staticFile(`img/${shot.img}`)}
          style={{ width: "100%", height: "100%", objectFit: "cover", transform: `scale(${1.02 + local * 0.03})`, filter: "brightness(1.15) contrast(1.05)" }}
        />
      </Monitor>
      <div
        style={{
          marginTop: 34 * u,
          textAlign: "center",
          fontFamily: MONO,
          fontWeight: 700,
          fontSize: nameSize,
          letterSpacing: "0.08em",
          color: C.bright,
          textShadow: glow(C.accent, 1.6),
          opacity: interpolate(pop, [0, 0.4], [0, 1], { extrapolateRight: "clamp" }),
          transform: `scale(${interpolate(pop, [0, 1], [1.25, 1])})`,
          whiteSpace: "nowrap",
        }}
      >
        {shot.name}
      </div>
    </div>
  );

  return (
    <AbsoluteFill style={{ background: "radial-gradient(ellipse at 60% 50%, #0b1420 0%, #020306 70%)" }}>
      <AbsoluteFill
        style={{
          flexDirection: portrait ? "column" : "row",
          alignItems: "center",
          justifyContent: "center",
          gap: portrait ? 50 * u : 70 * u,
          perspective: 2000 * u,
          padding: portrait ? `${height * 0.05}px 0` : 0,
        }}
      >
        {portrait ? null : list}
        {screen}
        {portrait ? <div style={{ transform: "scale(0.92)" }}>{list}</div> : null}
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
