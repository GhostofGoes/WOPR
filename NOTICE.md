# Notices

## Code

The source code of `wopr` is licensed under the MIT License; see [LICENSE](LICENSE). The MIT grant
covers the code only. It does **not** cover the material listed under "Film text", "Third-party
text" and "Third-party art" below.

## Film text

`wopr` is a fan-made homage to the 1983 film *WarGames*. It quotes short passages of the text that
the film shows on the WOPR computer's terminal: the game list, log-on and greeting lines, NORAD notices
and similar screen text. Those quotations remain the property of their copyright holders. They are
not licensed under the MIT License, and they are listed with a provenance tag in
`internal/wopr/lines.go`, `internal/assets/`, `internal/movie/scenes/` and the games' packages under
`internal/games/` (Global Thermonuclear War, tic-tac-toe and the ending). Movie mode (`wopr --movie`)
replays the film's terminal scenes with that text, including the lines typed at the terminal; it contains no
dialogue that is only spoken, no audio and no stills.

This project is not affiliated with, endorsed by, or sponsored by Metro-Goldwyn-Mayer, United
Artists, or anyone involved in making the film. *WarGames* is a trademark of its owner and is used
here only to describe what this project is inspired by. "WOPR" is used as the name of the film's
fictional computer.

## Third-party text

Two items of the film's screen text are taken from the transcription made by the `abs0/wargames`
project (<https://github.com/abs0/wargames>, `wargames.sh`, commit `010ed92`):

- the "backdoor" connection header block shown after log-on;
- the list of scenario names shown in the closing montage.

abs0's BSD 2-clause licence, reproduced below, covers the transcription. The text itself is the film's,
so like the other film text it is not covered by the MIT grant.

```text
(Standard 2 clause BSD licence)

Copyright (c) 2012 David Brownlee <abs@absd.org>
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
   notice, this list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright
   notice, this list of conditions and the following disclaimer in the
   documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED
TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS
BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE.
```

## Third-party art

The big board's world map in Global Thermonuclear War is Matthew Thomas's ASCII world map, from
<https://asciiart.website/art/3719>, reproduced exactly as he drew it; the board shows its northern
rows. It is used under the terms its author gave with it:

```text
Map (C) 1998 Matthew Thomas. Freely usable if this line is included.
```

## Map data

The side-choice outlines of the United States and the Soviet Union in Global Thermonuclear War are
original ASCII art, generated for this project from [Natural Earth](https://www.naturalearthdata.com/)
country polygons (1:50m scale). Natural Earth is in the public domain; this credit is a courtesy.

## Earlier drafts

An early draft of this project's plan (commit `ada63a0` in the git history) contained fragments of the
ASCII world map from Franklin Wei's `wargames` project (<https://github.com/built1n/wargames>, file
`MAP`), and [docs/reviews/PLAN-v1-review.md](docs/reviews/PLAN-v1-review.md) quotes three of them. Those
fragments are by Franklin Wei and are available under the
[Creative Commons Attribution-ShareAlike 4.0](https://creativecommons.org/licenses/by-sa/4.0/) licence.
The `wopr` program does not contain them.

## Rights holders

If you believe something here infringes your rights, contact the maintainer, @GhostofGoes, using the
contact details on their GitHub profile. The material will be removed or replaced promptly.

## Third-party software

The `wopr` binary is built with Go and links third-party Go modules. Their licences are reproduced in
[THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt), which ships in every release archive and is
printed by `wopr --licenses`.
