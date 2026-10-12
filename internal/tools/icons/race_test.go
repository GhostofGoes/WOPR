// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build race

package main

// raceDetector reports a build with -race, under which compressing every icon as the tool
// does takes most of a minute.
const raceDetector = true
