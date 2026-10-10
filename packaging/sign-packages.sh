#!/usr/bin/env bash
# Signs the Linux packages of a release with wopr's OpenPGP key (docs/PLAN.md §8, "Signatures").
#
#   WOPR_SIGNING_KEY=<armored secret key> [WOPR_SIGNING_PASSPHRASE=...] \
#     packaging/sign-packages.sh <release dir> <public key file> [<fingerprint file>]
#
# The release passes packaging/wopr-signing-key.asc and packaging/wopr-signing-key.fingerprint;
# CI passes a throwaway key, made for the run, and no fingerprint file.
#
# In the release directory (stage -assets's files and checksums.txt) it:
#   1. checks every file against checksums.txt;
#   2. signs each .rpm in place (rpmsign: the signature header dnf, zypper and rpm -K check),
#      and proves the signing changed nothing but that header: the header's and the payload's
#      SHA-256 digests are the same before and after, and rpmkeys accepts the signature with
#      only the public key;
#   3. writes a detached, armored signature beside each .deb (wopr_X.deb.asc; apt does not check
#      signatures on a downloaded .deb, so this is for gpg --verify), checked with gpgv;
#   4. rewrites checksums.txt for the signed files and the .asc files, refusing any change but
#      those, and signs it as checksums.txt.asc.
# The secret key must belong to the public key file's primary key (the committed key), so a
# replaced secret cannot sign. It is imported into a temporary GnuPG home, removed on exit.
set -euo pipefail

if [[ $# -ne 2 && $# -ne 3 ]]; then
  echo "usage: WOPR_SIGNING_KEY=... $0 <release dir> <public key file> [<fingerprint file>]" >&2
  exit 2
fi
dir=$1
pubkey=$(realpath "$2")
want_fpr=""
if [[ $# -eq 3 ]]; then
  want_fpr=$(tr -d ' \n' <"$3")
fi
: "${WOPR_SIGNING_KEY:?WOPR_SIGNING_KEY must hold the armored secret key}"

work=$(mktemp -d)
cleanup() {
  gpgconf --homedir "${work}/gnupg" --kill all 2>/dev/null || true
  rm -rf -- "${work}"
}
trap cleanup EXIT
export GNUPGHOME="${work}/gnupg"
mkdir -m 700 "${GNUPGHOME}"

fpr=$(gpg --batch --quiet --show-keys --with-colons "${pubkey}" | awk -F: '$1 == "fpr" { print $10; exit }')
if [[ ! "${fpr}" =~ ^[0-9A-F]{40}([0-9A-F]{24})?$ ]]; then
  echo "error: no key fingerprint in ${pubkey}" >&2
  exit 1
fi
if [[ -n "${want_fpr}" && "${fpr}" != "${want_fpr}" ]]; then
  echo "error: ${pubkey} is ${fpr}, not ${want_fpr} as $3 says" >&2
  exit 1
fi
gpg --batch --quiet --import <<<"${WOPR_SIGNING_KEY}"
if ! gpg --batch --list-secret-keys --with-colons "${fpr}" | grep -q '^sec'; then
  echo "error: WOPR_SIGNING_KEY is not the secret key of ${fpr} (${pubkey})" >&2
  exit 1
fi

gpg_args=(--batch --pinentry-mode loopback)
if [[ -n "${WOPR_SIGNING_PASSPHRASE:-}" ]]; then
  printf '%s' "${WOPR_SIGNING_PASSPHRASE}" >"${work}/passphrase"
  gpg_args+=(--passphrase-file "${work}/passphrase")
else
  gpg_args+=(--passphrase '')
fi
gpg --batch --dearmor -o "${work}/pubring.gpg" "${pubkey}"

cd "${dir}"
sha256sum --strict --quiet -c checksums.txt

shopt -s nullglob
rpms=(*.rpm)
debs=(*.deb)
if [[ ${#rpms[@]} -eq 0 || ${#debs[@]} -eq 0 ]]; then
  echo "error: want .rpm and .deb files in ${dir}" >&2
  exit 1
fi

mkdir "${work}/rpmdb"
rpmkeys --dbpath "${work}/rpmdb" --import "${pubkey}"
for f in "${rpms[@]}"; do
  digests=(--qf '%{SHA256HEADER} %{PAYLOADDIGEST}\n')
  before=$(rpm -qp "${digests[@]}" "${f}" 2>/dev/null)
  rpmsign --addsign --define "_gpg_name ${fpr}" --define "_gpg_path ${GNUPGHOME}" \
    --define "_gpg_sign_cmd_extra_args ${gpg_args[*]}" "${f}" >/dev/null
  after=$(rpm -qp "${digests[@]}" "${f}" 2>/dev/null)
  if [[ ! "${before}" =~ ^[0-9a-f]{64}\ [0-9a-f]{64}$ || "${before}" != "${after}" ]]; then
    echo "error: signing ${f} changed its header or payload digests: ${before} -> ${after}" >&2
    exit 1
  fi
  check=$(rpmkeys --dbpath "${work}/rpmdb" --checksig "${f}")
  if [[ "${check}" != "${f}: digests signatures OK" ]]; then
    echo "error: ${check}" >&2
    exit 1
  fi
  echo "signed ${f}"
done

for f in "${debs[@]}"; do
  gpg "${gpg_args[@]}" --yes --local-user "${fpr}" --armor --detach-sign -o "${f}.asc" "${f}"
  gpgv --keyring "${work}/pubring.gpg" "${f}.asc" "${f}" 2>/dev/null
  echo "signed ${f} (${f}.asc)"
done

# The new checksums.txt, in GoReleaser's order (sorted by name). Only the .rpm lines may change,
# and only the .asc lines may be new.
files=()
for f in *; do
  if [[ -f "${f}" && "${f}" != checksums.txt && "${f}" != checksums.txt.asc ]]; then
    files+=("${f}")
  fi
done
mapfile -t files < <(printf '%s\n' "${files[@]}" | LC_ALL=C sort || true)
sha256sum -- "${files[@]}" >"${work}/checksums.txt"
changed=$(diff <(LC_ALL=C sort checksums.txt) <(LC_ALL=C sort "${work}/checksums.txt") | grep '^[<>]' || true)
unexpected=$(grep -Ev '^[<>] [0-9a-f]{64}  [^ ]+\.rpm$|^> [0-9a-f]{64}  [^ ]+\.deb\.asc$' <<<"${changed}" || true)
if [[ -n "${unexpected}" ]]; then
  echo "error: unexpected checksum changes:" >&2
  echo "${unexpected}" >&2
  exit 1
fi
cp "${work}/checksums.txt" checksums.txt
gpg "${gpg_args[@]}" --yes --local-user "${fpr}" --armor --detach-sign -o checksums.txt.asc checksums.txt
gpgv --keyring "${work}/pubring.gpg" checksums.txt.asc checksums.txt 2>/dev/null
echo "signed checksums.txt (checksums.txt.asc), with ${fpr}"
