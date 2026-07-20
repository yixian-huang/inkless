# Inkless Brand Migration

This guide records the supported compatibility surface for the one-time
canonical brand switch to Inkless CMS and the external production steps that
must be executed by an authorized operator.

## Canonical values

| Surface | New value |
| --- | --- |
| Product | Inkless CMS |
| CLI | `inkless` |
| Service | `inkless` |
| API binary | `inkless-api-{version}` and `inkless-api-latest` |
| SQLite database | `inkless.db` |
| PostgreSQL database/user | `inkless` |
| Install root | `/opt/inkless` |
| Domain | `https://inkless.run` |
| Repository | `https://github.com/yixian-huang/inkless` |

## Compatibility matrix

| Legacy surface | Compatibility behavior | New output behavior | Removal gate |
| --- | --- | --- | --- |
| `IMPRESS_ENV_FILE` | Read only when `INKLESS_ENV_FILE` is unset and emit one deprecation warning. | Generated examples and docs use `INKLESS_ENV_FILE`. | Remove after one announced compatibility release. |
| `IMPRESS_SECRET_KEY` | Read only when `INKLESS_SECRET_KEY` is unset and emit one deprecation warning. | Generated examples and docs use `INKLESS_SECRET_KEY`. | Remove after one announced compatibility release. |
| `__IMPRESS_SHARED__` / `__IMPRESS_THEME_REGISTER__` | Runtime aliases keep existing external bundles loadable. | Built-in bundles write only `__INKLESS_SHARED__` and `__INKLESS_THEME_REGISTER__`. | Remove after old plugin fixtures are no longer supported. |
| `impress.setup.step`, `impress.setup.draft`, `impress.comment.guest` | Browser startup migrates valid legacy values to `inkless.*` keys, then deletes the old keys. | New browser sessions write only `inkless.*` keys. | Keep migration until telemetry or support policy confirms old clients have upgraded. |
| `.impress-storage-probe` | Startup may delete only this completed probe file. It must not delete uploads or real user files. | New probes use `.inkless-storage-probe`. | Keep cleanup indefinitely because it is harmless and bounded. |
| `IMPRESS_PLUGIN=impress-cms-v1` | Host preflights compiled binaries and selects this deprecated handshake only when both legacy markers are present and canonical markers are absent. Unknown or broken plugins are started once with the canonical handshake, so runtime errors cannot trigger a second launch. | SDK examples generate `INKLESS_PLUGIN=inkless-cms-v1`. | Remove only after legacy plugin fixture support is explicitly dropped. |
| `X-Impress-*` webhook headers | Transition releases send both old and `X-Inkless-*` headers with matching values. | Documentation and new consumers should use `X-Inkless-Event`, `X-Inkless-Timestamp`, and `X-Inkless-Signature`. | Remove after one announced webhook compatibility window. |
| `impress_` search indexes | The database upgrade writes an explicit `impress_` prefix only into existing Meilisearch records that had no prefix; those records are reindexed before cutover. | New Meilisearch configs default to `inkless_`. | Keep read support until all production indexes have been verified and cut over. |
| `/opt/impress`, `/opt/blotting`, old service units | Treated only as migration sources or rollback references. Scripts do not recursively delete them. | New bootstrap creates `inkless` user, `/opt/inkless`, and `inkless.service`. | Clean up in a separate authorized maintenance task after backup retention expires. |

## Production migration runbook

These steps are intentionally operational instructions, not actions performed by
the repository migration.

1. Freeze deployments and export the current database, PublishedConfig, plugin
   configuration, environment file, Nginx config, systemd unit, and release
   symlink target.
2. Back up uploads and record checksum samples for representative files.
3. Provision `/opt/inkless` with `ops/qb-host-bootstrap.sh` or the equivalent
   commands from `ops/qb-init-hk-artifact.json`.
4. Copy or restore data into `/opt/inkless/data/inkless.db`; do not move or
   delete the old database path during cutover.
