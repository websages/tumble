# Tumble

The canonical installation of tumble is found at [http://tumble.wcyd.org](http://tumble.wcyd.org).

## History

Tumble was a "Wouldn't it be cool?" project handed to [Scott Schnedier](https://github.com/sschneid) back in 2004. The idea was to create a website similar to a tumbleblog. Obviously, eventually tumblr ccame along and the rest was history.

## Deployment

The easiet way to deploy is type is to clone and type `make rpm` on an EL6 system. Things should justwork after that.

If you are not on EL, things should still work. Just `make install` or package it yourself.

## Database Support

Tumble now supports both **MySQL** and **SQLite** databases. Choose the one that fits your needs:

- **SQLite**: Recommended for development, testing, and small deployments. No separate database server required.
- **MySQL**: Recommended for production deployments with higher traffic.

## Development Workflows

### Database Migrations

Tumble uses [golang-migrate](https://github.com/golang-migrate/migrate) to manage database schemas. Migrations are automatically applied on application startup.

- **Location**: Migration files are stored in `sql/mysql/` and `sql/sqlite/`.
- **Versioning**: Files are named sequentially (e.g., `000001_init.up.sql`).
- **Adding Migrations**: To change the schema, create a new numbered `.up.sql` file in the appropriate directory.

### Testing Infrastructure

A robust test infrastructure is available for rapid development:

- **Run Tests**: `make test-api` runs the integration tests.
- **Test Database**: `make test-db` creates a fresh, disposable SQLite database (`tumble-test.sqlite`), runs migrations, and loads sample fixtures.
- **Run with Test DB**: `make run-test` starts the application using the test database.
- **Backup/Restore**: `make backup` and `make restore` allow you to snapshot your current development database.

To customize the test environment, you can edit `conf/config-test.yaml`.

## Quick Setup

### 1. Configure Database

Create or edit `config.yaml` in the htdocs directory:

**For SQLite (easiest):**

```yaml
driver: sqlite
database_file: tumble.db
baseurl: your.domain.com
```

**For MySQL:**

```yaml
driver: mysql
database: tumble
username: tumble
password: your_secure_password
host: localhost
baseurl: your.domain.com
```

### 2. Initialize Database

Run the setup script (works for both MySQL and SQLite):

```bash
perl scripts/setup_database.pl
```

That's it! The script will automatically detect your database type and create the necessary tables.

### 3. Configure Web Server

1. Update `/etc/httpd/conf.d/tumble.conf` with your server configuration
2. Disable SELinux or set proper context
3. Start httpd:

```bash
chkconfig httpd on
service httpd start
```

## Detailed Setup (MySQL)

If you prefer manual MySQL setup:

```bash
# Install MySQL
yum install mysql-server
service mysqld start
chkconfig mysqld on

# Create database and user
mysql -u root -p < sql/sql_setup

# Run schema
mysql -u tumble -p tumble < sql/schema.mysql
```

## Migration from MySQL to SQLite

1. Export your MySQL data:

   ```bash
   mysqldump -u tumble -p tumble > tumble_backup.sql
   ```

2. Update `config.yaml` to use SQLite

3. Run setup script:

   ```bash
   perl scripts/setup_database.pl
   ```

4. Import data (requires conversion from MySQL to SQLite format)

See [docs/database_setup.md](docs/database_setup.md) for detailed instructions.

## Debugging and Logging

Tumble provides comprehensive database logging to help troubleshoot connection issues and diagnose problems.

### Automatic Database Diagnostics

When the application starts, it automatically:

- **Logs connection attempts** with database details (host, database name, file paths)
- **Verifies database health** by checking for expected tables (`ircLink`, `image`, `quote`)
- **Reports table statistics** including row counts for each table
- **Warns about issues** such as:
  - Missing database files (SQLite)
  - Empty databases (no tables)
  - Missing expected tables
  - Empty tables (no data)
  - File permission problems (SQLite)
  - Connection failures with troubleshooting suggestions

### Debug Mode

For verbose logging of all database operations, enable debug mode:

```bash
export TUMBLE_DEBUG=1
```

With debug mode enabled, you'll see:

- All SQL queries being executed
- Row counts for each query result
- Detailed query execution information

**Example:**

```bash
# Enable debug mode
export TUMBLE_DEBUG=1

# Start your web server or run the application
perl -I htdocs/lib htdocs/index.cgi
```

### Log Output Examples

**Successful MySQL connection:**

```
[MySQL] Attempting to connect to database 'tumble' on host 'localhost' as user 'tumble'
[MySQL] Connection SUCCESSFUL
[MySQL] Database contains 3 table(s)
[MySQL] Tables: ircLink, image, quote
```

**SQLite with missing database file:**

```
[SQLite] Attempting to connect to database file: tumble.db
[SQLite] Database file DOES NOT EXIST
[SQLite]   - SQLite will create a new empty database file
[SQLite]   - You will need to run the database setup script to create tables
[SQLite] Connection SUCCESSFUL
[SQLite] WARNING: Database appears to be empty (no tables found)
[SQLite]   - You need to run the database setup script
```

**Connection failure with diagnostics:**

```
[MySQL] Connection FAILED: Access denied for user 'tumble'@'localhost'
[MySQL] Diagnostics:
[MySQL]   - DSN: dbi:mysql:tumble;host=localhost
[MySQL]   - Username: tumble
[MySQL] Troubleshooting suggestions:
[MySQL]   1. Verify MySQL server is running: systemctl status mysql
[MySQL]   2. Check credentials in config.yaml are correct
[MySQL]   3. Verify user has permissions: GRANT ALL ON tumble.* TO 'tumble'@'localhost'
```

## Bugs

    * fix user-agent being hardy for link verification
    * abstract quantity of items to be in 'hot shit' category
    * Fix odd encoding bugs for web site titles
    * Probably lots of others, but it has been in production for 10 years.
