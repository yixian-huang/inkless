# Changelog

All notable changes to Inkless CMS are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/) once
stable `1.0.0` is cut. Until then, `0.x` releases may include breaking changes;
those will be called out under **BREAKING**.

## [Unreleased]

### Added

- Open-source repository hygiene: `SECURITY.md`, `CODE_OF_CONDUCT.md`, Dependabot,
  documentation index, public ops summary, and tag-driven `release` workflow.
- Maintainer-only notes under `docs/internal/` (no host inventory secrets).

### Changed

- Public `OPS.md` no longer embeds production IPs or control-plane project IDs.
- Operator-specific runbooks moved under `docs/internal/` with placeholders.

## [0.1.0-alpha.1] - 2026-07

### Added

- Initial public alpha of Inkless CMS (React SPA + Go API): unified pages,
  articles, themes, plugins (beta), AI helpers, Docker and artifact deploy paths.

[Unreleased]: https://github.com/yixian-huang/inkless/compare/v0.1.0-alpha.1...HEAD
[0.1.0-alpha.1]: https://github.com/yixian-huang/inkless/releases/tag/v0.1.0-alpha.1
