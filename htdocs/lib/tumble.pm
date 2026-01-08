package tumble;

use base 'CGI::Application';

use lsrfsh::MySQL;

use DBI;
use POSIX qw( strftime );
use Time::Local qw( timelocal timegm );
use Cwd qw( abs_path getcwd );
use File::Spec;

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
        /rss|xml/ && do { $self->header_props( -type => 'text/xml; charset=UTF-8' ); };
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
            # Parse the timestamp: $1=year, $2=month, $3=day, $4=hour, $5=minute, $6=second
            my ($year, $month, $day, $hour, $minute, $second) = ($1, $2, $3, $4, $5, $6);
            
            # Convert to epoch time (month is 0-based in timelocal)
            # Note: timelocal interprets time as local time based on server timezone
            my $epoch = timelocal($second, $minute, $hour, $day, $month - 1, $year - 1900);
            
            # Format as RFC 822 date (RSS pubDate format)
            # Get localtime components for the timestamp
            my @lt = localtime($epoch);
            
            # Calculate timezone offset by comparing GMT and local time
            # timegm returns GMT epoch for given local time components
            my $gmt_epoch = timegm(@lt);
            my $local_epoch = timelocal(@lt);
            my $tz_offset_seconds = $local_epoch - $gmt_epoch;
            my $tz_offset_minutes = $tz_offset_seconds / 60;
            
            # Format timezone offset as +HHMM or -HHMM
            my $tz_sign = $tz_offset_minutes >= 0 ? '+' : '-';
            my $tz_hours = abs(int($tz_offset_minutes / 60));
            my $tz_mins = abs(int($tz_offset_minutes % 60));
            my $tz_offset = sprintf("%s%02d%02d", $tz_sign, $tz_hours, $tz_mins);
            
            $data->{$item}->{'timestamp'} =
                POSIX::strftime("%a, %d %b %Y %H:%M:%S $tz_offset", @lt);

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
                      use LWP::Simple;
                      use JSON;
                      my @parts = split('/' , $data->{$item}->{'url'});
                      my $id = $parts[-1];
                      # This is so URIs like id/photos/1 don't try to call json
                      next if $id !~ /[0-9]+/;
                      next if $#parts > 6;
                      my $tw_uri = "https://api.twitter.com/1/statuses/oembed.json?id=" . $id;
                      my $tw_j = get( $tw_uri );
                      next unless $tw_j;
                      my $stuff = from_json($tw_j);
                      $link_filler = $stuff->{'html'};
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
                
                /quote/ && do {
                    # For quote items, build description text
                    my $quote_text = $data->{$item}->{'quote'} || '';
                    my $author_text = $data->{$item}->{'author'} || '';
                    # Build the description text - will be escaped in wrap() function
                    $content = '"' . $quote_text . '" --' . $author_text;
                };
        }

        # For XML/RSS feeds, wrap HTML content in CDATA sections (for ircLink and image items)
        my $xml_content = $content;
        if ( $self->{'arg'}->{'dtype'} =~ /xml|rss/ && defined $content && $content ne '' && $data->{$item}->{'type'} ne 'quote' ) {
            # Wrap HTML content in CDATA for RSS descriptions (quote already has CDATA)
            $xml_content = '<![CDATA[' . $content . ']]>';
        }

        my %template_vars = (
            wrapper => 'tumble_item_' . $data->{$item}->{'type'},
            author  => $data->{$item}->{'user'},
            baseurl => $CONFIG->{'baseurl'},
            %{$data->{$item}}
        );
        
        # Add content or description depending on item type
        if ( $data->{$item}->{'type'} eq 'quote' ) {
            # For quote items, pass description (will be escaped in wrap() function)
            $template_vars{'description'} = $content if defined $content;
        } elsif ( defined $xml_content ) {
            $template_vars{'content'} = $xml_content;
        }

        $c .= $self->wrap( %template_vars );
    }

    # Escape XML special characters for non-CDATA sections
    if ( $self->{'arg'}->{'dtype'} =~ /xml|rss/ && $c ) {
        # Only escape if not already in CDATA sections
        # Escape & first, then other characters
        $c =~ s/&(?!lt;|gt;|amp;|quot;|apos;|#\d+;|#x[0-9a-f]+;)/&amp;/gi;
        # Don't escape < and > that are already in CDATA sections
        # But we need to escape them outside CDATA and in titles/links
        # Since we're wrapping descriptions in CDATA, we only need to escape in titles and links
        # This is handled by the XML escaping function below for non-CDATA content
    } elsif ( $c ) {
        # For HTML output, escape ampersands
        $c =~ s/\&/\&amp;/g;
    }

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


sub get_git_commit {
    my $self = shift;

    # Read git commit hash directly from .git files
    # This avoids needing the git command which might not be in PATH
    # Wrap in eval to prevent any errors from causing output
    my $git_commit;
    eval {
        local $SIG{__WARN__} = sub {};  # Suppress any warnings
        
        # Get current working directory
        my $cwd = getcwd() || '.';
        my @dirs_to_try = ();
        
        # Method 1: Try from current directory (might be htdocs/)
        push @dirs_to_try, $cwd;
        
        # Method 2: Go up one level from current directory
        my $parent = File::Spec->catdir($cwd, File::Spec->updir());
        $parent = abs_path($parent) if -d $parent;
        push @dirs_to_try, $parent if $parent && -d $parent;
        
        # Method 3: Try from template path
        if ($self->tmpl_path()) {
            my $tmpl_abs = abs_path($self->tmpl_path());
            if ($tmpl_abs) {
                my $tmpl_dir = $tmpl_abs;
                $tmpl_dir =~ s|/[^/]+$||;  # Remove thtml/
                push @dirs_to_try, $tmpl_dir;
                $tmpl_dir =~ s|/[^/]+$||;  # Remove htdocs/
                push @dirs_to_try, $tmpl_dir if $tmpl_dir && -d $tmpl_dir;
            }
        }
        
        # Try each directory until we find one with .git
        for my $dir (@dirs_to_try) {
            next unless $dir && -d $dir;
            
            my $git_dir = File::Spec->catdir($dir, '.git');
            next unless -d $git_dir || -f $git_dir;
            
            # Read HEAD file
            my $head_file = File::Spec->catfile($git_dir, 'HEAD');
            next unless -f $head_file;
            
            if (open(my $fh, '<', $head_file)) {
                my $head = <$fh>;
                close($fh);
                chomp $head if $head;
                
                # If HEAD points to a ref, follow it
                if ($head && $head =~ /^ref: (.+)$/) {
                    my $ref_file = File::Spec->catfile($git_dir, $1);
                    if (-f $ref_file && open($fh, '<', $ref_file)) {
                        $head = <$fh>;
                        close($fh);
                        chomp $head if $head;
                    } else {
                        next;
                    }
                }
                
                # Extract short commit hash (first 7 characters)
                if ($head && $head =~ /^([0-9a-f]{7,})/i) {
                    $git_commit = substr($1, 0, 7);
                    last;
                }
            }
        }
    };
    # Only return if it looks like a valid commit hash (7+ hex chars)
    return $git_commit if $git_commit && $git_commit =~ /^[0-9a-f]{7}$/i;
    return undef;
}

sub wrap {
    my $self = shift;

    my ( $arg );
    %{$arg} = @_;

    my $wrapper_name = $arg->{'wrapper'};
    my $template = $self->load_tmpl(
        $wrapper_name . '.t' . $self->{'arg'}->{'dtype'},
        die_on_bad_params => 0
    );

    delete $arg->{'wrapper'};

    # Add git commit hash and URL for index template
    if ($wrapper_name eq 'index') {
        my $git_commit = $self->get_git_commit();
        if ($git_commit) {
            $arg->{'git_commit'} = $git_commit;
            $arg->{'git_commit_url'} = "https://github.com/websages/tumble/commit/$git_commit";
        }
    }

    # Escape XML special characters for text fields in XML/RSS output
    my $is_xml = $self->{'arg'}->{'dtype'} =~ /xml|rss/;
    
    map {
        my $value = $arg->{$_};
        chomp( $value ) if ref $value;
        
        # For XML output, escape text fields (but not content which should be CDATA)
        if ( $is_xml && !ref $value && $_ ne 'content' && $_ ne 'container' ) {
            # Escape XML special characters for titles, links, descriptions, etc.
            # Must escape & first, then < and >
            $value =~ s/&/&amp;/g;
            $value =~ s/</&lt;/g;
            $value =~ s/>/&gt;/g;
            # Escape quotes for attribute safety (though we're using in content, not attributes)
            # Also escape quotes in description text to be safe
            if ( $_ eq 'description' ) {
                $value =~ s/"/&quot;/g;
            }
        }
        
        $template->param( $_ => $value );
    } keys %{$arg};

    return $template->output();
}



1;
