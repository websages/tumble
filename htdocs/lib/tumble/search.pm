package tumble::search;

use base 'CGI::Application';

use tumble::DB;
use YAML qw( LoadFile );
use Cwd qw( abs_path getcwd );
use File::Spec;

use DBI;

use strict;
use warnings;

my $CONFIG = LoadFile( 'config.yaml' );



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

    $self->{'dbh'} = tumble::DB->new( config => 'config.yaml' );

    $self->start_mode( 'displaySearch' );

    return $self;
}

sub displaySearch {
    my $self = shift;

    my $string = 'unicorn';
    return unless $string;

    my $search_filter = $self->{'dbh'}->fulltext_search_sql('title,url', $self->{'arg'}->{'search'});

    my $raw = $self->{'dbh'}->fetch(
        source => 'ircLink',
        filter => $search_filter,
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
                '<a href="http://'  .  $CONFIG->{'baseurl'} . '/irclink/?' .
                $raw->{$item}->{'ircLinkID'} .
                qq{">} .
                $raw->{$item}->{'title'} .
                qq{</a>};

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

    my $filter = $self->{'dbh'}->date_interval_sql(12) . " <= timestamp" .
                 " AND " .
                 $self->{'dbh'}->date_interval_sql(6) . " >= timestamp" .
                 " AND clicks > 1";

    my $hot = $self->{'dbh'}->fetch(
        source => 'ircLink',
        filter => $filter,
        limit => 5,
        key => 'timestamp'
    );

    map {
        my $co =
            "<a href=\"http://" .  $CONFIG->{'baseurl'} .  qq{/irclink/?} .
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
        $wrapper_name . '.thtml',
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

    map {
        chomp( $arg->{$_} ) if ref $arg->{$_};
        $template->param( $_ => $arg->{$_} );
    } keys %{$arg};

    return $template->output();
}



1;
