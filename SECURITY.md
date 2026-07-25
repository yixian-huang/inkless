# Security Policy

## Supported versions

Inkless CMS is in early public development (`0.x` / alpha). Security fixes are
applied on the default branch (`main`) and included in the next tagged release
when practical.

| Version | Supported |
|---------|-----------|
| `main` (unreleased) | ✅ |
| Latest `v0.*` GitHub Release | ✅ best-effort |
| Older alpha tags | ❌ unless noted in the release notes |

## Reporting a vulnerability

**Please do not open a public GitHub Issue for security vulnerabilities.**

Prefer one of:

1. **GitHub Security Advisories** — use
   [Report a vulnerability](https://github.com/yixian-huang/inkless/security/advisories/new)
   on this repository (private to maintainers until disclosed).
2. **Email** — contact the maintainer listed on
   [inkless.run](https://inkless.run) or the GitHub profile of the repository owner,
   with subject `[SECURITY] inkless …`.

Include, when possible:

- Affected version / commit SHA  
- Impact (auth bypass, RCE, data leak, etc.)  
- Reproduction steps or a minimal PoC  
- Whether you plan a public disclosure date  

We aim to acknowledge reports within **7 days** and to provide a remediation
plan or fix timeline after triage. Complex issues may take longer; we will keep
you informed.

## Safe harbor

Good-faith research that:

- avoids privacy violations, service degradation, and data destruction  
- does not access data that is not your own  
- reports findings privately first  

…is appreciated. Do not use automated scanning that generates significant load
against production demo or third-party sites without permission.

## Hardening checklist (operators)

When self-hosting:

- Set strong unique `JWT_SECRET` and `JWT_REFRESH_SECRET` (never use example values).  
- Change default seed admin passwords immediately (`admin` / `admin123` is **dev only**).  
- Keep `ENABLE_EXTERNAL_PLUGINS=false` unless you trust every installed package
  (external plugins run as trusted server-side code; manifest permissions are not
  an OS sandbox).  
- Restrict admin origins via `CORS_ALLOWED_ORIGINS` and put TLS at the reverse proxy.  
- Back up the database and `uploads/` on a schedule; test restore.  
- Run as a non-root service user; limit filesystem permissions on data dirs.  

See [`docs/deployment.md`](docs/deployment.md) and [`.env.example`](.env.example).
