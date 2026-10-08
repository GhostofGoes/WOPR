---
title: Movie scenes
weight: 6
description: Movie mode replays the film's six scenes at WOPR's terminal, typed and paced as on screen.
prev: /usage/troubleshooting
---

Movie mode replays the film's scenes at WOPR's terminal as a show that runs itself. David's lines type
themselves, WOPR answers, and the war and the ending play through the same console and games you play
with. Nothing needs typing.

The scenes show only what WOPR's terminal shows on screen, David's typing included; there is no dialogue
that is only spoken. The dial, the strike exchange, tic-tac-toe's prompts and the game clock's seconds
are this project's own text.

## The scenes

`wopr --scenes` lists them:

```text
 1. first-contact    LOGON ATTEMPTS, HELP AND THE LIST OF GAMES
 2. joshua           THE BACKDOOR, THE GREETING AND A GAME OF CHOICE
 3. first-strike     A SIDE, TWO TARGETS AND THE BIG BOARD
 4. call-back        WOPR CALLS BACK TO FINISH THE GAME
 5. norad-terminal   JOSHUA AT NORAD: KILL RATIOS AND FALKEN'S ADDRESS
 6. climax           DEFCON 1, TIC-TAC-TOE AND A STRANGE GAME
```

1. **First contact** (`first-contact`). David's first connection. A number at `LOGON:` is refused and
   the line drops. After the dial, he asks for help on logging on and on games, lists the games, and
   tries a game's name, which is refused too.
2. **Joshua** (`joshua`). The backdoor. The connection header and the status lines, WOPR's greeting,
   the small talk, and Global Thermonuclear War asked for twice.
3. **First strike** (`first-strike`). Global Thermonuclear War: David picks a side and two targets, and
   the missiles fly on the big board.
4. **Call-back** (`call-back`). WOPR phones David at home to finish yesterday's game, and shows how
   long it has left.
5. **NORAD terminal** (`norad-terminal`). Held at NORAD, David reaches Joshua again: the war goes on,
   the projected kill ratios, and the address that leads him to Falken.
6. **Climax** (`climax`). DEFCON 1 while WOPR searches for the launch code. A game it will not switch
   to, then tic-tac-toe: one player and a stalemate, then zero players, WOPR playing itself, and the
   film's last words.

{{< screenshot src="img/movie-joshua.png" caption="The `joshua` scene typing itself: the greeting, the small talk, and Global Thermonuclear War asked for twice." >}}

{{< screenshot src="img/movie-board.png" caption="The `first-strike` scene: the big board after the first strike."
  credit="The big board's map: Map (C) 1998 Matthew Thomas. Freely usable if this line is included." >}}

## Play one scene, or all of them

| Command | What it plays |
|---|---|
| `wopr --movie` or `wopr -m` | Opens the scene menu. Type a scene's number or name, or `q` to leave. |
| `wopr -m 3` | From scene 3 to the end of the list, then exits. |
| `wopr -m first-strike` | The same, by name. Any start of a name that fits only one scene works: `wopr -m first-s`. |
| `wopr -m 3 --only` | Scene 3 alone, then exits. `-o` is short for `--only`. |
| `wopr -m --only` | The menu, where each scene you pick plays alone and then the menu comes back. |
| `wopr --scenes` | Lists the scenes and exits. |

{{< screenshot src="img/movie-menu.png" caption="`wopr -m`: the scene menu and its keys." >}}

## Keys

While a scene plays:

| Key | What it does |
|---|---|
| Space | Pause, and Space again to go on. |
| `n` or Right | Skip to the next scene. |
| `p` or Left | Go back one scene. |
| Esc | Open the scene menu. |
| Ctrl+C | Quit at once. |

With `--only`, the scene that `n` or `p` reaches plays alone too. Once the climax reaches tic-tac-toe, it
plays to the end, and any other key only says so.

## Options

Every replay is the same: movie mode ignores `--seed` and `WOPR_SEED`. These options still apply:

- `--theme` picks the colours, as in play. `--theme norad` suits the NORAD scenes.
- `--instant` shows each line at once instead of typing it, but the scenes still pause so that every page
  can be read.
- `--reduce-motion` stops blinking and keeps the ending at a steady pace.

The film's lines in the scenes carry provenance tags, and they are still to be checked one by one against
the film. See [Credits and notices](/credits).
