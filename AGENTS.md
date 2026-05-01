# Repository Guidelines

## Project Structure & Module Organization
`cmd/ssg` contains the CLI entrypoint for the static site generator. Core application logic lives under `internal/`: `internal/app/site` builds pages, `internal/infra/content` loads Markdown content and topic metadata, `internal/infra/render` handles HTML templates, and `internal/infra/state` tracks incremental build state. Source content lives in `content/`, templates in `templates/`, and generated site files in `output/`. Do not hand-edit `output/`; regenerate it from source.

## Build, Test, and Development Commands
Use `go run ./cmd/ssg` to build the site with default paths. Override directories when needed with `go run ./cmd/ssg -content content -templates templates -output output`. Run `go test ./...` to execute all unit tests. Use `go build ./cmd/ssg` for a compile-only verification of the CLI. GitHub Pages deploys the contents of `output/` from `.github/workflows/static.yml`.

## Coding Style & Naming Conventions
Follow standard Go formatting: run `gofmt -w` on changed files before submitting. Keep package names short and lowercase. Exported identifiers use `CamelCase`; unexported helpers use `camelCase`. Tests in this repo prefer table-driven cases where behavior branches, with clear failure messages such as `got`/`want`. For content directories, follow the documented pattern `content/<Topic>/<YYYY MM Title>/1.md`; an optional day segment is supported as `YYYY MM DD Title`.

## Testing Guidelines
Place tests next to the code they cover as `*_test.go`. Current coverage centers on `internal/app/site` and `internal/infra/content`; extend those suites when changing rendering, slugging, content parsing, or topic metadata behavior. Prefer `t.TempDir()` for filesystem fixtures and small inline Markdown samples to exercise parser edge cases. Run `go test ./...` before opening a PR.

## Commit & Pull Request Guidelines
Recent history uses short Conventional Commit prefixes such as `feat:`, `fix:`, and `docs:`. Keep commit subjects imperative and specific, for example `fix: preserve custom inline tag output`. PRs should describe the user-visible change, note any content or template migrations, link related issues, and include screenshots when HTML or theme output changes.

## Content & Deployment Notes
Topic assets and metadata belong in `content/<Topic>/meta/` (`Links.md`, `Theme.yaml`, images). Because deployment uploads `./output` directly, verify generated pages locally before merging.
