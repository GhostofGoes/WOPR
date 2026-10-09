[**Download WOPR for Mac**](https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@_macos.dmg)

One app for every Mac, Apple silicon or Intel. Then:

1. Open the file you downloaded, `wopr_@VERSION@_macos.dmg`. A window shows **WOPR** and an
   **Applications** folder.
2. Drag **WOPR** onto **Applications**. Do this before you open WOPR: it may not start from the
   Downloads folder or from the disk image. You can then eject the **WOPR** disk in Finder's sidebar.
3. Open **WOPR** from your **Applications** folder, Launchpad or Spotlight.
4. The first time, macOS says it could not verify that WOPR is free of malware, because WOPR is not
   signed by Apple yet. Click **Done**, then:
   1. Open the Apple menu, then **System Settings**, then **Privacy & Security** in the sidebar (you may
      need to scroll down to it).
   2. Scroll down to **Security**, where it says that WOPR was blocked, and click **Open Anyway**. The
      button is there for about an hour after you tried to open WOPR; if it is gone, open WOPR again.
   3. macOS asks once more: click **Open Anyway** (or **Open**), and enter your password or use
      Touch ID.
5. WOPR opens in a **Terminal** window and starts dialing; the Dock shows Terminal's icon. From then on,
   WOPR opens like any other app.

The app always opens in Terminal, and it does not add a `wopr` command to your terminal: for that, use
the line below. When you update WOPR, drag the new one to Applications in the same way, and macOS may
ask you to allow it again. To remove it, see [Uninstall](/install#uninstall).

**Or install from Terminal**, which gives you the `wopr` command.

@COMMAND-LINE@
