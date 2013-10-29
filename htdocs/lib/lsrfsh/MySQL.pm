package lsrfsh::MySQL;

# lsrfsh::MySQL

# Copyright 2003-2004 S. Schneider.  This code cannot be redistributed without
# permission from loserfish.org.

# $Id: MySQL.pm,v 1.4 2004/07/23 19:30:10 schnesa Exp $

our @ISA     = qw( lsrfsh );
our $VERSION = 1.00;

use DBI;
use YAML qw( LoadFile );



# Create a new lsrfsh::MySQL object.
sub new {
    my $self = bless {}, shift;

    my ( $arg );
    %{$arg} = @_;

    my ( $config );

    if ( $arg->{'config'} ) {
      $config = LoadFile( $arg->{'config'} );
    }
    else {
      $config = LoadFile( 'config.yaml' );
    }

    map {
        die "You did not specify a $_.\n" unless $config->{$_};
    } qw( database username );

    $config->{'db'}  = 'dbi:mysql:' . $config->{'database'};
    $config->{'db'} .= ';host='     . $config->{'host'} if $config->{'host'};

    # Bind to the mySQL database via DBI.
    $self->{'dbi'} = DBI->connect(
        $config->{'db'},
        $config->{'username'},
        $config->{'password'}
    ) || die "Can't connect: $DBI::errstr\n";

    return $self;
}

# Create an SQL statement and feed it to DBI, returning results in a hash.
sub fetch {
    my $self = shift;

    my ( $arg );
    %{$arg} = @_;

    $arg->{'key'} ||= "$arg->{'source'}ID";

    my ( $sql );

    $sql  = "SELECT * FROM $arg->{'source'} ";
    $sql .= "WHERE $arg->{'filter'} " if $arg->{'filter'};
    $sql .= "ORDER BY $arg->{'key'} ";
    $sql .= "$arg->{'order'} " if $arg->{'order'};
    $sql .= "LIMIT $arg->{'limit'}" if $arg->{'limit'};

    return $self->{'dbi'}->selectall_hashref(
        $sql,                   # The already-built SQL statement
        "$arg->{'key'}"         # Always use "<sourcename>ID" as key
    );
}

# Create an SQL statement and feed it to DBI.
sub post {
    my $self = shift;

    my ( $arg );
    %{$arg} = @_;

    my ( $sql );

    $arg->{'authorName'} ||= 'anonymous';

    $sql .= "INSERT INTO `$arg->{'destination'}` ( ";

    delete $arg->{'destination'};

    map { $sql .= "`$_`, "; } sort keys %{$arg};
    for ( 1..2 ) { chop( $sql ); }
    $sql .= " ) VALUES ( ";
    map {
        $arg->{$_} =~ s/'/\\'/g;
        $sql .= "'$arg->{$_}', ";
    } sort keys %{$arg};
    for ( 1..2 ) { chop( $sql ); }
    $sql .= ");";

    my $sth = $self->{'dbi'}->prepare( $sql );
    $sth->execute();

    return;
}

# Disconnect.
sub disconnect { $self->{'dbi'} && $self->{'dbi'}->disconnect(); }

1;
