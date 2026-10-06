# Notices

## Code

The source code of `wopr` is licensed under the MIT License; see [LICENSE](LICENSE). The MIT grant
covers the code only. It does **not** cover the material listed under "Film text" and "Third-party
text" below.

## Film text

`wopr` is a fan-made homage to the 1983 film *WarGames*. It quotes short passages of the text that
the film shows on the WOPR computer's terminal: the game list, log-on and greeting lines, NORAD notices
and similar screen text. Those quotations remain the property of their copyright holders. They are
not licensed under the MIT License, and they are listed with a provenance tag in
`internal/wopr/lines.go`, `internal/assets/` and `internal/movie/scenes/`.

This project is not affiliated with, endorsed by, or sponsored by Metro-Goldwyn-Mayer, United
Artists, or anyone involved in making the film. *WarGames* is a trademark of its owner and is used
here only to describe what this project is inspired by. "WOPR" is used as the name of the film's
fictional computer.

## Third-party text

Two items of screen text were transcribed by the `abs0/wargames` project
(<https://github.com/abs0/wargames>, `wargames.sh`, commit `010ed92`) and are used under its BSD
2-clause licence, reproduced below:

- the "backdoor" connection header block shown after log-on;
- the list of scenario names shown in the closing montage.

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

## Third-party software

The `wopr` binary is built with Go and links third-party Go modules. Their licences are reproduced in
[THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt), which ships in every release archive and is
printed by `wopr --licenses`.
