#!/usr/bin/env bash
# Checks wopr_<version>_macos.dmg: it mounts, holds WOPR.app and a link to /Applications, the
# app's Info.plist names this version, its signature verifies, and its program is universal and
# runs. Given an e2e.test (internal/e2e, built for this Mac), it also runs the end-to-end tests
# against the program inside the app. Runs on macOS; each architecture runs its own half.
#
#   packaging/macos/test-dmg.sh <dmg> <version> [e2e.test]
set -euo pipefail

if [[ $# -lt 2 || $# -gt 3 ]]; then
  echo "usage: $0 <dmg> <version> [e2e.test]" >&2
  exit 2
fi
dmg=$1
version=$2
e2e=${3:-}
short="${version%%-*}" # Info.plist takes X.Y.Z only

fail() {
  echo "test-dmg: $*" >&2
  exit 1
}

mnt="$(mktemp -d)"
detach() {
  # A volume that Spotlight or XProtect is still reading is busy for a moment.
  hdiutil detach "${mnt}" -quiet || hdiutil detach "${mnt}" -force -quiet || true
  rmdir "${mnt}" || true
}
trap detach EXIT
hdiutil attach "${dmg}" -readonly -nobrowse -noautoopen -mountpoint "${mnt}"
app="${mnt}/WOPR.app"

# The window the user sees: the app and the folder to drag it onto, nothing else.
top="$(find "${mnt}" -mindepth 1 -maxdepth 1 ! -name '.*' -exec basename {} \; | sort | xargs)"
[[ "${top}" == "Applications WOPR.app" ]] || fail "the image holds ${top}, want Applications and WOPR.app"
[[ -L "${mnt}/Applications" ]] || fail "Applications is not a link"
link="$(readlink "${mnt}/Applications")"
[[ "${link}" == /Applications ]] || fail "Applications links to ${link}, not /Applications"
for f in Info.plist PkgInfo MacOS/wopr Resources/WOPR.icns Resources/LICENSE Resources/NOTICE.md \
  Resources/THIRD_PARTY_NOTICES.txt _CodeSignature/CodeResources; do
  [[ -f "${app}/Contents/${f}" ]] || fail "WOPR.app has no Contents/${f}"
done
[[ -x "${app}/Contents/MacOS/wopr" ]] || fail "Contents/MacOS/wopr is not executable"
pkginfo="$(cat "${app}/Contents/PkgInfo")"
[[ "${pkginfo}" == 'APPL????' ]] || fail "PkgInfo is ${pkginfo}, want APPL????"

plutil -lint "${app}/Contents/Info.plist"
expect() {
  local got
  got="$(plutil -extract "$1" raw -o - "${app}/Contents/Info.plist")"
  [[ "${got}" == "$2" ]] || fail "Info.plist's $1 is ${got}, want $2"
}
expect CFBundleIdentifier io.github.ghostofgoes.wopr
expect CFBundleName WOPR
expect CFBundleExecutable wopr
expect CFBundleIconFile WOPR
expect CFBundlePackageType APPL
expect CFBundleShortVersionString "${short}"
expect CFBundleVersion "${short}"
expect LSUIElement true
minos="$(plutil -extract LSMinimumSystemVersion raw -o - "${app}/Contents/Info.plist")"
echo "LSMinimumSystemVersion: ${minos}"

# Ad-hoc signed with the bundle's identifier and the hardened runtime. Gatekeeper rejects such an
# app until its user approves it; spctl's verdict is printed for the log only.
codesign --verify --strict --verbose=2 "${app}"
sig="$(codesign --display --verbose=2 "${app}" 2>&1)"
echo "${sig}"
grep -qx 'Identifier=io.github.ghostofgoes.wopr' <<< "${sig}" || fail "the signature's identifier is wrong"
grep -qx 'Signature=adhoc' <<< "${sig}" || fail "the signature is not ad-hoc"
grep -Eq '^CodeDirectory .*flags=0x[0-9a-f]+\([^)]*runtime' <<< "${sig}" || fail "the hardened runtime is off"
spctl --assess --type execute --verbose=2 "${app}" || true

archs="$(lipo -archs "${app}/Contents/MacOS/wopr" | tr ' ' '\n' | sort | xargs)"
[[ "${archs}" == "arm64 x86_64" ]] || fail "the program holds ${archs}, want arm64 and x86_64"

# With an argument, the program never reopens itself in Terminal.
out="$("${app}/Contents/MacOS/wopr" --version)"
[[ "${out}" == "wopr v${version} ("* ]] || fail "wopr --version printed ${out}, want v${version}"
echo "${out}"

# The e2e tests start the program from this shell, not from launchd, so it never reopens itself in
# Terminal either: a run without a terminal fails as it does anywhere else.
if [[ -n "${e2e}" ]]; then
  chmod +x "${e2e}"
  "${e2e}" -test.v -test.timeout=5m -binary "${app}/Contents/MacOS/wopr"
fi
