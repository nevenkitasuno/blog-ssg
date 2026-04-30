# Repository Guidelines

## Project Structure & Module Organization
`cmd/ssg/main.go` is the CLI entrypoint for the static site generator. Core logic lives under `internal/`: `app/site` coordinates builds, `infra/content` loads Markdown and assets from `content/`, `infra/render` renders `templates/`, `infra/state` tracks incremental build state, and `domain` holds shared models. Source content belongs in `content/<Topic>/<YYYY MM Title>/`, templates live in `templates/`, and generated site files are written to `output/`.

## Build, Test, and Development Commands
Use `go run ./cmd/ssg` to build the site with default paths. Use `go run ./cmd/ssg -content content -templates templates -output output` when testing alternate directories or scripts. Run `go test ./...` before opening a PR; the repository currently has no test files, but this command still validates package compilation. Use `go build ./cmd/ssg` to verify the CLI builds cleanly.

## Coding Style & Naming Conventions
Follow standard Go formatting with tabs and `gofmt`; run `gofmt -w` on edited `.go` files. Keep packages small and focused under `internal/`. Exported identifiers use `CamelCase`; unexported helpers use `camelCase`. Prefer descriptive names aligned with the existing codebase, such as `Loader`, `Renderer`, `ManifestStore`, and `Build`.

## Testing Guidelines
Add `_test.go` files beside the package they cover. Prefer table-driven tests for parsing, slug generation, manifest updates, and rendering edge cases. For content-related changes, validate both `go test ./...` and a real build with `go run ./cmd/ssg`, then inspect the affected files under `output/`.

## Commit & Pull Request Guidelines
The current history is minimal (`Initial commit`), so use short, imperative commit subjects such as `Add tag index generation` or `Fix markdown asset copying`. Keep commits scoped to one change. PRs should include a brief description, manual verification steps, and screenshots when template or generated HTML output changes.

## Content & Output Notes
Treat `output/` as generated artifacts: edit `content/`, `templates/`, or Go code, then regenerate. Keep content folder names consistent with the documented pattern `YYYY MM Title`, and place entry assets, such as `image.png`, inside the matching entry directory.
