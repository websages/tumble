package tumble::DB::MySQL;

use strict;
use warnings;
use base 'tumble::DB::Base';

sub _connect {
    my $self = shift;
    my $config = $self->{config};

    my $dsn = "dbi:mysql:$config->{database}";
    $dsn .= ";host=$config->{host}" if $config->{host};

    # Log connection attempt (sanitize password for logging)
    my $host_info = $config->{host} || 'localhost';
    my $db_name = $config->{database} || 'unknown';
    my $username = $config->{username} || 'unknown';
    
    warn "[MySQL] Attempting to connect to database '$db_name' on host '$host_info' as user '$username'\n";

    eval {
        $self->{dbi} = DBI->connect(
            $dsn,
            $config->{username},
            $config->{password},
            { RaiseError => 1, AutoCommit => 1, PrintError => 0 }
        );
    };
    
    if ($@) {
        my $error = $@;
        warn "[MySQL] Connection FAILED: $error\n";
        warn "[MySQL] Diagnostics:\n";
        warn "[MySQL]   - DSN: $dsn\n";
        warn "[MySQL]   - Username: $username\n";
        warn "[MySQL]   - Host: $host_info\n";
        warn "[MySQL]   - Database: $db_name\n";
        warn "[MySQL] Troubleshooting suggestions:\n";
        warn "[MySQL]   1. Verify MySQL server is running: systemctl status mysql (or mysqld)\n";
        warn "[MySQL]   2. Check credentials in config.yaml are correct\n";
        warn "[MySQL]   3. Verify user has permissions: GRANT ALL ON $db_name.* TO '$username'\@'$host_info'\n";
        warn "[MySQL]   4. Check MySQL is listening on $host_info (check bind-address in my.cnf)\n";
        warn "[MySQL]   5. Verify database '$db_name' exists: SHOW DATABASES;\n";
        die "MySQL connection failed: $error";
    }
    
    if (!$self->{dbi}) {
        warn "[MySQL] Connection FAILED: $DBI::errstr\n";
        warn "[MySQL] Diagnostics:\n";
        warn "[MySQL]   - DSN: $dsn\n";
        warn "[MySQL]   - Error: $DBI::errstr\n";
        die "Can't connect to MySQL: $DBI::errstr";
    }
    
    # Log successful connection with database info
    warn "[MySQL] Connection SUCCESSFUL\n";
    
    # Get and log basic database statistics
    eval {
        my $tables = $self->{dbi}->selectall_arrayref("SHOW TABLES");
        my $table_count = scalar @$tables;
        warn "[MySQL] Database contains $table_count table(s)\n";
        
        if ($table_count == 0) {
            warn "[MySQL] WARNING: Database '$db_name' appears to be empty (no tables found)\n";
            warn "[MySQL]   - You may need to run the database setup script\n";
            warn "[MySQL]   - Check if schema files need to be imported\n";
        } else {
            # Log table names for diagnostics
            my @table_names = map { $_->[0] } @$tables;
            warn "[MySQL] Tables: " . join(', ', @table_names) . "\n";
            
            # Check for expected tables
            my %tables_hash = map { $_ => 1 } @table_names;
            my @expected = qw(ircLink image quote);
            my @missing;
            foreach my $expected_table (@expected) {
                push @missing, $expected_table unless $tables_hash{$expected_table};
            }
            
            if (@missing) {
                warn "[MySQL] WARNING: Missing expected tables: " . join(', ', @missing) . "\n";
                warn "[MySQL]   - Database may not be fully initialized\n";
            }
        }
    };
    if ($@) {
        warn "[MySQL] Could not retrieve database statistics: $@\n";
    }
}

sub date_interval_sql {
    my ($self, $days, $op) = @_;
    # Defaults: $days is number of days, $op is '<=' or '>='
    # Current code uses: DATE_SUB(CURDATE(), INTERVAL $days DAY) <= timestamp
    
    # We'll return the expression 'timestamp' should be compared against, 
    # or the full condition string?
    # The existing code constructs: "DATE_SUB(CURDATE(), INTERVAL $i * 6 DAY) <= timestamp"
    # To make it cleaner, let's allow generating the LHS.
    
    return "DATE_SUB(CURDATE(), INTERVAL $days DAY)";
}

sub fulltext_search_sql {
    my ($self, $cols, $query) = @_;
    # MATCH (cols) AGAINST ('query')
    # We need to trust the caller to sanitize or we quote specific parts?
    # Simpler: just return string structure
    return "MATCH ($cols) AGAINST (" . $self->{dbi}->quote($query) . ")";
}

1;
