# Host updates (self-hosted)

There are two ways to get a new Inkless **host** (API + SPA) version:

| Path | When |
|------|------|
| **Control-plane / scripts** | First install, infra changes, multi-instance orchestration |
| **Admin → System status → About & updates** | Optional: apply a published GitHub Release to **this** instance |

## Release artifacts

Tags like `v0.1.1` publish:

- `backend-*.tar.gz` + `.sha256`
- `frontend-*.tar.gz` + `.sha256`

via [`.github/workflows/release.yml`](https://github.com/yixian-huang/inkless/blob/main/.github/workflows/release.yml).

Daily `main` deploys may use a different directory name (e.g. `main-<sha>`);
in-admin update is designed around **SemVer Release** packages.

## Enabling in-admin update (optional)

Per instance (example):

```bash
INKLESS_SELF_UPDATE_ENABLED=true
INKLESS_RELEASE_ROOT=/opt/inkless   # this instance only
INKLESS_SYSTEMD_UNIT=inkless
INKLESS_UPDATE_CHANNEL=stable
```

Requires writable release-tree paths and a safe way to restart the unit (see
public [OPS.md](https://github.com/yixian-huang/inkless/blob/main/OPS.md)).
**Default is off.**

## Themes vs host

| | Theme market auto-update | Host self-update |
|--|--------------------------|------------------|
| Updates | Theme UMD package pointer | Binary + SPA under release root |
| Restart | No | Yes |
| Admin UI | Theme market | System status |

## Related

- [Getting started](/guide/getting-started)
- [Theme market](/guide/theme-market)
