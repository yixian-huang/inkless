# Theme demo site (operator notes)

**Public URL (example):** `https://themes.inkless.run`  
**Role:** Public theme showcase instance (isolated from personal and product sites)

Resolve the real host with `npc server list` / handoff — do not hard-code IPs here.

| Item | Example value |
|------|----------------|
| Container | `inkless-theme-demo` |
| Image | `inkless-theme-demo:latest` |
| Port | `127.0.0.1:8098` → container `:8088` |
| Data | `/opt/inkless-theme-demo/{data,uploads,backups}` |
| Proxy | hostname → `127.0.0.1:8098` |
| Admin | `/admin` — change default seed credentials immediately |
| Active theme | e.g. `editorial-firm` |

## Ops commands (pattern)

```bash
npc server exec command <server-ref> -- \
  'docker ps --filter name=inkless-theme-demo; curl -sS http://127.0.0.1:8098/health'

npc server exec command <server-ref> -- \
  'docker logs --tail 100 inkless-theme-demo'
```

Rebuild: pull sources on the host, `docker build`, recreate container with the
same volume mounts and publish flags as the first deploy.

Switch theme via admin UI or `PUT /admin/themes/{id}/activate` with an admin JWT.

## Notes

- Blank-site seed may not insert `installed_themes` rows; first activate needs
  themes present (seed once for the demo).
- Keep this instance’s DB and JWT secrets separate from personal and product sites.
