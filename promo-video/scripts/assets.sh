#!/bin/sh
# SPDX-FileCopyrightText: 2026 Christopher Goes
# SPDX-License-Identifier: MIT
#
# Copies the screenshots and the icon that the promo shows from the repository into public/img.
# Run it from promo-video/.
set -eu
mkdir -p public/img
cp ../docs/screenshots/global-thermonuclear-war.png public/img/
for f in maze-1 chess-1 poker-1 blackjack-1 dogfight-1 hearts-1 tactical-1; do
	cp "../site/static/img/games/${f}.png" public/img/
done
cp ../packaging/icons/src/wopr.svg public/img/
