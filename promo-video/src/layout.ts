// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import { useVideoConfig } from "remotion";

// The same scenes serve 16:9 and 9:16. `u` scales sizes so that 1 is a pixel at 1080 short side.
export const useLayout = () => {
  const { width, height } = useVideoConfig();
  const portrait = height > width;
  return { width, height, portrait, u: Math.min(width, height) / 1080 };
};
