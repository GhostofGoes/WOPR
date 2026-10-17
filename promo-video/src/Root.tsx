// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { Composition } from "remotion";
import { Promo } from "./Promo";
import { DURATION, FPS } from "./timeline";

export const RemotionRoot: React.FC = () => (
  <>
    <Composition id="Promo-Landscape" component={Promo} durationInFrames={DURATION} fps={FPS} width={1920} height={1080} />
    <Composition id="Promo-Portrait" component={Promo} durationInFrames={DURATION} fps={FPS} width={1080} height={1920} />
  </>
);
