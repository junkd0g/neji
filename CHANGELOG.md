# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- CI matrix testing on Go 1.23, 1.24, and 1.25.
- `gofmt`, `go vet`, and `-race` checks in CI.
- `govulncheck` security scan job.
- `CHANGELOG.md` following Keep a Changelog format.

### Changed
- CI workflow renamed from `build.yaml` to `ci.yml`.
- CI actions upgraded: `setup-go@v4` → `v5`, `golangci-lint-action@v3` → `v6`.
- Dropped manual `actions/cache@v3` step — `setup-go@v5` caches modules implicitly.
- Dropped 100%-coverage gate in CI in favour of printing coverage; coverage gating belongs in review.
- README: moved title above badges; added `Why`, `Requirements`, `Testing`, and `Contributing` sections.
- `.gitignore` normalised to match other `junkd0g/*` repos.
