package tumble;

use base 'CGI::Application';

use tumble::DB;

use DBI;
use strict;
use warnings;

use tumble::Content;
use YAML qw( LoadFile );
use Cwd qw( abs_path getcwd );
use File::Spec;

my $CONFIG = LoadFile( 'config.yaml' );


sub setup {
    my $self = shift;

    # Ensure STDOUT handles UTF-8 to prevent "Wide character in print" errors
    binmode(STDOUT, ':encoding(UTF-8)');

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

    # Initialize database connection with error handling
    eval {
        $self->{'dbh'} = tumble::DB->new( config => 'config.yaml' );
    };
    if ($@) {
        warn "[tumble] FATAL: Failed to initialize database connection\n";
        warn "[tumble] Error: $@\n";
        warn "[tumble] Please check:\n";
        warn "[tumble]   1. config.yaml exists and is readable\n";
        warn "[tumble]   2. Database server is running (if using MySQL)\n";
        warn "[tumble]   3. Database file exists and is readable (if using SQLite)\n";
        warn "[tumble]   4. Database credentials are correct\n";
        die "Database initialization failed: $@";
    }
    
    $self->{'content_processor'} = tumble::Content->new( config => $CONFIG );

    $self->start_mode( 'displayTumble' );

    return $self;
}

sub displayTumble {
    my $self = shift;

    my ( $filter, $data, $r );

    if ( $self->{'arg'}->{'i'} ) {
        my $start_days = $self->{'arg'}->{'i'} * 6;
        my $end_days = ( $self->{'arg'}->{'i'} - 1 ) * 6;

        $filter = $self->{'dbh'}->date_interval_sql($start_days) . " <= timestamp AND " .
                  $self->{'dbh'}->date_interval_sql($end_days) . " >= timestamp";
    }
    else {
        $filter = $self->{'dbh'}->date_interval_sql(6) . " <= timestamp";
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
    
    # Check if we have any data at all
    if (!$data || (ref($data) eq 'HASH' && scalar(keys %$data) == 0)) {
        warn "[tumble] WARNING: No content found in database\n";
        warn "[tumble] Filter used: $filter\n";
        warn "[tumble] This could mean:\n";
        warn "[tumble]   1. Database is empty (no content has been added yet)\n";
        warn "[tumble]   2. No content matches the current date filter\n";
        warn "[tumble]   3. Database tables exist but contain no rows\n";
        warn "[tumble] Try:\n";
        warn "[tumble]   - Adding some content to the database\n";
        warn "[tumble]   - Checking if content exists: SELECT COUNT(*) FROM ircLink;\n";
        warn "[tumble]   - Verifying the date filter is not too restrictive\n";
    }

    my ( $c, $d, $date );

    foreach my $item_id ( reverse sort { $a cmp $b } keys %{$data} ) {
        my $item = $data->{$item_id};
        
        # Delegate processing to Tumble::Content
        # This handles date formatting, twitter/youtube embeds, title truncation etc.
        my $processed = $self->{'content_processor'}->process_item( $item );
        
        # Update the data hash with processed values
        $data->{$item_id} = $processed;
        my $formatted_timestamp = $processed->{'timestamp'};
        my $content = $processed->{'content'};

        # Date header logic
        if ( defined $processed->{_date_components} ) {
            my $comps = $processed->{_date_components};
            
            if ( ( !$d ) || ( $comps->{'raw_day'} ne $d ) ) {
                $d = $comps->{'raw_day'};

                $date->{'day'} = $comps->{'day'};
                $date->{'mon'} = $comps->{'mon'};

                $c .= $self->wrap(
                    wrapper => 'tumble_date',
                    month   => $date->{'mon'},
                    day     => $date->{'day'},
                    date    => $d
                );
            }
        }

        my %template_vars = (
            wrapper => 'tumble_item_' . $processed->{'type'},
            author  => $processed->{'user'},
            baseurl => $CONFIG->{'baseurl'},
            %{$processed}
        );
        
        # Add content or description depending on item type
        # For XML/RSS feeds, wrap HTML content in CDATA sections (for ircLink and image items)
        my $xml_content = $content;
        if ( $self->{'arg'}->{'dtype'} =~ /xml|rss/ && defined $content && $content ne '' && $processed->{'type'} ne 'quote' ) {
            # Wrap HTML content in CDATA for RSS descriptions (quote already has CDATA)
            $xml_content = '<![CDATA[' . $content . ']]>';
        }

        if ( $processed->{'type'} eq 'quote' ) {
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
        $filter = $self->{'dbh'}->date_interval_sql(12) . " <= timestamp" .
                  " AND " .
                  $self->{'dbh'}->date_interval_sql(6) . " >= timestamp" .
                  " AND clicks > 1";

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
