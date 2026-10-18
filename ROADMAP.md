# Roadmap

This roadmap covers October 2026 to October 2027. The maintainer reviews it at each milestone. The
milestones and their details are in [docs/PLAN.md](docs/PLAN.md#15-milestones), and each item below links
its issue when it has one.

WOPR has the [OpenSSF Best Practices](https://www.bestpractices.dev/projects/15394) silver badge.

## Planned

- **v1.0.0.** Finish milestone M5: quality checks of every game on Linux, macOS and Windows, and the
  dependency and runner checklist just before the release.
- **Signed installers.** Sign and notarize the Mac app
  ([#10](https://github.com/GhostofGoes/WOPR/issues/10)), and sign the Windows installer and `wopr.exe`
  ([#14](https://github.com/GhostofGoes/WOPR/issues/14)).
- **Stores and package managers.** Publish to the Snap Store
  ([#11](https://github.com/GhostofGoes/WOPR/issues/11)) and the Microsoft Store
  ([#9](https://github.com/GhostofGoes/WOPR/issues/9)), from v1.0.0. Look into Scoop and Chocolatey
  ([#12](https://github.com/GhostofGoes/WOPR/issues/12)).
- **A start screen** for people who open WOPR from a menu
  ([#13](https://github.com/GhostofGoes/WOPR/issues/13)).
- **A DCO sign-off requirement** for every commit, with a rule for AI-assisted changes
  ([#29](https://github.com/GhostofGoes/WOPR/issues/29)).

## Under consideration

- **Play in a web browser**, on desktop and mobile
  ([#15](https://github.com/GhostofGoes/WOPR/issues/15)).
- **An iOS app.** A demo is in `ios/`. A full app would show every game through the Go core
  (`docs/IOS.md`).
- **Translations** of the parts that are not film text: the docs, the help output, the manual page,
  the package metadata, the installers, the error messages and the games that do not come from the film
  ([#31](https://github.com/GhostofGoes/WOPR/issues/31)). The film's text stays in English.
- **The OpenSSF Best Practices gold badge.** It needs a second significant contributor from another
  organization, a review by a second person for most changes, hardening headers on the docs site, and
  80% branch coverage ([#35](https://github.com/GhostofGoes/WOPR/issues/35)).
- **An optional LLM brain** (milestone M7, `docs/PLAN.md` §4.7). It would be off unless the user turns it
  on, and it would be the only network access in `wopr`.

## Not planned

The project will not do these:

- Collect data or send telemetry.
- Use the network unless the user turns on a feature that needs it.
- Include film stills, audio or video, or change the film's text (`docs/PLAN.md`, R15).
- Support macOS versions older than 26, or 32-bit systems.
- Add accounts, online play between people, or payments.
