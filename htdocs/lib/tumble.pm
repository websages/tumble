package tumble;

use base 'CGI::Application';

use lsrfsh::MySQL;

use DBI;
use POSIX qw( strftime );

use YAML qw( LoadFile );

use strict;
use warnings;

my $CONFIG = LoadFile( 'config.yaml' );


sub setup {
    my $self = shift;

    $self->run_modes([qw/
        displayTumble
    /]);

    $self->{'cgi'} = $self->query();

    foreach my $param ( $self->{'cgi'}->param() ) {
        my $raw = [ $self->{'cgi'}->param( $param ) ];
        $self->{'arg'}->{$param} = @$raw > 1 ? $raw : $raw->[0];
    }

    $self->{'arg'}->{'dtype'} ||= 'html';

    for ( $self->{'arg'}->{'dtype'} ) {
        /rss|xml/ && do { $self->header_props( -type => 'text/xml' ); };
        /html/    && do { $self->header_props( -type => 'text/html; charset=UTF-8' ); };
    }

    $self->{'dbh'} = lsrfsh::MySQL->new( config => 'config.yaml' );

    $self->start_mode( 'displayTumble' );

    return $self;
}

sub displayTumble {
    my $self = shift;

    my ( $filter, $data, $r );

    if ( $self->{'arg'}->{'i'} ) {
        $filter = "DATE_SUB(CURDATE(), INTERVAL " . $self->{'arg'}->{'i'} * 6
            . " DAY) <= timestamp AND DATE_SUB(CURDATE(), INTERVAL "
            . ( $self->{'arg'}->{'i'} - 1 ) * 6 . " DAY) >= timestamp";
    }
    else {
        $filter = "DATE_SUB(CURDATE(), INTERVAL 6 DAY) <= timestamp";
    }

    foreach my $type ( qw( ircLink image quote ) ) {
        my ( $raw );

        $raw->{$type} = $self->{'dbh'}->fetch(
            source => $type,
            filter => $filter,
            key    => 'timestamp'
        );

        map {
            $data->{$_} = $raw->{$type}->{$_};
            $data->{$_}->{'type'} = $type;
        } keys %{$raw->{$type}}
    }

    my ( $c, $d, $date );

    foreach my $item ( reverse sort { $a cmp $b } keys %{$data} ) {
        my ( $content );

        if (
            $data->{$item}->{'timestamp'} =~
                /(\d{4})-(\d{2})-(\d{2})\s(\d{2}):(\d{2}):(\d{2})/
        ) {
            $data->{$item}->{'timestamp'} =
                POSIX::strftime(
                    "%a, %d %b %Y %T -0600", 0, $5, $4, $3, $2 - 1, $1 - 1900
                );

                if ( ( !$d ) || ( $3 ne $d ) ) {
                    $d = $3;

                    $date->{'day'} = POSIX::strftime(
                        "%a", 0, $5, $4, $3, $2 - 1, $1 - 1900
                    );
                    $date->{'mon'} = POSIX::strftime(
                        "%b", 0, $5, $4, $3, $2 - 1, $1 - 1900
                    );

                    $c .= $self->wrap(
                        wrapper => 'tumble_date',
                        month   => $date->{'mon'},
                        day     => $date->{'day'},
                        date    => $d
                    );
                }
        }

        for ( $data->{$item}->{'type'} ) {
                /ircLink/ && do {
                    if ( $data->{$item}->{'title'} =~ /^(http:\/\/.*)/ ) {
                        if ( length( $1 ) > 40 ) {
                            $data->{$item}->{'title'} = substr( $1, 0, 40 ) . '...';
                        }
                    }

                    my $link_filler =  $data->{$item}->{'title'};

                    # fall back to normal linking of images if they could be nsfw
                    if (($data->{$item}->{'content_type'} =~ /image/) and ($data->{$item}->{'user'} !~ /nsfw|otd/)) {
                      $link_filler =  '<img src="' .  $data->{$item}->{'url'} . '">';
                    }

                    if ($data->{$item}->{'url'} =~ /twitter/) {
                      $ENV{PATH} = "/usr/local/bin";
                      my $l = $data->{$item}->{'url'};
                      $link_filler = `/usr/local/bin/twit-link ${l}`;
                      if ($? != 0) {
                        $link_filler =  $data->{$item}->{'title'};
                      }
                    }

                    $content =
                        '<a href="http://' . $CONFIG->{'baseurl'} .
                        qq{/irclink/?} .
                        $data->{$item}->{'ircLinkID'} .
                        qq{">} .
                        $link_filler  .
                        qq{</a>}

                };

                /image/ && do {
                    $content =
                        qq{<img src="} .
                        $data->{$item}->{'url'} .
                        qq{" alt="image" />};
                };
        }

        $c .= $self->wrap(
            wrapper => 'tumble_item_' . $data->{$item}->{'type'},
            author  => $data->{$item}->{'user'},
	    baseurl => $CONFIG->{'baseurl'},
            content => $content,

            %{$data->{$item}}
        );
    }

    $c =~ s/\&/\&amp;/g;

    my ( $nav );

    if ( $self->{'arg'}->{'i'} ) {
        $nav->{'p'} = $self->{'arg'}->{'i'}+1;
        $nav->{'n'} = $self->{'arg'}->{'i'}-1;
    }
    else {
        $nav->{'p'} = 2;
        $nav->{'n'} = '';
    }

    $nav->{'p'} = qq(<a href="?i=)
        . $nav->{'p'}
        . qq("><img src="/img/prev.jpg" border="0" alt="" /></a>);
    $nav->{'n'} = qq( &nbsp;<a href="?i=)
        . $nav->{'n'}
        . qq(\"><img src="/img/next.jpg" border="0" alt="" /></a>);

    $nav->{'n'} = '' unless $self->{'arg'}->{'i'};

    if ( $self->{'arg'}->{'dtype'} =~ /html/ ) {
        $filter = qq{
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

        my ( $h );

        map {
            if ( $hot->{$_}->{'title'} =~ /^(http:\/\/.*)/ ) {
                if ( length( $1 ) > 15 ) {
                $hot->{$_}->{'title'} = substr( $1, 7, 15 ) . '...';
            }
                                                                                            }

            my $co =
                '<a href="http://' . $CONFIG->{'baseurl'} .  qq{/irclink/?} .
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
            hot       => $h,
            nav_p     => $nav->{'p'},
            nav_n     => $nav->{'n'},
	    baseurl => $CONFIG->{'baseurl'},
            container => $c
        );
    }
    else {
        return $self->wrap(
            wrapper   => 'index',
            nav_p     => $nav->{'p'},
            nav_n     => $nav->{'n'},
	    baseurl => $CONFIG->{'baseurl'},
            container => $c
        );
    }
}


sub wrap {
    my $self = shift;

    my ( $arg );
    %{$arg} = @_;

    my $template = $self->load_tmpl(
        $arg->{'wrapper'} . '.t' . $self->{'arg'}->{'dtype'},
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
