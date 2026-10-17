// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { AbsoluteFill, Audio, interpolate, Sequence, staticFile } from "remotion";
import { Climax } from "./scenes/Climax";
import { Gtw } from "./scenes/Gtw";
import { Montage } from "./scenes/Montage";
import { Strange } from "./scenes/Strange";
import { Terminal1 } from "./scenes/Terminal1";
import { Title } from "./scenes/Title";
import { SCENES, VO, f } from "./timeline";

const Scene: React.FC<{ span: readonly [number, number]; children: React.ReactNode }> = ({ span, children }) => (
  <Sequence from={f(span[0])} durationInFrames={f(span[1]) - f(span[0])}>
    {children}
  </Sequence>
);

// One sound: `at` and `len` in seconds, `from` skips into the file, `vol` is its level.
const Sfx: React.FC<{ src: string; at: number; len?: number; from?: number; vol?: number | ((frame: number) => number) }> = ({
  src,
  at,
  len = 10,
  from = 0,
  vol = 1,
}) => (
  <Sequence from={f(at)} durationInFrames={f(len)} layout="none">
    <Audio src={staticFile(`audio/${src}.mp3`)} startFrom={f(from)} volume={vol} />
  </Sequence>
);

// The score ducks under the narrator, cuts dead at the climax, and the title's swell takes over.
const scoreVolume = (frame: number) => {
  const t = frame / 30;
  return interpolate(t, [0, 0.4, 7.4, 8.0, 12.0, 19.6, 20.15, 20.2], [0, 0.32, 0.32, 0.5, 0.5, 0.62, 0.62, 0], {
    extrapolateRight: "clamp",
  });
};

export const Promo: React.FC = () => (
  <AbsoluteFill style={{ background: "#000" }}>
    <Sequence from={0} durationInFrames={f(SCENES.greeting[1])}>
      <Terminal1 />
    </Sequence>
    <Scene span={SCENES.montage}>
      <Montage />
    </Scene>
    <Scene span={SCENES.gtw}>
      <Gtw />
    </Scene>
    <Scene span={SCENES.climax}>
      <Climax />
    </Scene>
    <Scene span={SCENES.strange}>
      <Strange />
    </Scene>
    <Scene span={SCENES.title}>
      <Title />
    </Scene>

    {/* Score */}
    <Sfx src="score" at={0} len={20.2} vol={scoreVolume} />
    <Sfx src="sfx-drone" at={20.35} len={4.6} vol={(fr) => interpolate(fr, [0, 15, 110, 138], [0, 0.07, 0.07, 0], { extrapolateRight: "clamp" })} />
    <Sfx src="score-title" at={22.3} len={7.7} vol={(fr) => interpolate(fr, [0, 20, 200, 231], [0.15, 0.6, 0.6, 0], { extrapolateRight: "clamp" })} />

    {/* Dial-in */}
    <Sfx src="sfx-crt" at={0} vol={0.55} />
    <Sfx src="sfx-dial" at={0.5} vol={0.35} />
    <Sfx src="sfx-modem" at={1.55} len={0.8} vol={(fr) => interpolate(fr, [0, 18, 24], [0.22, 0.22, 0], { extrapolateRight: "clamp" })} />
    <Sfx src="sfx-keys" at={2.95} len={0.7} vol={0.45} />
    <Sfx src="sfx-keys" at={4.35} len={0.6} from={1} vol={0.35} />
    <Sfx src="sfx-keys" at={5.0} len={0.45} from={2} vol={0.4} />
    <Sfx src="sfx-keys" at={5.55} len={0.75} from={0.5} vol={0.35} />
    <Sfx src="sfx-boom" at={5.5} vol={0.45} />

    {/* Montage */}
    <Sfx src="sfx-keys" at={7.6} len={0.55} from={1.5} vol={0.3} />
    <Sfx src="sfx-chirps" at={7.7} len={3} vol={0.16} />
    <Sfx src="sfx-riser" at={9.5} len={2.5} vol={0.4} />

    {/* Global Thermonuclear War */}
    <Sfx src="sfx-boom" at={12.0} vol={0.85} />
    <Sfx src="sfx-klaxon" at={15.7} len={2.3} vol={0.3} />

    {/* Climax */}
    <Sfx src="sfx-chirps" at={17.2} len={3.0} vol={0.28} />
    <Sfx src="sfx-cut" at={20.15} vol={0.6} />

    {/* Ending */}
    <Sfx src="sfx-keys" at={20.62} len={0.9} vol={0.3} />
    <Sfx src="sfx-keys" at={21.86} len={1.2} from={1} vol={0.3} />
    <Sfx src="sfx-keys" at={23.44} len={0.67} from={2.4} vol={0.3} />
    <Sfx src="sfx-boom" at={24.8} vol={0.5} />
    <Sfx src="sfx-keys" at={28.4} len={1.0} from={0.3} vol={0.3} />

    {/* Voices */}
    {Object.entries(VO).map(([id, at]) => (
      <Sfx key={id} src={`vo-${id}`} at={at} len={6} />
    ))}
  </AbsoluteFill>
);
