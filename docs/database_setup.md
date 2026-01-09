# Database Setup Guide

This guide explains how to set up the Tumble database for both MySQL and SQLite.

## Quick Start

### SQLite Setup (Recommended for Development)

1. Create or edit `config.yaml`:
   ```yaml
   driver: sqlite
   database_file: tumble.db
   baseurl: your.domain.com
   ```

2. Run the setup script:
   ```bash
   perl scripts/setup_database.pl
   ```

3. Done! Your SQLite database is ready at `tumble.db`

### MySQL Setup (Production)

1. Create or edit `config.yaml`:
   ```yaml
   driver: mysql
   database: tumble
   username: tumble
   password: your_secure_password
   host: localhost
   baseurl: your.domain.com
   ```

2. (Optional) Create the MySQL user and database manually:
   ```bash
   mysql -u root -p < sql/sql_setup
   ```
   
   Or let the setup script create the database automatically.

3. Run the setup script:
   ```bash
   perl scripts/setup_database.pl
   ```

4. Done! Your MySQL database is ready.

## Configuration Options

### SQLite Configuration

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `driver` | Yes | `mysql` | Must be set to `sqlite` |
| `database_file` | No | `tumble.db` | Path to SQLite database file |
| `baseurl` | Yes | - | Base URL for the application |

### MySQL Configuration

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `driver` | No | `mysql` | Can be omitted or set to `mysql` |
| `database` | Yes | - | MySQL database name |
| `username` | Yes | - | MySQL username |
| `password` | No | - | MySQL password |
| `host` | No | `localhost` | MySQL server hostname |
| `baseurl` | Yes | - | Base URL for the application |

## Schema Files

The setup script uses database-specific schema files:

- **[sql/schema.mysql](file:///Users/stahnma/development/personal/tumble/sql/schema.mysql)** - MySQL schema with MyISAM engine and FULLTEXT indexes
- **[sql/schema.sqlite](file:///Users/stahnma/development/personal/tumble/sql/schema.sqlite)** - SQLite schema with compatible data types

Both schemas create the same tables:
- `image` - Image posts
- `ircLink` - Link posts with click tracking
- `quote` - Quote posts
- `schema_version` - Migration tracking (for future use)

## Differences Between MySQL and SQLite

### Full-Text Search

- **MySQL**: Uses native `FULLTEXT` indexes on `ircLink.title` and `ircLink.url`
- **SQLite**: Uses `LIKE`-based search (slower but functional)

For better SQLite performance, consider implementing FTS5 virtual tables in the future.

### Date Functions

The `tumble::DB` abstraction layer handles date differences:
- **MySQL**: `DATE_SUB(CURDATE(), INTERVAL N DAY)`
- **SQLite**: `date('now', '-N days')`

### Data Types

- **MySQL**: Uses specific types like `int(16)`, `varchar(255)`, `text`
- **SQLite**: Uses `INTEGER`, `TEXT`, `DATETIME`

## Troubleshooting

### "Could not connect to MySQL server"

If you see this error, the script will attempt to connect directly to the database. Make sure:
1. MySQL server is running
2. The database exists (or the user has CREATE DATABASE privileges)
3. Username and password are correct

### SQLite file permissions

Make sure the directory containing the SQLite database file is writable by the web server user.

### Verbose Output

For debugging, run with verbose output:
```bash
VERBOSE=1 perl scripts/setup_database.pl
```

## Migration from MySQL to SQLite

To migrate from MySQL to SQLite:

1. Export data from MySQL:
   ```bash
   mysqldump -u tumble -p tumble > tumble_backup.sql
   ```

2. Convert the dump to SQLite format (manual process or use a conversion tool)

3. Update `config.yaml` to use SQLite

4. Run the setup script to create the SQLite schema

5. Import the converted data

## Advanced: Manual Setup

If you prefer to run SQL manually:

### MySQL
```bash
mysql -u tumble -p tumble < sql/schema.mysql
```

### SQLite
```bash
sqlite3 tumble.db < sql/schema.sqlite
```

## See Also

- [Database Abstraction Implementation](file:///Users/stahnma/.gemini/antigravity/brain/6a9a5350-de23-4a28-bcc5-704005d13bc4/walkthrough.md) - Technical details of the DB abstraction layer
- [scripts/setup_database.pl](file:///Users/stahnma/development/personal/tumble/scripts/setup_database.pl) - Setup script source code
