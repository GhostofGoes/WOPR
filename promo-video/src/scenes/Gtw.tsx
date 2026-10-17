// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { AbsoluteFill, Easing, Img, interpolate, spring, staticFile, useCurrentFrame, useVideoConfig } from "remotion";
import { Defcon } from "../components/Defcon";
import { Lamps } from "../components/Lamps";
import { Monitor } from "../components/Monitor";
import { useLayout } from "../layout";
import { C, MONO, glow } from "../style";

const clamp = { extrapolateLeft: "clamp", extrapolateRight: "clamp" } as const;

// 12 to 17.2 s: the title slams in, then the big board while DEFCON falls from 5 to 1.
export const Gtw: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const t = frame / fps;
  const { width, portrait, u } = useLayout();

  const slam = spring({ frame, fps, config: { damping: 14, mass: 0.7, stiffness: 180 } });
  const titleOut = interpolate(t, [1.05, 1.3], [1, 0], clamp);
  const boardShown = interpolate(t, [1.15, 1.45], [0, 1], clamp);
  const level = t < 1.6 ? 5 : t < 2.3 ? 4 : t < 3.0 ? 3 : t < 3.7 ? 2 : 1;
  const red = interpolate(t, [3.7, 4.2], [0, 1], clamp) * (0.75 + 0.25 * Math.cos((t - 3.7) * Math.PI * 2));
  const zoom = interpolate(t, [1.2, 5.2], [1, 1.22], { ...clamp, easing: Easing.bezier(0.45, 0, 0.55, 1) });

  const monitorW = portrait ? width * 0.9 : width * 0.66;
  const titleSize = portrait ? 74 * u : 104 * u;
  return (
    <AbsoluteFill style={{ background: "#000" }}>
      {/* The title card. */}
      <AbsoluteFill style={{ alignItems: "center", justifyContent: "center", opacity: titleOut }}>
        <div
          style={{
            fontFamily: MONO,
            fontWeight: 700,
            fontSize: titleSize,
            letterSpacing: "0.12em",
            color: C.bright,
            textAlign: "center",
            lineHeight: 1.2,
            textShadow: `${glow(C.red, 1.4)}`,
            transform: `scale(${interpolate(slam, [0, 1], [1.7, 1])})`,
            opacity: interpolate(slam, [0, 0.3], [0, 1], clamp),
          }}
        >
          GLOBAL{portrait ? <br /> : " "}THERMONUCLEAR{portrait || width < 1700 ? <br /> : " "}WAR
        </div>
      </AbsoluteFill>
      <AbsoluteFill style={{ background: "#fff", opacity: interpolate(t, [0, 0.03, 0.25], [0, 0.55, 0], clamp) }} />

      {/* The war room. */}
      <AbsoluteFill style={{ opacity: boardShown, background: "radial-gradient(ellipse at 50% 45%, #06122a 0%, #02060f 65%, #000 100%)" }}>
        <AbsoluteFill style={{ justifyContent: "space-between", flexDirection: "row", alignItems: "center", padding: `0 ${40 * u}px`, opacity: portrait ? 0 : 0.5 }}>
          <Lamps cols={3} rows={14} size={22 * u} seed="gtw-left" />
          <Lamps cols={3} rows={14} size={22 * u} seed="gtw-right" />
        </AbsoluteFill>
        <AbsoluteFill
          style={{
            flexDirection: portrait ? "column" : "row",
            alignItems: "center",
            justifyContent: "center",
            gap: 60 * u,
            perspective: 2400 * u,
          }}
        >
          <div style={{ transform: `rotateY(${portrait ? 0 : 6}deg) rotateX(2deg)` }}>
            <Monitor width={monitorW} glowColor="rgba(90,200,250,0.22)">
              <Img
                src={staticFile("img/global-thermonuclear-war.png")}
                style={{ width: "100%", height: "100%", objectFit: "cover", transform: `scale(${zoom})`, transformOrigin: "45% 30%", filter: "brightness(1.15)" }}
              />
              <div style={{ position: "absolute", inset: 0, background: `radial-gradient(ellipse at center, rgba(255,40,30,0) 40%, rgba(255,30,20,${0.35 * red}) 100%)`, mixBlendMode: "screen" }} />
            </Monitor>
          </div>
          <Defcon level={level} size={(portrait ? 46 : 42) * u} />
        </AbsoluteFill>
        <AbsoluteFill style={{ background: `radial-gradient(ellipse at center, rgba(0,0,0,0) 50%, rgba(192,0,26,${0.45 * red}) 100%)` }} />
        <div
          style={{
            position: "absolute",
            bottom: (portrait ? 70 : 26) * u,
            width: "100%",
            textAlign: "center",
            fontFamily: MONO,
            fontSize: 17 * u,
            color: C.dim,
          }}
        >
          The big board&apos;s map: Map (C) 1998 Matthew Thomas. Freely usable if this line is included.
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
