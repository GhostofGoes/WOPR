//go:build race

package main

// raceDetector reports a build with -race, under which compressing every icon as the tool
// does takes most of a minute.
const raceDetector = true
