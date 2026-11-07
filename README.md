# Tumble

The canonical installation of tumble is found at [http://tumble.wcyd.org](http://tumble.wcyd.org).

## History

Tumble was a "Wouldn't it be cool?" project handed to [Scott Schnedier](https://github.com/sschneid) back in 2004. The idea was to create a website similar to a tumbleblog. Eventually tumblr came along and the rest was history.

The project was originally written in Perl and has since been rewritten in Go.

# Developer Guide

## Quick Start

This project uses [Flox](https://flox.dev) to manage its development environment. The Flox environment provides Go, make, jq, SQLite, ImageMagick, and libxml2.

```bash
# Activate the development environment
flox activate

# Build
make build

# Configure (edit to match your environment)
cp conf/config.yaml.sqlite conf/config.yaml
vi conf/config.yaml

# Run
make restart
```

Tumble will automatically create database tables on first startup.

## Configuration

Tumble uses [Viper](https://github.com/spf13/viper) for configuration via YAML files, environment variables, or defaults.

**Configuration search paths:** `conf/`, current directory (`.`)

### SQLite (recommended for development)

```yaml
driver: sqlite
database: tumble.db
baseurl: your.domain.com
port: 8080
request_timeout: 2s
```

### MySQL

```yaml
driver: mysql
database: tumble
username: tumble
password: your_secure_password
host: localhost
port: 8080
baseurl: your.domain.com
```

### Site Branding

`site_name` sets the wordmark shown in the masthead and page titles (e.g.
`tumblefish.stats`). It is independent of the RSS `site_title` below.

```yaml
site_name: "tumblefish"
```

`favicon` replaces the site icon. It's used for both the browser icon and
the Apple touch icon, and `/favicon.ico` and `/apple-touch-icon.png`
redirect to it. Use a URL or a site path; a square PNG of 180x180 works
everywhere.

```yaml
favicon: "https://example.com/icon.png"
```

If `favicon` is unset, the theme's icon is used when it ships one
(`corkboard` does), otherwise the built-in icons.

### Themes

`theme` picks a site-wide visual theme. Leave it unset for the default look.

```yaml
theme: "broadsheet"   # or "irc", "corkboard"
```

| Theme | Look |
|-------|------|
| `broadsheet` | A newspaper front page: blackletter nameplate and dateline, three ruled columns, a lead story each day, halftone photos, pull quotes |
| `irc` | An IRC client: channel sidebar, topic bar, timestamped log lines with colored nicks, a Hot Shit panel, and an input line that searches |
| `corkboard` | A pinboard: label-maker masthead, pinned index cards, sticky-note quotes, taped polaroids, tear-off calendar dates |

Every theme supports the dark mode toggle. Scott Mode is only available on
the default look; its buttons are hidden when a theme is set. An unknown
theme name logs a warning and falls back to the default look.

#### Writing a theme

A theme is a directory in `internal/assets/themes/<name>/`, served at
`/themes/<name>/`. Adding one needs no Go changes, but it is embedded in the
binary, so rebuild to pick it up.

| File | Purpose |
|------|---------|
| `theme.css` | Required. Loaded after `screen.css`; override its CSS variables or restyle anything |
| `theme.js` | Optional. Loaded with `defer` on every page |
| `views/*.html` | Optional. Replaces the base template with the same file name in `internal/templates/views/` |
| `favicon.png` | Optional. Site icon and Apple touch icon (180x180), used unless `favicon` is configured |
| anything else | Images, fonts, etc., served alongside |

Useful hooks in the base markup:

- The `masthead` partial (`views/masthead.html`) can be overridden on its own.
- Feed items carry `data-item-type` (`link`, `quote`, `image`) and links carry
  `data-user`. Scope feed styling to `.item[data-item-type]` so other `.item`
  boxes, like the stats table, aren't affected.
- Each item has a hidden `<time class="item-time">` (`15:04`) you can show.
- The header's styles are inline in `<body>`, after `theme.css`, so header
  rules need a `:root` prefix to win.

When overriding item templates, keep the `.item[data-url]`, `.og-preview`,
`.link-header .link`, `.date-section` and `.date-widget` hooks if you want link
previews and date grouping to keep working. See the bundled themes for
examples: `corkboard` is CSS only, `broadsheet` overrides the masthead, and
`irc` overrides the masthead and adds `theme.js`.

### Daily Kitten

By default tumble posts a cat from [cataas.com](https://cataas.com) every
day at 10 AM Central (and on startup, if it's later than that and today's
kitten is missing). To turn it off:

```yaml
daily_kitten: false
```

When disabled, nothing is fetched and `PUT /api/v1/kittens/daily` returns
`409`. Past kittens stay in the feed and can still be read or deleted.

### RSS Feed Settings

These optional fields customize the RSS feed metadata. If omitted, the feed title and description default to "tumble", and the other fields are omitted from the feed entirely.

```yaml
site_title: "My Tumble Site"
site_description: "A collection of interesting links and quotes"
managing_editor: "admin@example.com (Your Name)"
webmaster: "admin@example.com (Your Name)"
copyright: "http://creativecommons.org/licenses/by-sa/4.0"
```

### ActivityPub Federation

Tumble can publish new links, quotes, and images to Mastodon-like clients that follow the site. This is publish-only: the site can be followed, but doesn't follow, like, or reply to anything. All posts are federated under a single site-wide actor (there's no per-user identity).

**To enable:**

1. Set `baseurl` (or `TUMBLE_BASEURL`) to the site's real, public **HTTPS** URL. Remote servers fetch the actor and verify signatures against this URL, and most implementations (including Mastodon) refuse to federate with non-HTTPS/non-public hosts, so this won't work against `localhost` without a tunnel (e.g. ngrok/cloudflared).
2. Set `activitypub.enabled: true` (or `TUMBLE_ACTIVITYPUB_ENABLED=true`):

   ```yaml
   activitypub:
     enabled: true
     actor_name: "tumble"
     avatar_url: "https://github.com/websages.png"
   ```
3. Restart the app. On first request to the actor endpoint, an RSA keypair is generated and persisted automatically — no manual key setup needed.
4. The actor is now discoverable as `@tumble@your.domain.com` (substituting `actor_name` and the host from `baseurl`) — search for that handle from any Mastodon-like client to follow it. New links, quotes, and the daily kitten are federated to followers automatically going forward; nothing needs to be republished manually.

`site_title`/`site_name` and `site_description` are reused as the actor's display name and summary, and `avatar_url` (if set) becomes the actor's `icon`.

### Environment Variables

Any configuration value can be set with a `TUMBLE_`-prefixed environment
variable (use underscores for nested keys, e.g. `TUMBLE_LOGGING_LEVEL`).
Environment variables take precedence over the config file, and the
application can run **entirely from environment variables with no config file
present** — useful for container platforms.

**Server**

| Variable | Description | Default |
|----------|-------------|---------|
| `TUMBLE_PORT` | Listen port | `8080` |
| `TUMBLE_BASEURL` | Public base URL (used for RSS links and click tracking) | |
| `TUMBLE_MODE` | `development` or `production` | `production` |
| `TUMBLE_EMBED_ASSETS` | Load assets from binary (`true`) or filesystem (`false`) | `true` |
| `TUMBLE_REQUEST_TIMEOUT` | Outbound HTTP request timeout | `2s` |

**Database**

| Variable | Description | Default |
|----------|-------------|---------|
| `TUMBLE_DRIVER` | Database driver (`sqlite`, `mysql`) | `mysql` |
| `TUMBLE_DATABASE` | Database name (MySQL) or file path (SQLite) | |
| `TUMBLE_HOST` | MySQL host | |
| `TUMBLE_USERNAME` | MySQL username | |
| `TUMBLE_PASSWORD` | MySQL password | |

**Logging & caching**

| Variable | Description | Default |
|----------|-------------|---------|
| `TUMBLE_LOGGING_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `TUMBLE_LOGGING_OUTPUT` | `stdout`, `stderr`, or a file path | `stdout` |
| `TUMBLE_CACHING_ENABLED` | Cache link previews in the database | `true` |

**Secrets**

| Variable | Description | Default |
|----------|-------------|---------|
| `TUMBLE_ADMIN_SECRET` | Secret for admin API operations | |
| `TUMBLE_CLICK_SIGNING_KEY` | Secret for signed click tracking | |

**Site & RSS feed metadata**

| Variable | Description | Default |
|----------|-------------|---------|
| `TUMBLE_SITE_NAME` | Wordmark in masthead and page titles | `tumblefish` |
| `TUMBLE_THEME` | Visual theme (`broadsheet`, `irc`, `corkboard`) | default look |
| `TUMBLE_FAVICON` | Site icon and Apple touch icon (URL or path) | theme icon, else built-in |
| `TUMBLE_DAILY_KITTEN` | Post a daily cataas.com kitten at 10 AM Central | `true` |
| `TUMBLE_SITE_TITLE` | RSS feed title | `tumble` |
| `TUMBLE_SITE_DESCRIPTION` | RSS feed description | same as title |
| `TUMBLE_MANAGING_EDITOR` | RSS managingEditor field | omitted |
| `TUMBLE_WEBMASTER` | RSS webMaster field | omitted |
| `TUMBLE_COPYRIGHT` | RSS copyright field | omitted |

**ActivityPub**

| Variable | Description | Default |
|----------|-------------|---------|
| `TUMBLE_ACTIVITYPUB_ENABLED` | Enable ActivityPub federation | `false` |
| `TUMBLE_ACTIVITYPUB_ACTOR_NAME` | Site actor's `preferredUsername` | `tumble` |
| `TUMBLE_ACTIVITYPUB_AVATAR_URL` | Site actor's `icon` (avatar) image URL | `https://github.com/websages.png` |

### Environment Modes (`TUMBLE_MODE`)

- **development**: Text-format logging, debug level, full SQL query logging, detailed error messages in the browser.
- **production** (default): JSON-format logging, info level, error-only SQL logging, generic error messages to users.

### Asset Embedding (`TUMBLE_EMBED_ASSETS`)

Controls whether templates and static assets are loaded from the compiled binary or the filesystem.

- **true** (default): Assets are loaded from the binary. Only the binary and a config file are needed for deployment.
- **false**: Assets are loaded from `internal/templates/views/` and `internal/assets/`. Templates are re-parsed on every request, enabling hot-reload during development.

This setting is independent of `TUMBLE_MODE`.

## Database Support

Tumble supports both **MySQL** and **SQLite** databases.

- **SQLite**: Recommended for development, testing, and small deployments. No separate database server required.
- **MySQL**: Recommended for production deployments with higher traffic.

### Migrations

Tumble uses **GORM AutoMigrate** to manage database schemas. Migrations are automatically applied on application startup based on the Go structs in `internal/data/store.go`.

### MySQL Setup

To set up a MySQL database manually:

```bash
mysql -u root -p < sql/user_and_database_setup.mysql
```

Then configure `conf/config.yaml` with your MySQL credentials.

## Development

### Makefile Targets

| Command | Description |
|---------|-------------|
| `make build` | Build the binary |
| `make restart` | Build and restart the application |
| `make kill` | Stop the running application |
| `make test` | Run unit tests |
| `make test-api` | Run API integration tests |
| `make test-db` | Create a fresh test database with fixtures |
| `make run-test` | Run the application with the test database |
| `make backup` | Backup the current SQLite database |
| `make restore` | Restore the SQLite database from backup |
| `make fmt` | Run `go fmt` and clean up whitespace |
| `make build-linux` | Cross-compile for Linux amd64 |
| `make deploy` | Deploy to Fly.io (see below) |
| `make help` | Show all available targets |

### Testing

- `make test-api` runs integration tests against a test instance.
- `make test-db` creates a fresh SQLite test database with sample fixtures.
- `make run-test` starts the application using `conf/config-test.yaml`.

To test against MySQL, edit `conf/config-test-mysql.yaml` and run:

```bash
bin/tumble conf/config-test-mysql.yaml
```

### Logging

Set `TUMBLE_MODE=development` or `TUMBLE_LOGGING_LEVEL=debug` for verbose output including SQL queries. Logs are written to `tumble.log` by default.

## Deployment

See the systemd unit files in the `contrib/` directory for production deployment examples.

The recommended approach:

1. Build the binary with `make build` (or `make build-linux` for Linux targets).
2. Create a `config.yaml` with your production settings.
3. Run the binary: `bin/tumble config.yaml`

With `embed_assets: true` (the default), only the binary and config file are needed on the server.

The production systemd units in `contrib/` use Flox to manage the MySQL service (`tumble-db.service` runs `flox activate --start-services`). See `contrib/README` for details.

### Fly.io

Tumble ships with a `fly.toml` and `Dockerfile` for deploying to
[Fly.io](https://fly.io). The image bundles the tumble binary and the
[Litestream](https://litestream.io) binary: SQLite lives on a persistent
volume and is continuously replicated to object storage
([Tigris](https://fly.io/docs/tigris/)) for disaster recovery. All non-secret
configuration is supplied through `TUMBLE_*` environment variables (the
`[env]` block of `fly.toml`); secrets are set separately.

**One-time setup:**

```bash
fly auth login
fly apps create tumble2                              # pick your own app name
fly volumes create tumble_data --region iad --size 1 # persistent SQLite storage
fly storage create                                   # Tigris bucket; sets BUCKET_NAME + AWS_* secrets

# App secrets (never stored in fly.toml)
fly secrets set TUMBLE_ADMIN_SECRET="$(openssl rand -hex 32)"
fly secrets set TUMBLE_CLICK_SIGNING_KEY="$(openssl rand -hex 32)"  # optional
```

**Deploy** (stamps the current git commit into the version banner):

```bash
make deploy
```

**Custom domain:**

```bash
fly certs create tumble2.wcyd.org   # then add the DNS record Fly prints
```

Notes:

- Because SQLite is backed by a single volume, run exactly one machine — do
  not scale the count above 1.
- Litestream replicates continuously; on a fresh machine the entrypoint
  restores the database from the bucket before starting tumble. If no bucket
  is configured (`BUCKET_NAME` unset), the app runs without replication.
- Litestream's replica settings live in `conf/litestream.yml`.

## API

Full API documentation is available at `/api/docs` on a running instance, generated from `internal/assets/openapi.json`.

### Link Submission

Submit links via `POST /link/`. Duplicate detection is automatic:

- **JSON API** (`Accept: application/json` or `source=api`):
  - **201 Created**: New link
  - **208 Already Reported**: Duplicate, includes previous submission details
- **IRC Source** (`source=irc`): Returns ID with duplicate marker if applicable
- **HTML**: Shows duplicate notification with original poster and timestamp

Use `POST /api/v1/links` for programmatic link submission.

### Link and Quote Deletion

Delete links or quotes via `DELETE /link/{id}` or `DELETE /quote/{id}`. Requires authentication:

```bash
# Using header (recommended)
curl -X DELETE -H "X-Admin-Secret: your-secret" https://your-server/link/123

# Using query parameter
curl -X DELETE "https://your-server/link/123?secret=your-secret"
```

Configure `admin_secret` in your config file or via `TUMBLE_ADMIN_SECRET`. Without it, deletion falls back to localhost-only access.

### Click Signature Tracking

Optional HMAC-signed URLs for verified click tracking. When `click_signing_key` is configured, links render as `/link/123?sig=abc123...` and the server validates signatures to distinguish real clicks from bots.

### Dead Link Detection

Tumble automatically detects dead links (4xx/5xx responses) and checks the [Wayback Machine](https://archive.org) for archived snapshots. A background job runs on startup and daily. Archive.org requests are rate-limited to 5 req/s.

### Caching

Link previews are cached in the database. Disable with `caching.enabled: false` in config. Invalidate manually via `GET /api/caching/invalidate?url=<encoded_url>`.

## Project Structure

```
cmd/tumble/          Entry point
internal/
  assets/            Static assets and OpenAPI spec
  config/            Configuration loading
  data/              Database models and queries
  handler/           HTTP handlers
  templates/         HTML templates
  service/           Business logic
  scheduler/         Background jobs
  archive/           Archive.org integration
conf/                Configuration files
contrib/             Systemd unit files
sql/                 SQL setup scripts
tests/               Integration tests and fixtures
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
