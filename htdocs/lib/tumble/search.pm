package tumble::search;

use base 'CGI::Application';

use lsrfsh::MySQL;

use DBI;

use strict;
use warnings;



sub setup {
    my $self = shift;

    $self->run_modes([qw/
        displaySearch
    /]);

    $self->{'cgi'} = $self->query();

    foreach my $param ( $self->{'cgi'}->param() ) {
        my $raw = [ $self->{'cgi'}->param( $param ) ];
        $self->{'arg'}->{$param} = @$raw > 1 ? $raw : $raw->[0];
    }

    $self->{'arg'}->{'dtype'} ||= 'html';

    for ( $self->{'arg'}->{'dtype'} ) {
        /rss|xml/ && do { $self->header_props( -type => 'text/xml' ); };
    }

    $self->{'dbh'} = lsrfsh::MySQL->new( config => 'config.yaml' );

    $self->start_mode( 'displaySearch' );

    return $self;
}

sub displaySearch {
    my $self = shift;

    my $string = 'unicorn';
    return unless $string;

    my $raw = $self->{'dbh'}->fetch(
        source => 'ircLink',
        filter => "MATCH (title,url) AGAINST ('$self->{'arg'}->{'search'}')",
        key    => 'ircLinkID'
    );

    my ( $c, $h );

    if ( keys %{$raw} > 0 ) {
        foreach my $item (
            reverse sort {
                $raw->{$a}->{'clicks'} cmp $raw->{$b}->{'clicks'}
            } keys %{$raw}
        ) {
            my $link  =
                qq{<a href="http://tumble.stahnkage.com/irclink/?} .
                $raw->{$item}->{'ircLinkID'} .
                qq{">} .
                $raw->{$item}->{'title'} .
                qq{</a>};

            print "link: $link\n";

            $c .= $self->wrap(
                wrapper => 'tumble_item_ircLink',
                author  => $raw->{$item}->{'user'},
                content => $link,
                %{$raw->{$item}}
            );
        }
    }
    else {
        $c = $self->wrap(
            wrapper => 'tumble_item_text',
            content => qq{
            <font color="#000">Your search-fu is weak.</font><br /><br />
            Your search for '$self->{'arg'}->{'search'}' did not return any results.  Perhaps the following tips can help aid you on your quest:
            <ul>
                <li>Searches must be done using four or more characters.<br /><br />
                <li>MySQL fulltext-searching is the magic behind this.  Stop blaming scott.<br /><br />
                <li>Try not to be such a fucking idiot.
            </ul>
        }
        );
    }

    my $filter = qq{
        DATE_SUB(
            CURDATE(), INTERVAL 12 DAY
        ) <= timestamp
        AND DATE_SUB(
            CURDATE(), INTERVAL 6 DAY
        ) >= timestamp
        AND clicks > 1
    };

    my $hot = $self->{'dbh'}->fetch(
        source => 'ircLink',
        filter => $filter,
        limit => 5,
        key => 'timestamp'
    );

    map {
        my $co =
            qq{<a href="http://tumble.stahnkage.com/irclink/?} .
            $hot->{$_}->{'ircLinkID'} .
            qq{">} .
            $hot->{$_}->{'title'} .
            qq{</a>};

        $h .= $self->wrap(
            wrapper => 'tumble_item_top5',
            content => $co
        );
    } keys %{$hot};

    return $self->wrap(
        wrapper   => 'index',
        'page-title' => " &gt; $self->{'arg'}->{'search'}",
        hot       => $h,
        container => $c
    );
}



sub wrap {
    my $self = shift;

    my ( $arg );
    %{$arg} = @_;

    my $template = $self->load_tmpl(
        $arg->{'wrapper'} . '.thtml',
        die_on_bad_params => 0
    );

    delete $arg->{'wrapper'};

    map {
        chomp( $arg->{$_} ) if ref $arg->{$_};
        $template->param( $_ => $arg->{$_} );
    } keys %{$arg};

    return $template->output();
}



1;
