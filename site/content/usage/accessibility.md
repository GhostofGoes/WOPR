---
title: Accessibility
weight: 1
description: Meaning in characters as well as colour, reduced motion, instant text, plain ASCII and tested contrast.
aliases: [/accessibility/]
prev: /usage
---

You can play `wopr` without colour, without motion, and without waiting for text to appear one character
at a time. Its tests check every game's screens for the points below.

## Meaning in characters, not just colour

Everything that colour shows is also in the characters. So nothing is lost in a terminal without colour,
or with `NO_COLOR` set:

- On Global Thermonuclear War's big board, outgoing missiles are `+`, incoming missiles are `*`, and
  impacts are `X`. A `>` points at the current DEFCON level.
- Every card shows its suit letter (`7H` is the seven of hearts), and the card that is winning a trick is
  edged `.===.`.
- Gin Rummy names your melds by their positions in your hand.
- Chess and checkers mark the last move with brackets.

Set `NO_COLOR` (to any value but an empty one) to turn colour off. Every theme then uses bold, dim,
underline and reverse instead.

{{< screenshot src="img/themes-nocolor.png" caption="Black Jack with `NO_COLOR=1`: the cards read `AS` and `JS`." >}}

## Reduced motion

`--reduce-motion` (`-r`, or `WOPR_REDUCE_MOTION=1`) does three things:

- It stops the cursor and DEFCON 1 from blinking.
- It holds the front panel's lights and the `PROCESSING` dots still.
- It keeps the ending at a steady pace, instead of letting it get faster.

The typing and the animations still play. Add `--instant` to skip them. With or without either option,
nothing ever flashes more than three times a second.

## Instant text

`--instant` (`-i`, or `WOPR_INSTANT=1`) shows each line at once, instead of one character at a time. It
also draws each strike on the big board at once, and it skips the ending's self-play and scrolling
scenarios. Without it, any key shows the rest of the line that WOPR is typing, and what you type is kept.

## Plain text

- Everything on screen is plain ASCII.
- Every prompt with words ends with a question mark or a colon, and the cursor sits at the end of the
  line you are typing.
- WOPR's conversation, as in the film, has no prompt text: what WOPR said last is the question. After a
  list, such as `HELP` or `LIST GAMES`, the list's last line comes just before the empty prompt.
- WOPR writes its moves in words, as well as drawing them on the boards (`WOPR: D7D5`).
- In Hearts and Bridge, the console says which cards are already in the trick before you play
  (`WEST LEADS 7H. NORTH PLAYS KH.`).

## Contrast

In every theme's full colours, text meets the WCAG AA contrast ratio of 4.5:1 against the background.
Text that is dim on purpose meets 3:1. In 16-colour terminals, every style keeps at least 3:1 on xterm's
standard palette.

## Screen readers

Nobody has tried the full-screen interface with a screen reader yet.
[Reports are welcome](/usage/troubleshooting#getting-help). The printing options (`--help`,
`--games`, `--scenes`, `--licenses`) write plain text, and they work anywhere.