5. Deploy Inkless artifacts and install `ops/systemd/inkless.service`.
6. Export the operating site's current PublishedConfig before changing it:

   ```bash
   curl --fail-with-body \
     -H "Authorization: Bearer ${INKLESS_ADMIN_TOKEN}" \
     "${INKLESS_ADMIN_ORIGIN}/admin/global-config" \
     > "published-config-before-$(date -u +%Y%m%dT%H%M%SZ).json"
   ```

   Review `ops/inkless-site-config.example.json`, preserving any intentional
   site-specific fields. Put it as the next draft with the exported
   `draftVersion`, publish it, then GET the config again and verify that
   `publishedVersion` increased by exactly one. Keep both JSON responses as
   rollback inputs. This example targets the operated `inkless.run` site only;
   it is not a global default for user-created sites.
7. Start the service locally and verify `/health`, admin login, content reads,
   uploads, plugins, webhooks, and search before switching public traffic.
8. Point DNS for `inkless.run` and `www.inkless.run` to the production host,
   issue certificates, then install `nginx.conf`.
9. Set `BASE_URL=https://inkless.run` and include the actual frontend origin in
   `CORS_ALLOWED_ORIGINS`.
10. Smoke test HTTPS, canonical redirects, sitemap/RSS/canonical/OG URLs,
   `/admin`, `/api/*`, `/uploads/*`, and client route refreshes.
11. Keep old directories and units available for rollback through at least one
    release observation window.

Rollback restores the saved Nginx config and previous service/symlink/database
backup. Rollback must not delete `/opt/inkless`; leave it available for
diagnosis.

## External identity cutover snapshot

Read-only checks performed at `2026-07-20T02:41Z` after the GitHub identity
cutover, with `main` still at commit `a8b6290`, found the following external
state:

| Surface | Observed state | Cutover implication |
| --- | --- | --- |
| Domain registration | `inkless.run` was registered through Spaceship on 2026-07-19, expires 2027-07-19, and has transfer lock enabled. | Registration is complete; no purchase or transfer is required. |
| DNS delegation | Authoritative nameservers are `elsa.ns.cloudflare.com` and `donovan.ns.cloudflare.com`. | Cloudflare is the DNS control plane. |
| DNS records | The authoritative nameservers return no apex A/AAAA and no `www` A/AAAA/CNAME. | The domain does not currently route to a host. |
| TLS/HTTP | Neither apex nor `www` resolves, so no HTTPS endpoint or served certificate can be validated. | DNS and certificate issuance remain gated external actions. |
| GitHub | `yixian-huang/inkless` is public at `a8b6290`; the previous web URL returns HTTP 301 to the new repository, and Git operations through both clone URLs resolve the same HEAD. The maintained local remote points to the new name. | Repository rename and redirect validation are complete. Do not reuse the old repository name while compatibility redirects are needed. |
| GitHub Actions | Quality Gate succeeded for `a8b6290`. Automatic Deploy built artifacts but failed because the remote Inkless destination directory did not exist. | Bootstrap `/opt/inkless` before enabling another automatic deployment. |
| GitHub variables | `AUTO_DEPLOY_ENABLED=false`, `BACKEND_SERVICE=inkless`, `BACKEND_HEALTH_URL=http://127.0.0.1:8088/health`, `DEPLOY_ENVIRONMENT=production`. | Identity variables are cut over. Keep automatic deployment disabled until the origin preflight succeeds. |
| GitHub secrets | Actions has `DEPLOY_HOST`, `DEPLOY_KNOWN_HOSTS`, `DEPLOY_ROOT`, `DEPLOY_SSH_PRIVATE_KEY`, and `DEPLOY_USER`; values are unreadable by design. | An authorized operator must verify that `DEPLOY_ROOT` is `/opt/inkless`. |
| GitHub Pages/badges/self-actions | No Pages site, badges, or `uses: yixian-huang/impress@...` references were found. | Normal repository redirects are sufficient; the non-redirecting renamed-action exception does not currently apply. |
| Releases/tags | No GitHub release exists. The only tag is root tag `v0.1.0-alpha.1`. | Release history will follow the repository rename, but this tag does not publish the module located in `backend/`. |
| Go module | Source declares `github.com/yixian-huang/inkless/backend`; the repository now exists at that path, but no `backend/v*` tag exists. | Publish a `backend/vX.Y.Z` tag only after separate release authorization; a root `vX.Y.Z` tag is not a version of a subdirectory module. |
| npm | The root, frontend, and docs packages are private. Registry lookups for `inkless`, `@inkless/web`, and `@inkless/cms` return 404; this machine is not authenticated to npm. | npm publication is not required for the product cutover. Reserve an npm scope only if a public SDK/package release is approved. |
| Public old links | GitHub code search found no indexed external references to `yixian-huang/impress`. | There is no known consumer migration list, but private clones and bookmarks may still exist. |
| Existing operated site | `yx.ink` resolves through Cloudflare but returned HTTP 502 during the check. Direct health checks to both `103.73.220.161:8088` and the older `82.158.226.66:8088` endpoint were not reachable. | No currently documented host is ready to receive `inkless.run` traffic. Restore origin health before any DNS write. |

