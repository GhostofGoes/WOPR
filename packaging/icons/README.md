# wopr's icon

Everything here except `src/` is generated. To change the icon, edit or replace the sources,
then run, from the repository root:

```sh
go run ./internal/tools/icons
```

It draws every size from the sources and rewrites the files that changed, here and in
`site/static/`. `go run ./internal/tools/icons -check` fails, listing the files, when they are
out of date, and so does `go test ./...`.

## The sources

All three are original work. The comment at the top of each says so (its provenance tag).

| File | What it is |
|---|---|
| `src/wopr.svg` | The icon, on a 1024 by 1024 viewBox, with a transparent margin around the art, as Windows and Linux icons have. |
| `src/wopr-small.svg` | Optional. Simpler art for 32 px and below, where fine detail turns to mud. Without it, those sizes come from `wopr.svg`. |
| `src/wopr-full.svg` | The art on an opaque square with no margin and square corners, for macOS (`wopr.icns`) and the docs site's `apple-touch-icon.png`. macOS rounds icons itself. |

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
