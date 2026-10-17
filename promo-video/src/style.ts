// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

// Colours from wopr's own themes (internal/theme/theme.go): imsai, the default white phosphor, and
// norad for the war room.
export const C = {
  text: "#DCE6F0",
  bright: "#FFFFFF",
  dim: "#7A8694",
  accent: "#9EC5FF",
  red: "#FF3B30",
  redBG: "#C0001A",
  yellow: "#FFD60A",
  noradText: "#5AC8FA",
  amber: "#FFB020",
};

export const MONO = "'DejaVu Sans Mono', Menlo, Consolas, 'Liberation Mono', monospace";

export const glow = (color = C.accent, strength = 1) =>
  `0 0 ${4 * strength}px ${color}cc, 0 0 ${14 * strength}px ${color}66, 0 0 ${30 * strength}px ${color}33`;
