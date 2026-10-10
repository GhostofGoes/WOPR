# wopr's icon

The icon is "front-panel lamps", the design the owner chose on 2026-10-09: a gunmetal frame
around a panel of indicator lamps, lit amber along one diagonal, with one red lamp.

Everything here except `src/` is generated. To change the icon, edit or replace the sources,
then run, from the repository root:

```sh
go run ./internal/tools/icons
```

It draws every size from the sources and rewrites the files that changed, here and in
`site/static/`. `go run ./internal/tools/icons -check` fails, listing the files, when they are
out of date, and so does `go test ./...`.

## The sources

All of them are original work. The comment at the top of each says so (its provenance tag).

| File | What it is |
|---|---|
| `src/wopr.svg` | The icon, on a 1024 by 1024 viewBox, with a transparent margin around the art, as Windows and Linux icons have. |
| `src/wopr-small.svg` | Optional. Simpler art for 32 px and below, where fine detail turns to mud. Without it, those sizes come from `wopr.svg`. |
| `src/wopr-<N>.svg` | Optional, as many as needed. Art drawn for exactly N by N pixels, used for that size only, in place of `wopr-small.svg` or `wopr.svg`: for a size where their edges fall between pixels and blur. A viewBox of `0 0 N N` makes each unit one pixel. `src/wopr-24.svg` is `wopr-small.svg` redrawn for 24 px, the Windows taskbar's size at 100% scale. |
| `src/wopr-full.svg` | The art on an opaque square with no margin and square corners, for macOS (`wopr.icns`) and the docs site's `apple-touch-icon.png`. macOS rounds icons itself. It is used at every size, whatever `wopr-<N>.svg` files there are. |

The tool refuses any other SVG file in `src/`, and a `wopr-<N>.svg` for a size it never draws,
since nothing would use either. To see a size's pixels before committing, run the tool and
open the PNG it wrote at that size under `hicolor/` or `msix/`.

The sources may use only a subset of SVG, which the tool draws itself and checks: shapes
(`rect`, `circle`, `ellipse`, `line`, `polyline`, `polygon`, `path`), groups with transforms,
solid colours, linear and radial gradients, strokes, and opacity. Text, filters, masks,
clipping, patterns, images, `use`, CSS and editor metadata are refused with an error naming
the file and line, so convert text to paths and save as plain SVG. The full list is in
`internal/tools/icons/svg.go`.

## The generated files

| File | Used by |
|---|---|
| `wopr.ico` | Windows: `wopr.exe` and the installer (16 to 256 px) |
| `wopr.icns` | macOS: `WOPR.app` (16 to 1024 px, with the Retina sizes) |
| `hicolor/` | Linux: the `.deb`, the `.rpm` and the snap, named `io.github.ghostofgoes.wopr` (16 to 512 px, and the SVG) |
| `msix/` | The MSIX package for the Microsoft Store: the Store logo, the tiles and the app list icons |
| `site/static/favicon.svg`, `favicon.ico`, `apple-touch-icon.png` | The docs site |
