// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

// Every cue in the promo, in seconds. The voice lines' word timings come from ElevenLabs'
// character alignment (audio-src/vo-*.json), so the cuts land on the words.

export const FPS = 30;
export const f = (seconds: number) => Math.round(seconds * FPS);
export const DURATION = f(30);

export const SCENES = {
  dial: [0, 4.3],
  greeting: [4.3, 7.6],
  montage: [7.6, 12.0],
  gtw: [12.0, 17.2],
  climax: [17.2, 20.2],
  strange: [20.2, 24.7],
  title: [24.7, 30.0],
} as const;

// Where each voice line starts.
export const VO = {
  n1: 0.45, // "In nineteen eighty-three... a kid dialed into the wrong computer."
  w1: 5.55, // "Shall we play a game?"
  n2: 7.75, // "Falken's Maze. Chess. Poker. Fighter Combat."
  n3: 12.55, // "And one game... that cannot be won."
  n4: 17.35, // "Is it a game... or is it real?"
  w2: 20.4, // "A strange game. The only winning move is not to play."
  n5: 24.8, // "Whopper. Sixteen games. Free, and open source." (the film says WOPR as "whopper")
};

// The montage's cuts, each a beat before the narrator names the game.
export const MONTAGE = [
  { img: "maze-1.png", name: "FALKEN'S MAZE", at: 7.6 },
  { img: "chess-1.png", name: "CHESS", at: 8.78 },
  { img: "poker-1.png", name: "POKER", at: 9.57 },
  { img: "dogfight-1.png", name: "FIGHTER COMBAT", at: 10.44 },
  { img: "blackjack-1.png", name: "BLACK JACK", at: 11.42 },
  { img: "hearts-1.png", name: "HEARTS", at: 11.62 },
  { img: "tactical-1.png", name: "THEATERWIDE TACTICAL WARFARE", at: 11.82 },
];
