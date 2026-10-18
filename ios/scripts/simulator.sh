#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Christopher Goes
# SPDX-License-Identifier: MIT
#
# Prints the UDID of an available iPhone simulator on the newest iOS runtime that Xcode has, for
# xcodebuild's -destination id=<UDID>. With an argument (26, say), it uses the newest runtime of
# that major version. It fails when there is none. macOS only (xcrun and jq).
#
# Usage: ios/scripts/simulator.sh [major-version]
set -euo pipefail

major="${1:-}"
xcrun simctl list devices available --json | jq -er --arg major "${major}" '
  .devices
  | to_entries
  | map(select(.key | test("SimRuntime\\.iOS-[0-9]+-[0-9]+$")))
  | map(.version = (.key | capture("iOS-(?<maj>[0-9]+)-(?<min>[0-9]+)$") | [(.maj | tonumber), (.min | tonumber)]))
  | map(select($major == "" or .version[0] == ($major | tonumber)))
  | map(select(any(.value[]; .name | startswith("iPhone"))))
  | sort_by(.version)
  | last
  | .value
  | map(select(.name | startswith("iPhone")))
  | first
  | .udid'
