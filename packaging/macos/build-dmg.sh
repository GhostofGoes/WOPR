#!/usr/bin/env bash
# Builds wopr_<version>_macos.dmg from the release files: the two darwin programs, joined into one
# universal program with lipo, in WOPR.app (internal/tools/macapp), ad-hoc signed, beside a link to
# /Applications for the user to drag it onto. It never compiles wopr: the programs and the notices
# must match checksums.txt, which the release attests. Runs on macOS, from the repository's root.
#
#   packaging/macos/build-dmg.sh <dist-dir> <version> <out-dir>
#
# The signature is ad-hoc (codesign --sign -), with no Apple Developer ID: Apple silicon Macs run
# no program without a signature, and Gatekeeper still asks the user to approve the app once. It
# uses the hardened runtime, which notarization will require, so the app runs now as it will then.
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <dist-dir> <version> <out-dir>" >&2
  exit 2
fi
dist=$1
version=$2
out=$3
if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "build-dmg: ${version} is not a version like 1.2.3 or 1.2.4-snapshot.abc1234" >&2
  exit 2
fi
if [[ ! -f internal/tools/macapp/main.go ]]; then
  echo "build-dmg: run this from the repository's root" >&2
  exit 2
fi
bundle_id=io.github.ghostofgoes.wopr
dmg="${out}/wopr_${version}_macos.dmg"
work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

# The inputs, each listed once in checksums.txt and matching it.
for name in "wopr_${version}_darwin_amd64" "wopr_${version}_darwin_arm64" LICENSE NOTICE.md THIRD_PARTY_NOTICES.txt; do
  entry="$(awk -v n="${name}" '$2 == n' "${dist}/checksums.txt")"
  if [[ -z "${entry}" || "${entry}" == *$'\n'* ]]; then
    echo "build-dmg: checksums.txt must list ${name} once" >&2
    exit 1
  fi
  printf '%s\n' "${entry}" >> "${work}/inputs.sha256"
done
(cd "${dist}" && shasum -a 256 --check "${work}/inputs.sha256")

lipo -create -output "${work}/wopr" "${dist}/wopr_${version}_darwin_amd64" "${dist}/wopr_${version}_darwin_arm64"
archs="$(lipo -archs "${work}/wopr" | tr ' ' '\n' | sort | xargs)"
if [[ "${archs}" != "arm64 x86_64" ]]; then
  echo "build-dmg: the universal program holds ${archs}, want arm64 and x86_64" >&2
  exit 1
fi

mkdir -p "${work}/dmg"
app="${work}/dmg/WOPR.app"
go run ./internal/tools/macapp -binary "${work}/wopr" -version "${version}" \
  -icon packaging/icons/wopr.icns -notices "${dist}" -out "${app}"
codesign --force --sign - --identifier "${bundle_id}" --options runtime "${app}"
codesign --verify --strict --verbose=2 "${app}"
codesign --display --verbose=2 "${app}"
ln -s /Applications "${work}/dmg/Applications"

# HFS+ rather than whatever the runner's disk uses, so every macOS the app supports reads it.
# hdiutil on CI runners now and then fails with "Resource busy"; only that is retried.
mkdir -p "${out}"
for attempt in 1 2 3 4 5; do
  if hdiutil create -volname WOPR -srcfolder "${work}/dmg" -fs HFS+ -format UDZO "${dmg}" 2> "${work}/hdiutil.log"; then
    break
  fi
  cat "${work}/hdiutil.log" >&2
  if [[ ${attempt} -eq 5 ]] || ! grep -q 'Resource busy' "${work}/hdiutil.log"; then
    exit 1
  fi
  rm -f "${dmg}"
  echo "build-dmg: hdiutil create failed (attempt ${attempt}); retrying" >&2
  sleep $((attempt * 10))
done
hdiutil verify "${dmg}"
shasum -a 256 "${dmg}"
