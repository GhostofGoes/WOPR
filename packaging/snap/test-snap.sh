#!/usr/bin/env bash
# Installs a wopr snap as App Center would (but from a file, so unsigned: --dangerous), checks the
# menu entry and icon that snapd exports, runs wopr in a terminal under its confinement, and removes
# the snap. Given an e2e.test (internal/e2e, built for this machine), it also runs the end-to-end
# tests against the confined program. Runs on Ubuntu, as a user who may use sudo.
#
#   packaging/snap/test-snap.sh <snap> <version> [e2e.test]
set -euo pipefail

if [[ $# -lt 2 || $# -gt 3 ]]; then
  echo "usage: $0 <snap> <version> [e2e.test]" >&2
  exit 2
fi
snap_file=$1
version=$2
e2e=${3:-}
app_id=io.github.ghostofgoes.wopr
desktop=/var/lib/snapd/desktop/applications/wopr_wopr.desktop
icon=/snap/wopr/current/meta/gui/icon.png

fail() {
  echo "test-snap: $*" >&2
  exit 1
}

# snapd, and desktop-file-validate for the exported menu entry. snap-confine refuses to start a
# snap while / is not owned by root, as it has not always been on runner images.
if ! command -v snap > /dev/null || ! command -v desktop-file-validate > /dev/null; then
  sudo apt-get update -q
  sudo apt-get install -y -q --no-install-recommends snapd desktop-file-utils
fi
root_owner="$(stat -c %u:%g /)"
if [[ "${root_owner}" != 0:0 ]]; then
  sudo chown root:root /
fi
sudo snap wait system seed.loaded

sudo snap install --dangerous "${snap_file}"
snap list wopr

out="$(/snap/bin/wopr --version)"
[[ "${out}" == "wopr v${version} ("* ]] || fail "wopr --version printed ${out}, want v${version}"

# snapd's copy of meta/gui/wopr.desktop: Exec and Icon rewritten for the snap, the terminal kept,
# and the app ID that ties the snap to the other packages' AppStream metadata.
cat "${desktop}"
for line in 'Exec=/snap/bin/wopr' 'Exec=/snap/bin/wopr --play chess' 'Exec=/snap/bin/wopr --play gtw' \
  'Exec=/snap/bin/wopr --movie' 'Terminal=true' "Icon=${icon}" \
  "X-SnapCommonID=${app_id}"; do
  grep -Fxq -- "${line}" "${desktop}" || fail "${desktop} has no line ${line}"
done
report="$(desktop-file-validate "${desktop}")"
[[ -z "${report}" ]] || fail "desktop-file-validate: ${report}"
kind="$(file -b "${icon}")"
[[ "${kind}" == "PNG image data, 256 x 256,"* ]] || fail "${icon} is ${kind}, want a 256x256 PNG"

# In an 80x24 terminal, with the debug log on: wopr must draw its LOGON: prompt, and log under the
# snap's home, ~/snap/wopr/<revision>, where a strictly confined snap may write. timeout interrupts
# it after 5 seconds, so 124 means it was still running.
typescript="$(mktemp)"
trap 'rm -f "${typescript}"' EXIT
status=0
env -u XDG_CACHE_HOME WOPR_DEBUG=1 TERM=xterm-256color \
  script -qec 'stty cols 80 rows 24 && timeout --signal=INT 5 /snap/bin/wopr --instant' "${typescript}" \
  < /dev/null > /dev/null || status=$?
[[ ${status} -eq 124 ]] || fail "wopr in a terminal exited with ${status}, want 124 (interrupted by timeout)"
grep -q 'LOGON:' "${typescript}" || fail "wopr drew no LOGON: prompt"
log="${HOME}/snap/wopr/current/.cache/wopr/debug.log"
grep -F "wopr v${version} " "${log}" || fail "no debug log in ${log}"

# TestDebugLog is skipped: it points HOME and XDG_CACHE_HOME at a directory in /tmp, and a snap has
# its own /tmp. The run above checks the log where the snap writes it.
if [[ -n "${e2e}" ]]; then
  chmod +x "${e2e}"
  "${e2e}" -test.v -test.timeout=5m -test.skip '^TestDebugLog$' -binary /snap/bin/wopr
fi

sudo snap remove --purge wopr
for f in /snap/bin/wopr "${desktop}"; do
  [[ ! -e "${f}" ]] || fail "removing the snap left ${f} behind"
done
