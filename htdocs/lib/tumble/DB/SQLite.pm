package tumble::DB::SQLite;

use strict;
use warnings;
use base 'tumble::DB::Base';

sub _connect {
    my $self = shift;
    my $config = $self->{config};

    my $dbfile = $config->{database_file} || 'tumble.db';
    
    $self->{dbi} = DBI->connect(
        "dbi:SQLite:dbname=$dbfile",
        "",
        "",
        { RaiseError => 1, AutoCommit => 1 }
    ) or die "Can't connect: $DBI::errstr";
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
