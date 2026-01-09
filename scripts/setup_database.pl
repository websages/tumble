#!/usr/bin/env perl

=head1 NAME

setup_database.pl - Database-agnostic setup script for Tumble

=head1 SYNOPSIS

    perl scripts/setup_database.pl [--config=path/to/config.yaml]

=head1 DESCRIPTION

This script initializes the Tumble database based on the driver specified
in config.yaml. It supports both MySQL and SQLite databases.

=cut

use strict;
use warnings;
use FindBin;
use lib "$FindBin::Bin/../htdocs/lib";
use YAML qw( LoadFile );
use DBI;
use Getopt::Long;

my $config_file = 'config.yaml';
GetOptions(
    'config=s' => \$config_file,
) or die "Usage: $0 [--config=path/to/config.yaml]\n";

# Load configuration
my $config = LoadFile($config_file);
my $driver = lc($config->{driver} || 'mysql');

print "=" x 60 . "\n";
print "Tumble Database Setup\n";
print "=" x 60 . "\n";
print "Driver: $driver\n";
print "Config: $config_file\n";
print "=" x 60 . "\n\n";

if ($driver eq 'mysql') {
    setup_mysql($config);
} elsif ($driver eq 'sqlite') {
    setup_sqlite($config);
} else {
    die "Unknown database driver: $driver\n";
}

print "\n" . "=" x 60 . "\n";
print "Database setup completed successfully!\n";
print "=" x 60 . "\n";

exit 0;

#
# MySQL Setup
#
sub setup_mysql {
    my ($config) = @_;
    
    print "Setting up MySQL database...\n\n";
    
    # Check required config
    die "Missing 'database' in config\n" unless $config->{database};
    die "Missing 'username' in config\n" unless $config->{username};
    
    my $database = $config->{database};
    my $username = $config->{username};
    my $password = $config->{password} || '';
    my $host = $config->{host} || 'localhost';
    
    # Connect to MySQL server (without database)
    print "Connecting to MySQL server at $host...\n";
    my $dsn = "dbi:mysql:host=$host";
    my $dbh = DBI->connect($dsn, $username, $password, {
        RaiseError => 0,
        PrintError => 0,
    });
    
    if (!$dbh) {
        print "Warning: Could not connect to MySQL server.\n";
        print "Error: $DBI::errstr\n";
        print "Attempting to connect directly to database '$database'...\n\n";
        
        # Try connecting directly to the database (assume it exists)
        $dsn = "dbi:mysql:database=$database;host=$host";
        $dbh = DBI->connect($dsn, $username, $password, {
            RaiseError => 1,
            PrintError => 1,
        }) or die "Could not connect to database: $DBI::errstr\n";
    } else {
        # Create database if it doesn't exist
        print "Creating database '$database' if it doesn't exist...\n";
        $dbh->do("CREATE DATABASE IF NOT EXISTS `$database`") 
            or die "Could not create database: " . $dbh->errstr . "\n";
        
        # Switch to the database
        $dbh->do("USE `$database`")
            or die "Could not use database: " . $dbh->errstr . "\n";
    }
    
    print "Connected to database '$database'\n\n";
    
    # Read and execute schema file
    my $schema_file = "$FindBin::Bin/../sql/schema.mysql";
    print "Executing schema from: $schema_file\n";
    
    open my $fh, '<', $schema_file or die "Could not open $schema_file: $!\n";
    
    # Read and execute SQL statements
    my $current_stmt = '';
    my $count = 0;
    
    while (my $line = <$fh>) {
        # Skip comment-only lines
        next if $line =~ /^\s*--/;
        
        # Remove inline comments
        $line =~ s/--.*$//;
        
        # Accumulate the statement
        $current_stmt .= $line;
        
        # If we hit a semicolon, execute the statement
        if ($line =~ /;\s*$/) {
            $current_stmt =~ s/^\s+|\s+$//g;  # Trim
            
            if ($current_stmt && $current_stmt !~ /^\s*$/) {
                eval {
                    $dbh->do($current_stmt);
                    $count++;
                    print "  ✓ Executed statement $count\n" if $ENV{VERBOSE};
                };
                if ($@) {
                    warn "Warning executing statement: $@\n";
                    warn "Statement was: $current_stmt\n" if $ENV{VERBOSE};
                }
            }
            
            $current_stmt = '';
        }
    }
    
    close $fh;
    
    print "Executed $count SQL statements\n";
    
    $dbh->disconnect();
    print "\nMySQL setup complete!\n";
}

#
# SQLite Setup
#
sub setup_sqlite {
    my ($config) = @_;
    
    print "Setting up SQLite database...\n\n";
    
    my $dbfile = $config->{database_file} || 'tumble.db';
    
    print "Database file: $dbfile\n";
    
    if (-e $dbfile) {
        print "Warning: Database file already exists. Tables will be created if they don't exist.\n\n";
    }
    
    # Connect to SQLite
    print "Connecting to SQLite database...\n";
    my $dbh = DBI->connect("dbi:SQLite:dbname=$dbfile", "", "", {
        RaiseError => 1,
        PrintError => 1,
    }) or die "Could not connect to SQLite: $DBI::errstr\n";
    
    print "Connected successfully\n\n";
    
    # Read and execute schema file
    my $schema_file = "$FindBin::Bin/../sql/schema.sqlite";
    print "Executing schema from: $schema_file\n";
    
    open my $fh, '<', $schema_file or die "Could not open $schema_file: $!\n";
    
    # Read and execute SQL statements
    my $current_stmt = '';
    my $count = 0;
    
    while (my $line = <$fh>) {
        # Skip comment-only lines
        next if $line =~ /^\s*--/;
        
        # Remove inline comments
        $line =~ s/--.*$//;
        
        # Accumulate the statement
        $current_stmt .= $line;
        
        # If we hit a semicolon, execute the statement
        if ($line =~ /;\s*$/) {
            $current_stmt =~ s/^\s+|\s+$//g;  # Trim
            
            if ($current_stmt && $current_stmt !~ /^\s*$/) {
                eval {
                    $dbh->do($current_stmt);
                    $count++;
                    print "  ✓ Executed statement $count\n" if $ENV{VERBOSE};
                };
                if ($@) {
                    warn "Warning executing statement: $@\n";
                    warn "Statement was: $current_stmt\n" if $ENV{VERBOSE};
                }
            }
            
            $current_stmt = '';
        }
    }
    
    close $fh;
    
    print "Executed $count SQL statements\n";
    
    $dbh->disconnect();
    print "\nSQLite setup complete!\n";
    print "Database location: $dbfile\n";
}

__END__

=head1 CONFIGURATION

The script reads config.yaml to determine which database driver to use.

For MySQL:
    driver: mysql
    database: tumble
    username: tumble
    password: your_password
    host: localhost

For SQLite:
    driver: sqlite
    database_file: tumble.db

=head1 AUTHOR

Tumble Development Team

=cut
