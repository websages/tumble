package tumble::Content;

use strict;
use warnings;

use POSIX qw( strftime );
use Time::Local qw( timelocal timegm );
use LWP::Simple qw(get);
use JSON qw(from_json);

sub new {
    my ($class, %args) = @_;
    my $self = {
        config => $args{config} || {},
        fetcher => $args{fetcher} || \&LWP::Simple::get,
    };
    bless $self, $class;
    return $self;
}

sub process_item {
    my ($self, $item_in) = @_;
    my $item = { %$item_in };
    
    $self->_process_dates($item);
    $self->_process_content($item);
    
    return $item;
}

sub _process_dates {
    my ($self, $item) = @_;
    if ($item->{timestamp} && $item->{timestamp} =~ /(\d{4})-(\d{2})-(\d{2})\s(\d{2}):(\d{2}):(\d{2})/) {
        my ($year, $month, $day, $hour, $minute, $second) = ($1, $2, $3, $4, $5, $6);
        
        my $epoch = timelocal($second, $minute, $hour, $day, $month - 1, $year - 1900);
        my @lt = localtime($epoch);
        
        my $gmt_epoch = timegm(@lt);
        my $local_epoch = timelocal(@lt);
        my $tz_offset_seconds = $local_epoch - $gmt_epoch;
        my $tz_offset_minutes = $tz_offset_seconds / 60;
        
        my $tz_sign = $tz_offset_minutes >= 0 ? '+' : '-';
        my $tz_hours = abs(int($tz_offset_minutes / 60));
        my $tz_mins = abs(int($tz_offset_minutes % 60));
        my $tz_offset = sprintf("%s%02d%02d", $tz_sign, $tz_hours, $tz_mins);
        
        $item->{timestamp} = POSIX::strftime("%a, %d %b %Y %H:%M:%S $tz_offset", @lt);
        
        # Capture date parts for grouping if needed
        $item->{_date_components} = {
            day   => POSIX::strftime("%a", 0, $5, $4, $3, $2 - 1, $1 - 1900),
            mon   => POSIX::strftime("%b", 0, $5, $4, $3, $2 - 1, $1 - 1900),
            raw_day => $3
        };
    }
}

sub _process_content {
    my ($self, $item) = @_;
    
    if ($item->{type} eq 'ircLink') {
        $self->_process_irclink($item);
    } elsif ($item->{type} eq 'image') {
        $item->{content} = qq{<img src="} . $item->{url} . qq{" alt="image" />};
    } elsif ($item->{type} eq 'quote') {
        my $quote_text = $item->{quote} || '';
        my $author_text = $item->{author} || '';
        $item->{content} = '"' . $quote_text . '" --' . $author_text;
    }
}

sub _process_irclink {
    my ($self, $item) = @_;
    
    # Title truncation
    if ($item->{title} =~ /^(http:\/\/.*)/) {
        if (length($1) > 40) {
            $item->{title} = substr($1, 0, 40) . '...';
        }
    }

    my $link_filler = $item->{title};
    
    # Image content type check
    if (($item->{content_type} && $item->{content_type} =~ /image/) && 
        ($item->{user} && $item->{user} !~ /nsfw|otd/)) {
        $link_filler = '<img src="' . $item->{url} . '">';
    }
    
    my $is_youtube = 0;
    
    # Twitter
    if ($item->{url} && $item->{url} =~ /twitter/) {
        my @parts = split('/', $item->{url});
        my $id = $parts[-1];
        # basic check
        if ($id =~ /[0-9]+/) {
            my $tw_uri = "https://api.twitter.com/1/statuses/oembed.json?id=" . $id;
            my $tw_j = $self->{fetcher}->($tw_uri);
            if ($tw_j) {
                my $stuff = eval { from_json($tw_j) };
                if ($stuff && $stuff->{html}) {
                    $link_filler = $stuff->{html};
                }
            }
        }
    }
    
    # YouTube
    if ($item->{url} && $item->{url} =~ /youtube\.com|youtu\.be/i) {
        my $video_id;
        my $url = $item->{url};
        if ($url =~ /(?:youtube\.com\/watch\?v=|youtube\.com\/embed\/|youtu\.be\/)([a-zA-Z0-9_-]{11})/i) {
            $video_id = $1;
        } elsif ($url =~ /youtube\.com\/watch\?.*[&?]v=([a-zA-Z0-9_-]{11})/i) {
            $video_id = $1;
        }
        
        if ($video_id) {
            $item->{content} = '<div class="youtube-embed-wrapper">' .
                               '<iframe width="560" height="315" ' .
                               'src="https://www.youtube.com/embed/' . $video_id . '?rel=0" ' .
                               'frameborder="0" ' .
                               'allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" ' .
                               'allowfullscreen></iframe>' .
                               '</div>';
            $is_youtube = 1;
        }
    }
    
    unless ($is_youtube) {
        my $baseurl = $self->{config}->{baseurl} || '';
        $item->{content} = '<a href="http://' . $baseurl .
                           qq{/irclink/?} .
                           ($item->{ircLinkID} || '') .
                           qq{">} .
                           $link_filler .
                           qq{</a>};
    }
}

1;
