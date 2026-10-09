[**Download the WOPR installer**](https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@_windows_setup.exe)

One installer for every Windows PC, Intel, AMD or Arm. Then:

1. Open the file you downloaded, `wopr_@VERSION@_windows_setup.exe`: click it in your browser's
   downloads, or double-click it in your **Downloads** folder. If the browser says the file is not
   commonly downloaded, choose to keep it.
2. If Windows says **Windows protected your PC**, click **More info**, then **Run anyway**. It warns
   because the installer is not signed yet, which is also why it says **Unknown publisher**.
3. Click **Next**, then **Install**, then **Finish**. It does not ask for an administrator's password.
   It also adds WOPR to your `PATH`, so that typing `wopr` in a terminal starts it, unless you untick
   that box on the first page.
4. WOPR opens in a terminal window and starts dialing.
5. To play again later, open the **Start** menu and click **WOPR** in the list of apps, or search for
   it.

If Windows says that **Smart App Control** blocked the installer, it cannot run while Smart App Control
is on, because it is not signed yet. [Troubleshooting](/usage/troubleshooting) says what you can do.

To update WOPR, run a newer installer the same way: it replaces the old version. To remove it, see
[Uninstall](/install#uninstall).

**Or install from PowerShell.**

@COMMAND-LINE@
