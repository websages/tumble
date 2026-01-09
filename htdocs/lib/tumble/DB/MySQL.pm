package tumble::DB::MySQL;

use strict;
use warnings;
use base 'tumble::DB::Base';

sub _connect {
    my $self = shift;
    my $config = $self->{config};

    my $dsn = "dbi:mysql:$config->{database}";
    $dsn .= ";host=$config->{host}" if $config->{host};

    $self->{dbi} = DBI->connect(
        $dsn,
        $config->{username},
        $config->{password},
        { RaiseError => 1, AutoCommit => 1 }
    ) or die "Can't connect: $DBI::errstr";
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
