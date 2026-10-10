Both packages are signed with wopr's key. The `.rpm` carries its signature inside it. The `.deb`'s
signature is a file of its own on the [release page](https://github.com/GhostofGoes/WOPR/releases/tag/v@VERSION@),
named after the package with `.asc` on the end, such as `wopr_@VERSION@-1_amd64.deb.asc`, because apt does
not check a signature in a package you download. To check either before you install it, see
[Verifying binaries](/install#verifying-binaries-attestation). The **Linux (RPM)** line in
[Command line install methods](/install#command-line-install-methods) has `dnf` check the signature as it
installs the package.
