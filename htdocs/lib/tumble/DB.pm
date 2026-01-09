package tumble::DB;

use strict;
use warnings;
use YAML qw( LoadFile );
use tumble::DB::MySQL;
use tumble::DB::SQLite;

sub new {
    my $class = shift;
    my %args = @_;

    my $config_file = $args{config} || 'config.yaml';
    my $config = LoadFile($config_file);

    my $driver = $config->{driver} || 'mysql'; # Default to MySQL

    if ( lc($driver) eq 'sqlite' ) {
        return tumble::DB::SQLite->new( %args, config_data => $config );
    }
    elsif ( lc($driver) eq 'mysql' ) {
        return tumble::DB::MySQL->new( %args, config_data => $config );
    }
    else {
        die "Unknown database driver: $driver";
    }
}

1;
