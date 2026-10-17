# WOPR promo video

A 30-second promo for `wopr`, in the style of the *WarGames* trailer. It is built with
[Remotion](https://www.remotion.dev/) in two formats: 16:9 (1920×1080) and 9:16 (1080×1920).

## The story

| Time | Scene | Voice |
|---|---|---|
| 0–4.3 s | The terminal warms up and dials in. The player types `Joshua` at `LOGON:`. | Narrator: "In nineteen eighty-three... a kid dialed into the wrong computer." |
| 4.3–7.6 s | WOPR greets Professor Falken. The camera moves in on `SHALL WE PLAY A GAME?`. | WOPR: "Shall we play a game?" |
| 7.6–12 s | The film's `LIST GAMES`, and each game cuts in as the narrator names it. | Narrator: "Falken's Maze. Chess. Poker. Fighter Combat." |
| 12–17.2 s | Global Thermonuclear War: the title, then the big board while DEFCON falls to 1. | Narrator: "And one game... that cannot be won." |
| 17.2–20.2 s | DEFCON 1. Tic-tac-toe boards fill, and the launch code cracks. The cut comes before the last character. | Narrator: "Is it a game... or is it real?" |
| 20.2–24.7 s | Silence, then WOPR's verdict, typed with its voice. | WOPR: "A strange game. The only winning move is not to play." |
| 24.7–30 s | The name, what it is, where to get it, and `HOW ABOUT A NICE GAME OF CHESS?` | Narrator: "WOPR. Sixteen games. Free, and open source." |

`src/timeline.ts` holds every cue. The cuts land on the narrator's words, from ElevenLabs' character
timings.

## Sources

- The screenshots and the icon are wopr's own. `scripts/assets.sh` copies them from `docs/screenshots/`, `site/static/img/games/` and `packaging/icons/src/`.
- The terminal text is the film's screen text, exactly as the `film_lines.golden` files hold it, and
  wopr's own `CONNECTING...` and `CONNECTED.`.
- The colours are wopr's `imsai` and `norad` themes. The DEFCON ladder and the front-panel lamps follow
  the program. Nothing flashes more than three times a second.
- The voices, the sound effects and the score are generated with ElevenLabs (`scripts/voice.py`,
  `scripts/sfx.py`). The narrator is the "Wayne" voice. WOPR is the "Daniel" voice at full stability,
  through a telephone band, a light bit crusher, a phaser and a short echo. The alarm is synthesized
  with ffmpeg. The film's own audio and footage are not used.
- The big board's map carries its credit line on screen, and the end card says that wopr is a fan-made
  homage.

## Build it

You need Node.js 18 or newer, ffmpeg, Python 3, and an ElevenLabs API key with text to speech, sound
effects and music.

```sh
cd promo-video
npm ci
./scripts/assets.sh
python3 -I scripts/voice.py   # voices into public/audio, timings into audio-src
python3 -I scripts/sfx.py     # sound effects and the score into public/audio
```

The alarm and the WOPR voice's processing are ffmpeg commands:

```sh
for w in w1 w2; do
  ffmpeg -y -i audio-src/raw-$w.mp3 -af "asetrate=44100*0.94,aresample=44100,highpass=f=280,lowpass=f=3600,acrusher=bits=10:mode=log:aa=1:samples=1,aphaser=in_gain=0.8:out_gain=0.9:delay=2:decay=0.3:speed=0.4,aecho=0.8:0.6:28:0.25,volume=1.6" public/audio/vo-$w.mp3
done
ffmpeg -y -f lavfi -i "aevalsrc='0.5*tanh(4*sin(2*PI*(380*mod(t,0.8)+260*mod(t,0.8)*mod(t,0.8)/0.8)))*lt(mod(t,0.8),0.62)*lt(t,1.6)':s=44100:d=2.4" -af "lowpass=f=2500,highpass=f=200,aecho=0.8:0.7:120|260:0.35|0.2,afade=t=out:st=1.9:d=0.5" public/audio/sfx-klaxon.mp3
```

ElevenLabs gives a different take each time. If a voice line's length changes, move the cues in
`src/timeline.ts` and the scenes to match.

Then render and normalize the loudness:

```sh
npx remotion render src/index.ts Promo-Landscape out/promo-landscape.mp4 --crf=16
npx remotion render src/index.ts Promo-Portrait out/promo-portrait.mp4 --crf=16
for o in landscape portrait; do
  ffmpeg -y -i out/promo-$o.mp4 -c:v copy -af "loudnorm=I=-14:TP=-1.5:LRA=11" -c:a aac -b:a 192k -ar 48000 -movflags +faststart out/wopr-promo-$o.mp4
done
```

`npx remotion studio src/index.ts` previews it in a browser. In a container without Remotion's browser,
add `--browser-executable` with a Chromium headless shell.
