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

## Bugs

    * fix user-agent being hardy for link verification
    * Should warn if unable to talk to databse or database is empty
    * abstract quantity of items to be in 'hot shit' category
    * Fix odd encoding bugs for web site titles
    * Probably lots of others, but it has been in production for 10 years.
