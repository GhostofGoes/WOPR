#!/usr/bin/env bash
# Opens WOPR.app from wopr_<version>_macos.dmg as Finder does, and checks the hand-off: the app's
# program, started by launchd with no terminal, opens Terminal running itself and exits, and
# Terminal's run has a terminal. Then it stops both. The image comes from a CI artifact, which
# macOS does not quarantine, so Gatekeeper's first-launch prompt is not part of this; that needs a
# person on a Mac. Runs on macOS, in a logged-in session (GitHub's macOS runners have one).
#
#   packaging/macos/test-launch.sh <dmg>
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <dmg>" >&2
  exit 2
fi
dmg=$1

fail() {
  echo "test-launch: $*" >&2
  local pid
  for pid in $(pgrep -f 'WOPR\.app|Terminal' || true); do
    ps -o pid=,ppid=,tty=,command= -p "${pid}" >&2 || true
  done
  exit 1
}

# Copied out of the image first, as a user drags it to Applications.
dest="$(mktemp -d)"
mnt="$(mktemp -d)"
hdiutil attach "${dmg}" -readonly -nobrowse -noautoopen -mountpoint "${mnt}"
ditto "${mnt}/WOPR.app" "${dest}/WOPR.app"
hdiutil detach "${mnt}" -quiet || hdiutil detach "${mnt}" -force -quiet
# launchd may start the program by its /var path, and Terminal gets the /private/var one, so the
# processes are matched by the end of the path, which is unique to this copy.
exe="${dest##*/}/WOPR.app/Contents/MacOS/wopr"

cleanup() {
  pkill -f "${exe}" || true
  pkill -x Terminal || true
}
trap cleanup EXIT

# The processes running the program, as "pid tty" lines; a process with no terminal shows "??".
runs() {
  ps -axo pid=,tty=,command= |
    awk -v exe="/${exe}" 'length($3) >= length(exe) && substr($3, length($3) - length(exe) + 1) == exe { print $1, $2 }'
}

open "${dest}/WOPR.app"
found=
for _ in $(seq 60); do
  list="$(runs)"
  if awk 'NF >= 2 && $2 != "??" { f = 1 } END { exit !f }' <<< "${list}"; then
    found=1
    break
  fi
  sleep 1
done
[[ -n "${found}" ]] || fail "Terminal did not run ${dest}/WOPR.app/Contents/MacOS/wopr within 60 seconds"
echo "Terminal runs the program with a terminal:"
echo "${list}"

# The first run, started by launchd with no terminal, has exited.
gone=
for _ in $(seq 10); do
  list="$(runs)"
  if ! awk '$2 == "??" { f = 1 } END { exit !f }' <<< "${list}"; then
    gone=1
    break
  fi
  sleep 1
done
[[ -n "${gone}" ]] || fail "the run started by launchd is still running"
echo "ok: the app opened in Terminal, and the first run exited"
