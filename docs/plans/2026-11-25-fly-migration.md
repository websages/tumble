# Migrate tumble.wcyd.org to Fly.io

Status: draft, 2026-11-25

## Goal

Move the production tumble (tumble.wcyd.org, tumble.mandatoryfun.dev)
off the Linode host (172.105.16.86) onto a Fly.io app in the personal
org. Deploy it from `main` via GitHub Actions. Keep SQLite and add
Litestream replication to Tigris. Lose no data. Put it behind a shared
Caddy app that does the AI-bot and abuse filtering for tumble and for
the other sites moving to Fly later.

Out of scope: the `tumble2` app (feed.stahnma.us, `stahnma` branch),
moving the other Linode sites (the edge app is built so they can follow),
and Postgres.

## Current state (verified on the host 2026-11-25)

| Item | Today |
|------|-------|
| Process | `tumble.service`, `/srv/tumble/bin/tumble /srv/tumble/conf/config-sqlite.yml`, port 8083 |
| Database | SQLite, `/srv/tumble/tumble.sqlite` (~103 MB + WAL), integrity ok |
| Rows | ~79.5k links, 7.8k quotes, 8.3k images (URLs only), 17.7k previews; ActivityPub unused |
| Front end | Caddy: TLS, `blockAiBots` user-agent abort, access log |
| Hostnames | `tumble.wcyd.org`, `tumble.mandatoryfun.dev` (both on ns0x.websages.com); `tumble.loserfish.org` is in Caddy but has no DNS record, so it gets dropped |
| Backups | `tumble` user crontab, 05:00 UTC, `/srv/tumble/bin/backup.sh` (host copy rsyncs to `root@odin.websages.com`, unlike the repo copy) |
| Deploy | `.github/workflows/deploy.yml`: scp the binary, `sudo systemctl restart`, IRC notify to #heathsmom |
| Writers | Hubot (67.249.164.162) POSTs `/api/v1/links` and `/api/v1/quotes` with no API key; creates need no key, by design |
| Leftovers | `/srv/tumble/mysql`, `tumble.dmp.*`, `config.yml` (MySQL): no longer used |

There are no other cron jobs, systemd timers or on-disk assets. The
scheduled jobs (archive batch at 03:00, daily kitten at 10:00 Central)
run inside the app.

### Config to carry over

From `config-sqlite.yml`:

- Secrets: `click_signing_key` (copy it **exactly**, or existing signed
  click links break) and `admin_secret`.
- Env: `site_title`/`site_description` = `tumblefish.`,
  `managing_editor`/`webmaster` = `scott@loserfish.org (Scott)`,
  `copyright` = `http://creativecommons.org/licenses/by-sa/1.0`,
  `baseurl` = `https://tumble.wcyd.org`.
- Intentional changes: `mode` goes from `prod` to `production` (JSON logs),
  `logging` from debug to a file to info on stdout, and `request_timeout`
  from the 2s default to 10s, matching tumble2. `daily_kitten` stays on
  (the default).

## Decision: a shared Caddy edge app

Options considered:

1. **Caddy in the same machine as tumble.** Rejected. Caddy would proxy
   from 127.0.0.1, which brings back the localhost auth bypass on
   delete and admin endpoints (`isAuthorizedAPIKey`, `isAuthorizedAdmin`).
   It would also have to be repeated in every app.
2. **Middleware in tumble.** Fine for one app, but every site that moves
   later would need its own copy of the filter.
3. **A separate Caddy edge app (chosen).** One public app (`websages-edge`)
   holds every public hostname and certificate. It runs
   the filtering and proxies to private backend apps over Flycast. The
   backends have no public IPs, so the filter can't be bypassed.
4. **Cloudflare in front.** Not possible for wcyd.org: a friend owns
   that domain, and it can't move off its nameservers
   (ns0x.websages.com), though we can edit its records. stahnma.us is on Cloudflare, but `feed.stahnma.us` is
   DNS-only (it points straight at a Fly IP). If it ever goes behind
   Cloudflare's proxy, the edge would need to read `CF-Connecting-IP`
   for those hosts, and Fly certificates would need the DNS-challenge
   CNAME.

```
client --TLS--> Fly proxy --> edge (Caddy, :8080, 2 machines)
                                  | filter, then reverse_proxy
                                  v
                    tumble-wcyd.flycast (private, no public IPs)
                                  v
                    tumble + Litestream --> Tigris
```

### How the edge app is built

