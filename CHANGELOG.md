# Changelog

All notable changes to Inkless CMS are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/) once
stable `1.0.0` is cut. Until then, `0.x` releases may include breaking changes;
those will be called out under **BREAKING**.

## [Unreleased]

## [0.1.0-alpha.2] - 2026-07-25

First tag-driven GitHub Release with build artifacts from
[`.github/workflows/release.yml`](.github/workflows/release.yml).

### Added

- Open-source repository hygiene: `SECURITY.md`, `CODE_OF_CONDUCT.md`, Dependabot
  (grouped updates), documentation index, public ops summary, release workflow.
- Maintainer notes under `docs/internal/` (no host inventory secrets).
- README product screenshots from the live product site.
- Explicit `@testing-library/dom` (peer of jest-dom 7).

### Changed

- Public `OPS.md` no longer embeds production IPs or control-plane project IDs.
- Operator runbooks moved under `docs/internal/` with placeholders.
- **BREAKING (dev):** require **Node.js 22+** (CI + docs); required by
  `@testing-library/jest-dom` 7.
- `@testing-library/jest-dom` 6 → 7.
- TipTap packages aligned to `~3.28.0` with pnpm overrides.
- Dependabot groups: major vs minor/patch; TipTap / CodeMirror / Testing Library.
- E2E: expand default-collapsed Settings nav before ops links; scope ambiguous buttons.

### Assets

On GitHub Releases for this tag:

- `frontend-v0.1.0-alpha.2.tar.gz` (+ `.sha256`) — static SPA (`out/`)
- `backend-v0.1.0-alpha.2.tar.gz` (+ `.sha256`) — API binary package

## [0.1.0-alpha.1] - 2026-07

### Added

- Initial public alpha of Inkless CMS (React SPA + Go API): unified pages,
  articles, themes, plugins (beta), AI helpers, Docker and artifact deploy paths.

[Unreleased]: https://github.com/yixian-huang/inkless/compare/v0.1.0-alpha.2...HEAD
[0.1.0-alpha.2]: https://github.com/yixian-huang/inkless/releases/tag/v0.1.0-alpha.2
[0.1.0-alpha.1]: https://github.com/yixian-huang/inkless/releases/tag/v0.1.0-alpha.1
