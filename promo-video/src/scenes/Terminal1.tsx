// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { AbsoluteFill, Easing, interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import { Monitor } from "../components/Monitor";
import { Terminal, type Line } from "../components/Terminal";
import { useLayout } from "../layout";

// 0 to 7.6 s: the dial-in, the logon and the greeting, on one terminal that the camera closes in
// on. CONNECTING... and CONNECTED. are wopr's own dial lines. The rest is the film's screen text.
const LINES: Line[] = [
  { text: "CONNECTING...", at: 0.7, cps: 30 },
  { text: "CONNECTED.", at: 2.25, cps: 40 },
  { text: "", at: 2.5 },
  { text: "LOGON:  ", at: 2.5, cps: 200, then: { text: "Joshua", at: 2.95, cps: 9 } },
  { text: "", at: 4.3 },
  { text: "GREETINGS PROFESSOR FALKEN.", at: 4.35, cps: 48 },
  { text: "", at: 4.95 },
  { text: "Hello.", at: 5.0, cps: 14, user: true },
  { text: "", at: 5.5 },
  { text: "SHALL WE PLAY A GAME?", at: 5.55, cps: 28, bright: true },
];

export const Terminal1: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const t = frame / fps;
  const { width, portrait, u } = useLayout();

  // The tube warms up: a line that opens into the screen.
  const open = interpolate(t, [0.05, 0.45], [0.004, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) });
  const flash = interpolate(t, [0.05, 0.2, 0.6], [0, 1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });

  // The camera: a slow dolly in, then a push onto the last line once WOPR asks.
  const ease = Easing.bezier(0.65, 0, 0.35, 1);
  const rotY = interpolate(t, [0, 4.3, 7.6], [-16, -7, 0], { extrapolateRight: "clamp", easing: ease });
  const rotX = interpolate(t, [0, 7.6], [5, 1], { extrapolateRight: "clamp" });
  const scale = interpolate(t, [0, 4.3, 6.4, 7.6], [0.82, 1.0, 1.1, 1.75], { extrapolateRight: "clamp", easing: ease });
  const shiftY = interpolate(t, [0, 6.4, 7.6], [0, 0, -0.17], { extrapolateRight: "clamp", easing: ease });
  const shiftX = interpolate(t, [0, 6.4, 7.6], [0, 0, 0.28], { extrapolateRight: "clamp", easing: ease });

  const monitorW = portrait ? width * 0.86 : width * 0.62;
  const screenH = monitorW * (564 / 960);
  const fontSize = monitorW / 38;
  return (
    <AbsoluteFill style={{ background: "radial-gradient(ellipse at 50% 60%, #0d1622 0%, #030509 60%, #000 100%)" }}>
      <AbsoluteFill style={{ alignItems: "center", justifyContent: "center", perspective: 2200 * u }}>
        <div style={{ transform: `translate(${shiftX * monitorW * scale}px, ${shiftY * screenH * scale}px) rotateX(${rotX}deg) rotateY(${portrait ? rotY * 0.35 : rotY}deg) scale(${scale})` }}>
          <Monitor width={monitorW}>
            <div style={{ position: "absolute", inset: 0, transform: `scaleY(${open})`, transformOrigin: "center" }}>
              <Terminal lines={LINES} fontSize={fontSize} style={{ padding: `${fontSize * 1.1}px ${fontSize * 1.4}px` }} />
            </div>
            <div style={{ position: "absolute", inset: 0, background: "#dfe9ff", opacity: flash * 0.5 }} />
          </Monitor>
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