- **Its own repo,** `websages/tumble-edge` (created 2026-11-25; it's
  shared infrastructure, not tumble's), with a
  `Dockerfile` (official `caddy` image), a `Caddyfile`, a `sites/`
  directory with one file per site, and a GitHub Action that deploys on
  push.
- **TLS:** Fly's proxy terminates it, and `fly certs` for every hostname
  live on the edge app. Caddy runs with `auto_https off` and listens on
  plain HTTP on :8080.
- **Real client IP:** Caddy has to trust Fly's proxy, or it will see
  every request coming from Fly's internal address, and per-IP rules
  and logs would be useless:
  ```
  {
      auto_https off
      http_port 8080
      servers {
          trusted_proxies static private_ranges
          client_ip_headers Fly-Client-IP
      }
  }
  import snippets/*.caddy
  import sites/*.caddy
  ```
  Caddy then sends the right `X-Forwarded-For` to backends. tumble's
  request log already reads that header (`pickIPFromXFF`), and nothing
  in tumble's auth trusts it.
- **Site file** (`sites/tumble.caddy`):
  ```
  http://tumble.wcyd.org, http://tumble.mandatoryfun.dev {
      import blockAiBots
      reverse_proxy http://tumble-wcyd.flycast
  }
  ```
- **Bot list:** `snippets/ai-bots.caddy` is the same snippet as on
  Linode today, taken from the ai-robots-txt project. A scheduled
  Action can open a PR when the upstream list changes. Add rate limits
  and other abuse rules here later.
- **Availability:** every site goes down if the edge does. It's
  stateless, so run one machine in each of `iad` and `sjc`
  (`min_machines_running = 2`, `auto_stop_machines = false`).
  Add a `/healthz` handler that responds directly, plus an
  `[[http_service.checks]]` entry for it. The extra hop within `iad`
  adds about a millisecond.
- **Fallback:** a catch-all `:8080` site returns 404 for unknown hosts,
  so a forgotten DNS record doesn't silently reach some backend.

## Plan

### Phase 1: Build the edge app (new repo; no traffic yet)

1. Create the repo with the files above, deploying through
   `fly tokens create deploy -a websages-edge`.
2. `fly apps create websages-edge --org personal`,
   `fly ips allocate-v6 -a websages-edge`, `fly ips allocate-v4 --shared -a websages-edge`.
3. Deploy it with only the catch-all and health check; confirm
   `https://websages-edge.fly.dev/healthz`.
4. Add `sites/tumble.caddy` (it'll return 502 until Phase 3, which is
   fine).

### Phase 2: tumble Fly config and workflow (tumble repo, PR merged at cutover)

1. `fly.toml` on `main` targets the new app (`tumble-wcyd`)
   and gets the env values above. Under `[http_service]`, set
   **`force_https = false`**: edge-to-backend traffic over Flycast is
   plain HTTP, so redirecting to HTTPS there would loop. HTTPS is
   enforced on the edge instead. The `stahnma` branch keeps its own
   `fly.toml` for `tumble2`. (Merging main into `stahnma` will keep
   conflicting on `fly.toml`, as it does today.)
2. `Makefile` `deploy`: allow `main` as well as `stahnma`.
3. Replace `deploy.yml`: on push to `main`, run tests, then
   `make deploy FLY=flyctl` using a new repo secret
   `FLY_API_TOKEN_WCYD` (`fly tokens create deploy -a tumble-wcyd`,
   scoped to this app only). Keep the IRC notify step. Add
   `workflow_dispatch` for manual redeploys.
4. Update `fly.toml`/`Makefile`/workflow comments that say main deploys
   to the Linode host.

### Phase 3: Provision the tumble app (no traffic yet)

```sh
fly apps create tumble-wcyd --org personal
fly volumes create tumble_data -a tumble-wcyd --region iad --size 3
fly storage create -a tumble-wcyd          # Tigris; sets BUCKET_NAME + AWS_* secrets
fly secrets set -a tumble-wcyd --stage \
  TUMBLE_ADMIN_SECRET=... TUMBLE_CLICK_SIGNING_KEY=...   # from config-sqlite.yml, never echoed
# Unfurl worker: reuse tumble2's (it only checks a shared key). Fly secrets
# can't be read back, so pipe them from the running tumble2 machine:
#   fly ssh console -a tumble2 -C 'printenv TUMBLE_UNFURL_WORKER_URL TUMBLE_UNFURL_WORKER_KEY'
#   -> fly secrets set -a tumble-wcyd --stage TUMBLE_UNFURL_WORKER_URL=... TUMBLE_UNFURL_WORKER_KEY=...
fly ips allocate-v6 --private -a tumble-wcyd   # Flycast address only
```

Give it **no public IPs**: `fly ips list -a tumble-wcyd` must show only
the private address. The first `fly deploy` can allocate public ones on
its own; release them if it does.

**Don't deploy until the bucket is seeded (Phase 4).** On an empty
volume with an empty bucket, tumble creates a blank database and
Litestream replicates it.

### Phase 4: Rehearsal (repeatable; this is also the cutover procedure)

Seed the bucket so the first boot restores the real data through the
same path a disaster recovery would use.

1. Snapshot on the Linode host (safe while tumble is running):
   `sqlite3 /srv/tumble/tumble.sqlite ".backup /tmp/tumble-snap.sqlite"`,
   then copy it to the laptop and run `PRAGMA integrity_check`.
2. Seed Tigris from the laptop with **Litestream 0.5.14** (it must
   match the image): run `litestream replicate` with a config pointing
   at the snapshot and the app's bucket credentials, wait for the
   snapshot upload, then stop it.
3. Verify: `litestream restore -o check.sqlite` from the bucket;
   integrity check; compare `count(*)` on `ircLink`, `quote`, `image`
   against the snapshot.
4. `fly deploy -a tumble-wcyd` (with `GIT_COMMIT` build arg). The
   entrypoint restores onto the empty volume and starts.
5. Smoke test.
   - **Backend directly:** `fly proxy 8080 -a tumble-wcyd`, then hit
     `localhost:8080`. Note that the tunnel counts as localhost, so the
     admin bypass applies there and you can't test the key this way.
   - **Through the edge**, once the certificates exist (Phase 5.2):
     `curl --resolve tumble.wcyd.org:443:<edge v4> https://tumble.wcyd.org/`
   - **Checks:** front page, a permalink, a signed click link copied
     from production (checks the signing key), search, RSS,
     `/api/docs`, a POST to `/api/v1/links`, a DELETE without the key
     (must be 403) and then with it, a bot UA (blocked by the edge),
     the log showing the real client IP, and `fly logs` for the
     scheduler start line.

To reset for another rehearsal or for the real cutover:
`fly machine destroy`, `fly volumes destroy`, delete the `tumble/` prefix
in the bucket, `fly volumes create`, and seed again.

### Phase 5: Pre-cutover (a day before)

1. Lower the TTL on `tumble.wcyd.org` and `tumble.mandatoryfun.dev`
   to 60s (currently ~900s).
2. Issue the certificates ahead of time on the **edge** app:
   `fly certs add tumble.wcyd.org -a websages-edge` (likewise for
   mandatoryfun.dev), then create the `_acme-challenge` CNAMEs Fly
   prints.
3. Make a dedicated old-host Caddy file that proxies to the edge, ready
   but not enabled:
   ```
   tumble.wcyd.org, tumble.mandatoryfun.dev {
       reverse_proxy https://websages-edge.fly.dev {
           header_up Host {host}
           transport http { tls_server_name {host} }
       }
   }
   ```
   (The edge routes by hostname, so the original Host header and SNI
   must be passed through.)

### Phase 6: Cutover (~15 min of read-only/down time)

1. On Linode: `sudo systemctl stop tumble`. Writes stop here, and Hubot
   posts made during the window are lost.
2. Snapshot, then reset and seed (Phase 4, steps 1-4) with this final copy.
3. Smoke test via `--resolve` to the edge; row counts must match the
   final snapshot exactly.
4. Swap the old Caddy to the proxy config (`caddy reload`). Stragglers
   on stale DNS now reach Fly.
5. Change the A/AAAA records for both hostnames to the **edge** IPs.
6. Merge the Phase 2 PR. Its push to `main` deploys via the new
   workflow, which proves CI works end to end.
7. `sudo systemctl disable tumble`; remove the `tumble` crontab entry.
8. Watch `fly logs -a websages-edge` and `-a tumble-wcyd`, and look for Hubot's
   next POST arriving.

**Rollback** (before decommissioning): revert Caddy to `reverse_proxy
127.0.0.1:8083`, start `tumble.service`, and point DNS back. Anything
posted on Fly since cutover has to be pulled back first
(`fly ssh sftp get` of a `.backup` copy).

### Phase 7: Backups after cutover

- Litestream to Tigris (continuous) plus Fly's daily volume snapshots
  (5-day retention).
- No off-Fly copy (decided 2026-11-25). The odin rsync stops when the
  Linode cron is removed.
- Test a restore once after cutover.

### Phase 8: Decommission (after ~2 weeks stable)

- Final archive of `/srv/tumble` (DB, configs, logs), then
  remove `/srv/tumble`, `tumble.service`, the `tumble.caddyfile` site,
  and the `tumble` user's sudo rule. The other Linode sites stay until
  they move behind the edge.
- Delete the `DEPLOY_SSH_KEY` GitHub secret.
- Repo cleanup: `contrib/*.service`, `scripts/backup.sh`, and the MySQL-era docs if wanted.
- Raise the DNS TTLs back up.

### Later: moving other sites behind the edge

For each site: a private app (Flycast address only, `force_https = false`),
a `sites/<name>.caddy` file, `fly certs add <host> -a websages-edge`, then DNS to
the edge IPs. tumble2 (feed.stahnma.us) can move behind it the same way.

## Decisions (2026-11-25)

- Names: `websages-edge` (Fly app), `tumble-wcyd` (Fly app), `websages/tumble-edge` (repo).
  (`edge` was taken; Fly app names are global.)
- Backups: Litestream to Tigris only, plus Fly volume snapshots.
- Unfurl worker: wcyd reuses the existing `tumble-unfurl` worker and key.
- Edge: two machines, `iad` and `sjc`.
- DNS: we edit records for wcyd.org and mandatoryfun.dev directly.
- Delete/admin endpoints: nothing known calls them without a key. Any
  caller that breaks on Fly gets fixed to send `X-API-Key`.