Re-run the status probe at any time:

```bash
./scripts/check-external-identity.sh --status
```

After the external actions, require the cutover assertions:

```bash
./scripts/check-external-identity.sh --expect-cutover
```

## Repository rename impact matrix

| Consumer | GitHub behavior after rename | Required follow-up | Rollback |
| --- | --- | --- | --- |
| Repository web URLs, issues, wiki, stars, watchers | GitHub redirects the previous repository URL to the renamed repository. | Verify old and new URLs, Issues, commit permalinks, archives, and raw file URLs. | Rename the same repository back only if cutover validation fails; do not create a new repository under the old name because that removes redirects. |
| Existing `git clone`, fetch, and push URLs | GitHub continues serving Git operations through the old location. | Run `git remote set-url origin https://github.com/yixian-huang/inkless.git` in maintained clones and deployment systems. | Set maintained remotes back to the old URL if the repository name is rolled back. |
| Actions workflows in this repository | Workflows, secrets, variables, run history, and artifacts remain attached to the same repository object. The runner checkout path changes from `.../impress/impress` to `.../inkless/inkless`; current workflows use relative paths. | Disable automatic deployment before rename, update external variables, then manually run Quality Gate before re-enabling deployment. | Restore the previous variable values and repository name. |
| Reusable actions consumed as `uses: owner/repository@ref` | GitHub does not redirect actions hosted in a renamed repository. | None currently: no self-hosted action consumer was found. Re-run GitHub code search before rename. | If such a consumer appears, keep a compatibility action repository under the old name instead of relying on rename redirects. |
| Badges and Pages | No badges or Pages configuration currently exist. | Add new-name badges only after the rename. The operated site remains `inkless.run`, not a GitHub Pages URL. | Remove new badges or restore their old URLs if the rename is rolled back. |
| Releases and artifacts | Existing tags and release objects stay on the same repository; Actions artifact names are already brand-neutral or Inkless. | Create future releases only after rename. Keep binary names `inkless` and `inkless-api-*`. | Existing objects follow a repository-name rollback. |
| Go module consumers | The declared module path resolves only when the new GitHub repository exists. Go proxy content is immutable once published. | Create and push a submodule tag such as `backend/v0.1.0-alpha.1`, then run the proxy verification command below. Never move a published tag. | Before publication, delete a local mistaken tag. After publication, publish a new version; do not retarget the old tag. |
| Documentation and QuickBox | Current files use the new clone URL, which now resolves to the renamed repository. | Keep README, docs-site, `ops/qb-init-hk-artifact.json`, and deployment clone operations on the canonical URL. | A repository-name rollback restores the old endpoint, but current docs would need a reverting commit. |

## External authorization gate

The following operations change external state and must not be run without
explicit authorization.

### GitHub repository and Actions

Completed on 2026-07-20 under explicit authorization:

- Renamed `yixian-huang/impress` to `yixian-huang/inkless`.
- Set Actions variable `AUTO_DEPLOY_ENABLED=false`.
- Set Actions variable `BACKEND_SERVICE=inkless`.
- Updated the maintained local remote to `https://github.com/yixian-huang/inkless.git`.
- Verified the previous web URL returns HTTP 301 and both old and new Git URLs
  resolve `main` at `a8b6290`.

The following repository metadata remains optional and was not part of that
authorization:

Exact objects:

- Set repository description to `Inkless CMS — a self-hosted bilingual publishing and site-building system`.
- Set repository homepage to `https://inkless.run` and topics to include `cms`, `self-hosted`, `golang`, `react`, and `bilingual`.
- Verify secret `DEPLOY_ROOT` contains `/opt/inkless`; secret values must not be printed.

### DNS, origin, and TLS

The DNS record names and types are fixed, but the target address is not yet
safe to choose. Repository evidence names two different hosts: the older
`OPS.md` endpoint at `82.158.226.66` and a newer gomami migration target at
`103.73.220.161`. Neither direct health endpoint was reachable during the
check, while the existing `yx.ink` Cloudflare route returned 502.

Pending DNS objects:

- `A inkless.run -> <authorized healthy origin IPv4>`
- `CNAME www.inkless.run -> inkless.run`
- Optional `CAA inkless.run -> 0 issue "letsencrypt.org"`

Do not write the apex A record until the operator selects a host and that host
passes localhost and direct-origin health checks. Start as DNS-only while
validating the origin and issuing the certificate expected by `nginx.conf`;
Cloudflare proxying can be enabled only after origin HTTPS works.

Before DNS writes:

1. Bootstrap `/opt/inkless` and install `inkless.service` without deleting the old deployment.
2. Put the Inkless environment file and data in place and verify localhost `/health`.
3. Issue a certificate for both `inkless.run` and `www.inkless.run` after DNS resolves.
4. Validate Nginx configuration, then reload it.

DNS rollback restores the previous record set (currently no address records),
restores the previous Nginx config, and routes traffic back to the previous
service. Do not delete the Inkless release or migrated data during rollback.

### Go module publication

After the repository rename and a green Quality Gate:

```bash
git tag backend/v0.1.0-alpha.1 <verified-commit>
git push origin backend/v0.1.0-alpha.1
GOPROXY=https://proxy.golang.org go list -m \
  github.com/yixian-huang/inkless/backend@v0.1.0-alpha.1
```

If the version should represent changes after `a8b6290`, choose a later
prerelease instead of reusing `v0.1.0-alpha.1`. Once fetched by a public proxy,
the tag must never be moved.

### npm reservation or publication

No npm action is required for the current private workspace packages. If a
public SDK is approved later, create or verify the `inkless` npm organization,
authenticate with an authorized publisher, change only the publishable package
to an organization-scoped name, add `publishConfig.access=public`, and publish
with provenance and 2FA. A registry 404 does not by itself prove that the npm
organization scope is available.

## Authorized cutover order

1. **Complete:** freeze deployments; set `AUTO_DEPLOY_ENABLED=false`.
2. Bootstrap `/opt/inkless`, install `inkless.service`, and verify the origin locally.
3. **Partially complete:** `BACKEND_SERVICE=inkless`; `DEPLOY_ROOT` still requires operator verification without printing its value.
4. **Complete for identity:** rename the GitHub repository, update the maintained remote, and validate old/new redirects. Repository description, homepage, and topics remain optional metadata follow-up.
5. Run Quality Gate manually. Do not re-enable deployment until the remote root preflight passes.
6. Publish the correctly prefixed Go module tag only if public SDK consumption is intended.
7. Write DNS records, issue the two-name certificate, install Nginx, and run HTTPS/canonical/CORS smoke tests.
8. Apply the operated site's PublishedConfig and verify sitemap, RSS, canonical, OG, favicon, admin, API, uploads, plugins, webhooks, and search.
9. Run `./scripts/check-external-identity.sh --expect-cutover` and retain its output as release evidence.
10. Re-enable automatic deployment only after a manual release and rollback exercise succeeds.
