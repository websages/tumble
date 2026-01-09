package tumble::DB::Base;

use strict;
use warnings;
use DBI;

sub new {
    my $class = shift;
    my $self = bless {}, $class;
    
    my %args = @_;
    $self->{config} = $args{config_data};
    
    $self->_connect();
    
    return $self;
}

sub _connect {
    die "Subclass must implement _connect";
}

sub fetch {
    my $self = shift;
    my %args = @_;

    $args{key} ||= $args{source} . 'ID';

    my $where_clause = '';
    if ( $args{filter} ) {
        $where_clause = "WHERE $args{filter}";
    }

    my $order_clause = "ORDER BY $args{key}";
    if ( $args{order} ) {
        $order_clause .= " $args{order}";
    }
    
    my $limit_clause = '';
    if ( $args{limit} ) {
        $limit_clause = "LIMIT $args{limit}";
    }

    my $sql = "SELECT * FROM $args{source} $where_clause $order_clause $limit_clause";

    return $self->{dbi}->selectall_hashref(
        $sql,
        $args{key}
    );
}

sub post {
    my $self = shift;
    my %args = @_;

    $args{authorName} ||= 'anonymous';
    
    my $table = delete $args{destination};
    my @columns = sort keys %args;
    
    my $cols_str = join( ', ', map { $self->quote_identifier($_) } @columns );
    my $vals_str = join( ', ', map { $self->{dbi}->quote($args{$_}) } @columns );
    
    my $sql = "INSERT INTO " . $self->quote_identifier($table) . " ( $cols_str ) VALUES ( $vals_str )";

    $self->{dbi}->do($sql);
}

sub disconnect {
    my $self = shift;
    $self->{dbi}->disconnect() if $self->{dbi};
}

# Helper to quote identifiers (table/column names) - overridden by drivers if needed
sub quote_identifier {
    my ($self, $ident) = @_;
    return "`$ident`"; # Default to backticks (MySQL style)
}

sub date_interval_sql {
    die "Subclass must implement date_interval_sql";
}

sub fulltext_search_sql {
    die "Subclass must implement fulltext_search_sql";
}

# Pass-through DBI
sub prepare { return shift->{dbi}->prepare(@_); }
sub selectrow_array { return shift->{dbi}->selectrow_array(@_); }

1;
