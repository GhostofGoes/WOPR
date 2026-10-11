wopr's key signs both packages. The `.rpm` carries its signature inside it. The `.deb`'s signature is a
separate file on the [release page](https://github.com/GhostofGoes/WOPR/releases/tag/v@VERSION@), named
after the package with `.asc` on the end, such as `wopr_@VERSION@-1_amd64.deb.asc`. It is separate because
apt does not check a signature inside a package you download. To check either package before you install
it, see [Verifying binaries](/install#verifying-binaries-attestation). The **Linux (RPM)** line in
[Command line install methods](/install#command-line-install-methods) makes `dnf` check the signature as
it installs the package.
