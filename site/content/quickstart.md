---
title: Quickstart
weight: 1
description: Install wopr, log on, and play your first game in two minutes.
---

## 1. Get it

Pick your system, copy the line, and paste it into a terminal as the tab says. Every line except Go's
also starts `wopr` once it is installed, so WOPR starts dialing right away.

{{< install-tabs >}}

[Installation](/install) says how to remove it again, and how to check that a download is genuine.

## 2. Dial in

If step 1 started it, WOPR is already dialing. If not (the Go tab), or to play again later, open a terminal
at least 80 columns wide and 24 rows tall, and type:

```sh
wopr
```

WOPR dials in, connects, and asks `LOGON:`.

{{< screenshot src="img/logon-1.png" caption="The first screen: `CONNECTING...`, `CONNECTED.` and the `LOGON:` prompt." >}}

## 3. Log on

If you have seen the film, you know what to type. If not: the password is the name of Professor
Falken's son, `Joshua`. A wrong name drops the line and WOPR dials again; every third try it gives you a
hint. You can also type `HELP GAMES` or `LIST GAMES` at `LOGON:`, as David does in the film.

Once you are in, WOPR greets you and talks the way it does in the film. Answer it, and it asks:
`SHALL WE PLAY A GAME?`

{{< screenshot src="screenshots/logon.png" caption="Logged on as Joshua: `GREETINGS PROFESSOR FALKEN.` and the film's conversation." >}}

## 4. Play a game

Type `LIST GAMES` to see what WOPR knows, then type a game's number or its name. You can also just ask:
`Let's play chess.` Ask for Global Thermonuclear War and WOPR suggests chess first, as in the film; ask
again and it says `FINE.`

Each game prints its own instructions, and the [Games](/games) pages explain them all. To leave a game,
press Esc twice.

To skip the dial and the logon, name the game when you start:

```sh
wopr chess
```

## 5. Log off

Type `LOGOFF` to end the session. Ctrl+C quits at any time.

## Next

- [Usage](/usage): every option, the themes, and what WOPR understands.
- [Movie scenes](/movie): watch the film's terminal scenes replay themselves.
- [Accessibility](/usage/accessibility): reduced motion, instant text and running without colour.
- [Troubleshooting](/usage/troubleshooting): what to do when something goes wrong, and where to ask.
