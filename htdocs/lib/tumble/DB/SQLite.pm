package tumble::DB::SQLite;

use strict;
use warnings;
use base 'tumble::DB::Base';

sub _connect {
    my $self = shift;
    my $config = $self->{config};

    my $dbfile = $config->{database_file} || 'tumble.db';
    
    # Log connection attempt with file path
    use Cwd qw(abs_path getcwd);
    my $abs_dbfile = abs_path($dbfile) || File::Spec->rel2abs($dbfile);
    my $cwd = getcwd();
    
    warn "[SQLite] Attempting to connect to database file: $dbfile\n";
    warn "[SQLite] Absolute path: $abs_dbfile\n";
    warn "[SQLite] Current working directory: $cwd\n";
    
    # Check file existence and permissions before connecting
    my $file_exists = -e $dbfile;
    if ($file_exists) {
        warn "[SQLite] Database file EXISTS\n";
        
        # Check permissions
        my $readable = -r $dbfile;
        my $writable = -w $dbfile;
        my $file_size = -s $dbfile;
        
        warn "[SQLite] File size: " . ($file_size || 0) . " bytes\n";
        warn "[SQLite] Readable: " . ($readable ? "YES" : "NO") . "\n";
        warn "[SQLite] Writable: " . ($writable ? "YES" : "NO") . "\n";
        
        if (!$readable) {
            warn "[SQLite] WARNING: Database file is not readable\n";
            warn "[SQLite]   - Check file permissions: chmod 644 $dbfile\n";
        }
        if (!$writable) {
            warn "[SQLite] WARNING: Database file is not writable\n";
            warn "[SQLite]   - Check file permissions: chmod 644 $dbfile\n";
        }
        
        if ($file_size == 0) {
            warn "[SQLite] WARNING: Database file is empty (0 bytes)\n";
            warn "[SQLite]   - This may be a newly created file\n";
            warn "[SQLite]   - You may need to run the database setup script\n";
        }
    } else {
        warn "[SQLite] Database file DOES NOT EXIST\n";
        warn "[SQLite]   - SQLite will create a new empty database file\n";
        warn "[SQLite]   - You will need to run the database setup script to create tables\n";
        
        # Check if directory is writable
        my $dir = $dbfile;
        $dir =~ s/[^\/]+$//;
        $dir = '.' if $dir eq '';
        
        if (!-w $dir) {
            warn "[SQLite] ERROR: Directory '$dir' is not writable\n";
            warn "[SQLite]   - Cannot create database file\n";
            warn "[SQLite]   - Check directory permissions\n";
        }
    }
    
    eval {
        $self->{dbi} = DBI->connect(
            "dbi:SQLite:dbname=$dbfile",
            "",
            "",
            { RaiseError => 1, AutoCommit => 1, PrintError => 0 }
        );
    };
    
    if ($@) {
        my $error = $@;
        warn "[SQLite] Connection FAILED: $error\n";
        warn "[SQLite] Diagnostics:\n";
        warn "[SQLite]   - Database file: $dbfile\n";
        warn "[SQLite]   - Absolute path: $abs_dbfile\n";
        warn "[SQLite]   - File exists: " . ($file_exists ? "YES" : "NO") . "\n";
        warn "[SQLite] Troubleshooting suggestions:\n";
        warn "[SQLite]   1. Check file permissions: ls -la $dbfile\n";
        warn "[SQLite]   2. Verify directory is writable\n";
        warn "[SQLite]   3. Check disk space: df -h\n";
        warn "[SQLite]   4. Ensure DBD::SQLite module is installed: cpan DBD::SQLite\n";
        die "SQLite connection failed: $error";
    }
    
    if (!$self->{dbi}) {
        warn "[SQLite] Connection FAILED: $DBI::errstr\n";
        die "Can't connect to SQLite: $DBI::errstr";
    }
    
    # Log successful connection
    warn "[SQLite] Connection SUCCESSFUL\n";
    
    # Get and log basic database statistics
    eval {
        my $tables = $self->{dbi}->selectall_arrayref(
            "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
        );
        my $table_count = scalar @$tables;
        warn "[SQLite] Database contains $table_count table(s)\n";
        
        if ($table_count == 0) {
            warn "[SQLite] WARNING: Database appears to be empty (no tables found)\n";
            warn "[SQLite]   - You need to run the database setup script\n";
            warn "[SQLite]   - Check for schema initialization files\n";
        } else {
            # Log table names and row counts for diagnostics
            my @table_names = map { $_->[0] } @$tables;
            warn "[SQLite] Tables: " . join(', ', @table_names) . "\n";
            
            # Check for expected tables and get row counts
            my %tables_hash = map { $_ => 1 } @table_names;
            my @expected = qw(ircLink image quote);
            my @missing;
            foreach my $expected_table (@expected) {
                if ($tables_hash{$expected_table}) {
                    my ($count) = $self->{dbi}->selectrow_array(
                        "SELECT COUNT(*) FROM \"$expected_table\""
                    );
                    warn "[SQLite]   - $expected_table: $count row(s)\n";
                    
                    if ($count == 0) {
                        warn "[SQLite] WARNING: Table '$expected_table' is empty\n";
                    }
                } else {
                    push @missing, $expected_table;
                }
            }
            
            if (@missing) {
                warn "[SQLite] WARNING: Missing expected tables: " . join(', ', @missing) . "\n";
                warn "[SQLite]   - Database may not be fully initialized\n";
            }
        }
    };
    if ($@) {
        warn "[SQLite] Could not retrieve database statistics: $@\n";
    }
}

sub quote_identifier {
    my ($self, $ident) = @_;
    return qq("$ident"); # Double quotes for standard SQL / SQLite
}

sub date_interval_sql {
    my ($self, $days) = @_;
    # SQLite: date('now', '-$days days')
    return "date('now', '-$days days')";
}

sub fulltext_search_sql {
    my ($self, $cols, $query) = @_;
    # SQLite basic substitute: OR of LIKEs?
    # Or assuming one col for now, or concat?
    # Simple fallback: LIKE
    # Since existing searches 'title,url', we might match either.
    
    my @fields = split( /,/, $cols );
    my @parts = ();
    my $q = $self->{dbi}->quote("%$query%");
    
    foreach my $f (@fields) {
        push @parts, "$f LIKE $q";
    }
    
    return "(" . join( " OR ", @parts ) . ")";
}

1;
