For Debian, Ubuntu, Linux Mint and their relatives:

@DOWNLOAD-DEB@

For Fedora, RHEL, CentOS Stream, Rocky Linux, AlmaLinux and their relatives:

@DOWNLOAD-RPM@

Then open the file you downloaded: click it in your browser's downloads, or double-click it in your
**Downloads** folder. Your desktop's software app opens, such as Software on Fedora, and offers to
install it. On Ubuntu, App Center opens and warns that the package comes from a third party. Click
**Install**.

To play, open a terminal and type `wopr`.@APP-MENU@ The package also installs the manual page,
`man wopr`. To remove it, see [Uninstall](/install#uninstall).

wopr's key signs both packages. The `.rpm` carries its signature inside it. The `.deb`'s signature is a
separate file on the [release page](https://github.com/GhostofGoes/WOPR/releases/tag/v@VERSION@), named
after the package with `.asc` on the end, such as `wopr_@VERSION@-1_amd64.deb.asc`. It is separate because
apt does not check a signature inside a package you download. To check either package before you install
it, see [Verifying binaries](/install#verifying-binaries-attestation). The **Linux (RPM)** line in
[Command line install methods](/install#command-line-install-methods) makes `dnf` check the signature as
it installs the package.

If your software app will not install the package, install WOPR from the
[command line](/install#command-line-install-methods). Do the same on another Linux, such as openSUSE, or
Fedora Silverblue and the other atomic desktops.
