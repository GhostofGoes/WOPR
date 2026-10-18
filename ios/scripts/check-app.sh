#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Christopher Goes
# SPDX-License-Identifier: MIT
#
# Checks the iOS demo app's security properties (ios/README.md, "Security") in its sources and in a
# built WOPR.app, as the release configuration builds it for devices:
#
#   - The sources use no network, storage, web view or dynamic-loading API.
#   - The app has no entitlements, so it asks for no capability.
#   - Its Info.plist has no App Transport Security exception, no permission request, no URL scheme
#     and no background mode, and it needs iOS 26 or later.
#   - Its privacy manifest declares no tracking, no collected data and no API that needs a reason.
#   - Its program is position independent and links only the system's libraries. It embeds no
#     framework.
#
# WOPRTests/SecurityTests.swift checks the Info.plist and the privacy manifest of the debug build.
# macOS only (plutil, otool).
#
# Usage: ios/scripts/check-app.sh path/to/WOPR.app
set -euo pipefail

app="${1:?usage: check-app.sh path/to/WOPR.app}"
ios="$(cd "$(dirname "$0")/.." && pwd)"
failed=0

fail() {
  echo "::error::$*"
  failed=1
}

# --- the sources ---

# WOPR has no network code, and the demo keeps nothing between launches (SECURITY.md).
if grep -rnF \
  -e URLSession -e URLRequest -e NWConnection -e NWListener -e NWBrowser -e 'import Network' \
  -e 'import WebKit' -e WKWebView -e SFSafariViewController -e UserDefaults -e '@AppStorage' \
  -e FileManager -e NSUbiquitousKeyValueStore -e 'import SwiftData' -e 'import CoreData' \
  -e dlopen -e dlsym -e NSClassFromString -e 'openURL' -e 'UIApplication.shared.open' \
  "${ios}/WOPR"; then
  fail "the app's sources use a network, storage, web or dynamic-loading API"
fi
entitlements="$(find "${ios}" -name '*.entitlements' -print -quit)"
if [[ -n "${entitlements}" ]]; then
  fail "the app has an entitlements file"
fi
if grep -q CODE_SIGN_ENTITLEMENTS "${ios}/WOPR.xcodeproj/project.pbxproj"; then
  fail "the project sets CODE_SIGN_ENTITLEMENTS"
fi

# --- the bundle ---

info="${app}/Info.plist"
manifest="${app}/PrivacyInfo.xcprivacy"
program="${app}/WOPR"
for f in "${info}" "${manifest}" "${program}"; do
  [[ -f "${f}" ]] || fail "${f} is missing"
done
if [[ "${failed}" -ne 0 ]]; then
  exit 1
fi

for key in NSAppTransportSecurity CFBundleURLTypes LSApplicationQueriesSchemes UIBackgroundModes \
  UIFileSharingEnabled LSSupportsOpeningDocumentsInPlace UIRequiredDeviceCapabilities; do
  if plutil -extract "${key}" raw -o - "${info}" >/dev/null 2>&1; then
    fail "Info.plist has ${key}"
  fi
done
if plutil -convert xml1 -o - "${info}" | grep -q 'UsageDescription</key>'; then
  fail "Info.plist asks for a permission (a UsageDescription key)"
fi
minimum="$(plutil -extract MinimumOSVersion raw -o - "${info}")"
[[ "${minimum}" == "26.0" ]] || fail "MinimumOSVersion is ${minimum}, want 26.0"

tracking="$(plutil -extract NSPrivacyTracking raw -o - "${manifest}" 2>/dev/null || echo missing)"
[[ "${tracking}" == "false" ]] || fail "the privacy manifest's NSPrivacyTracking is ${tracking}, want false"
for key in NSPrivacyTrackingDomains NSPrivacyCollectedDataTypes NSPrivacyAccessedAPITypes; do
  # For an array, raw prints the number of its elements.
  n="$(plutil -extract "${key}" raw -o - "${manifest}" 2>/dev/null || echo missing)"
  [[ "${n}" == "0" ]] || fail "the privacy manifest's ${key} is ${n}, want an empty array"
done

# --- the program ---

otool -hv "${program}" | grep -qw PIE || fail "the program is not position independent (PIE)"
if [[ -d "${app}/Frameworks" ]]; then
  fail "the app embeds frameworks (${app}/Frameworks)"
fi
# The first line is the program's own path; each other line is a library it links.
libs="$(otool -L "${program}")"
while read -r lib _; do
  case "${lib}" in
    /System/Library/Frameworks/Network.framework/* | /System/Library/Frameworks/WebKit.framework/*)
      fail "the program links ${lib}" ;;
    /System/Library/* | /usr/lib/*) ;;
    *) fail "the program links ${lib}, which is not the system's" ;;
  esac
done <<<"${libs#*$'\n'}"

if [[ "${failed}" -ne 0 ]]; then
  exit 1
fi
echo "${app}: the security checks pass"
