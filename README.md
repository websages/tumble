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

### Environment Variables

Override any config value with the `TUMBLE_` prefix. Use underscores for nested keys.

| Variable | Description | Default |
|----------|-------------|---------|
| `TUMBLE_PORT` | Listen port | `8080` |
| `TUMBLE_DRIVER` | Database driver (`sqlite`, `mysql`) | `mysql` |
| `TUMBLE_DATABASE` | Database name or file path | |
| `TUMBLE_MODE` | `development` or `production` | `production` |
| `TUMBLE_EMBED_ASSETS` | Load assets from binary or filesystem | `true` |
| `TUMBLE_LOGGING_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `TUMBLE_REQUEST_TIMEOUT` | HTTP request timeout | `2s` |
| `TUMBLE_ADMIN_SECRET` | Secret for admin API operations | |
| `TUMBLE_CLICK_SIGNING_KEY` | Secret for signed click tracking | |

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
