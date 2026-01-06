package tumble;

use base 'CGI::Application';

use lsrfsh::MySQL;

use DBI;
use POSIX qw( strftime );

use YAML qw( LoadFile );
use HTML::Entities;
use LWP::UserAgent;

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
                    if ( defined($data->{$item}->{'title'}) && $data->{$item}->{'title'} =~ /^(http:\/\/.*)/ ) {
                        if ( length( $1 ) > 40 ) {
                            $data->{$item}->{'title'} = substr( $1, 0, 40 ) . '...';
                        }
                    }

                    # Escape title to prevent XSS - it will be replaced with HTML for images/previews if needed
                    my $link_filler = encode_entities($data->{$item}->{'title'});
                    # Default link goes through irclink for stats tracking (clicks, views, etc.)
                    my $link_url = 'http://' . $CONFIG->{'baseurl'} . qq{/irclink/?} . $data->{$item}->{'ircLinkID'};
                    my $is_apple_photos = 0;
                    my $apple_photos_url = '';

                    # Detect Apple Photos shared links (e.g., https://www.icloud.com/photos/#/icloudlinks/...)
                    # These links need special handling because they're not direct image URLs
                    # Match various iCloud Photos URL formats: photos, sharedalbum, or any photos-related path
                    if (defined($data->{$item}->{'url'}) &&
                        ($data->{$item}->{'url'} =~ /icloud\.com.*photos/i ||
                         $data->{$item}->{'url'} =~ /icloud\.com.*sharedalbum/i)) {
                        $is_apple_photos = 1;
                        $apple_photos_url = $data->{$item}->{'url'};
                    }

                    # For Apple Photos links, fetch the page to extract the actual image URL or OpenGraph data
                    # This allows us to render images inline instead of requiring users to click through
                    if ($is_apple_photos && $data->{$item}->{'user'} !~ /nsfw|otd/) {
                      # Set up HTTP client with appropriate user agent to avoid being blocked
                      my $ua = LWP::UserAgent->new(
                          ssl_opts => { verify_hostname => 0 },
                          timeout => 10
                      );
                      $ua->agent('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36');
                      print STDERR "Fetching Apple Photos URL: $apple_photos_url\n";
                      my $response = $ua->get($apple_photos_url);
                      print STDERR "Response status: " . $response->status_line . "\n";
                      if ($response->is_success) {
                          my $html = $response->content;
                          my $is_album = 0;
                          my $og_image = '';
                          my $og_title = '';
                          my $og_description = '';

                          print STDERR "Successfully fetched HTML, length: " . length($html) . "\n";

                          # Check for OpenGraph meta tags - these indicate an album or shared collection
                          # Single photos typically don't have OpenGraph tags, albums do
                          # Use more strict regex to prevent XSS: match content attribute value until closing quote
                          # The pattern ensures we only capture the value between quotes, stopping at the first quote
                          if ($html =~ /<meta\s+property=["']og:image["']\s+content=["']([^"']*)["']/i) {
                              $og_image = $1;
                              $is_album = 1;
                              print STDERR "Found OpenGraph image: $og_image\n";
                          }
                          if ($html =~ /<meta\s+property=["']og:title["']\s+content=["']([^"']*)["']/i) {
                              $og_title = $1;
                              print STDERR "Found OpenGraph title: $og_title\n";
                          }
                          if ($html =~ /<meta\s+property=["']og:description["']\s+content=["']([^"']*)["']/i) {
                              $og_description = $1;
                              print STDERR "Found OpenGraph description: $og_description\n";
                          }
                          
                          # Debug: Check if any meta tags exist at all
                          my $meta_count = () = $html =~ /<meta/gi;
                          print STDERR "Found $meta_count meta tags in HTML\n";
                          if ($meta_count > 0 && !$og_image) {
                              # Try to find any og: tags to see what's there
                              if ($html =~ /<meta[^>]*property=["']og:([^"']+)["'][^>]*>/i) {
                                  print STDERR "Found OpenGraph property (but not image): og:$1\n";
                              }
                          }

                          # Sanitize all OpenGraph values immediately after extraction to prevent XSS
                          # These values come from untrusted remote HTML and must be treated as unsafe
                          # For URLs (og_image): escape for HTML attribute context (escapes &, <, >, ")
                          # For text (og_title, og_description): escape all HTML entities
                          
                          # Filter out OpenGraph images that are just logos/icons - not actual photos
                          if ($og_image && ($og_image =~ /(?:logo|icon|favicon|icloud_logo)/i || 
                              $og_image !~ /icloud-content\.com/i)) {
                              print STDERR "Skipping OpenGraph image (appears to be logo/icon): $og_image\n";
                              $og_image = '';
                              $is_album = 0;
                          }
                          
                          my $escaped_og_image = $og_image ? encode_entities($og_image) : '';
                          my $escaped_og_title = $og_title ? encode_entities($og_title) : '';
                          my $escaped_og_description = $og_description ? encode_entities($og_description) : '';

                          # Handle albums: render a rich preview card with image, title, and description
                          # This gives users a better sense of what's in the album before clicking
                          # Only use OpenGraph if we have a valid photo URL (not a logo)
                          if ($is_album && $og_image) {
                              my $preview_html = '<div style="border: 1px solid #ddd; border-radius: 4px; padding: 10px; max-width: 500px; background: #f9f9f9;">';
                              if ($escaped_og_image) {
                                  my $escaped_og_title_alt = $escaped_og_title || encode_entities('Apple Photos album');
                                  $preview_html .= '<img src="' . $escaped_og_image . '" style="max-width: 100%; height: auto; border-radius: 4px; margin-bottom: 8px;" alt="' . $escaped_og_title_alt . '">';
                              }
                              if ($escaped_og_title) {
                                  # Already HTML-escaped, safe to use in text content
                                  $preview_html .= '<div style="font-weight: bold; margin-bottom: 4px;">' . $escaped_og_title . '</div>';
                              }
                              if ($escaped_og_description) {
                                  # Already HTML-escaped, safe to use in text content
                                  $preview_html .= '<div style="font-size: 0.9em; color: #666;">' . $escaped_og_description . '</div>';
                              }
                              $preview_html .= '</div>';
                              $link_filler = $preview_html;
                          } else {
                              # Handle single photos: extract the direct image URL from the page HTML
                              # We try multiple patterns because Apple's HTML structure may vary
                              # Apple Photos pages are JS-heavy, so URLs may be in JSON/script tags
                              my $img_url = '';
                              
                              # Helper function to check if URL is likely a logo/icon (not a photo)
                              my $is_logo_or_icon = sub {
                                  my $url = shift;
                                  return 1 if $url =~ /(?:logo|icon|favicon|sprite|button|badge)/i;
                                  return 1 if $url =~ /icloud_logo/i;
                                  return 1 if $url =~ /\.(js|css|html|json|svg)$/i;
                                  return 0;
                              };
                              
                              # Pattern 1: Look for downloadURL in JSON (highest priority - most reliable)
                              if ($html =~ /"downloadURL"\s*:\s*"([^"]+)"/i) {
                                  $img_url = $1;
                                  $img_url =~ s/\\\//\//g;  # Unescape JSON-encoded slashes
                                  $img_url =~ s/\\u([0-9a-fA-F]{4})/chr(hex($1))/eg;  # Unescape Unicode
                                  unless ($is_logo_or_icon->($img_url)) {
                                      print STDERR "Found image via Pattern 1 (downloadURL): $img_url\n";
                                  } else {
                                      $img_url = '';
                                  }
                              }
                              
                              # Pattern 2: Look for iCloud CDN URLs (cvws.icloud-content.com - these are actual photos)
                              if (!$img_url && $html =~ /(https?:\/\/[^"'\s<>]*icloud-content\.com[^"'\s<>]+\.(jpg|jpeg|png|gif|webp))/i) {
                                  $img_url = $1;
                                  unless ($is_logo_or_icon->($img_url)) {
                                      print STDERR "Found image via Pattern 2 (iCloud CDN): $img_url\n";
                                  } else {
                                      $img_url = '';
                                  }
                              }
                              
                              # Pattern 3: Look for image URLs in JSON data structures with photo-specific field names
                              if (!$img_url && $html =~ /"(?:photoUrl|imageUrl|previewUrl|thumbnailUrl)"\s*:\s*"([^"]+\.(jpg|jpeg|png|gif|webp))"/i) {
                                  $img_url = $1;
                                  $img_url =~ s/\\\//\//g;  # Unescape JSON-encoded slashes
                                  $img_url =~ s/\\u([0-9a-fA-F]{4})/chr(hex($1))/eg;  # Unescape Unicode
                                  unless ($is_logo_or_icon->($img_url)) {
                                      print STDERR "Found image via Pattern 3 (JSON photo field): $img_url\n";
                                  } else {
                                      $img_url = '';
                                  }
                              }
                              
                              # Pattern 4: Look in script tags for JSON data with downloadURL or photo URLs
                              if (!$img_url && $html =~ /<script[^>]*>.*?"(?:downloadURL|photoUrl|imageUrl)"\s*:\s*"([^"]+)"/is) {
                                  $img_url = $1;
                                  $img_url =~ s/\\\//\//g;
                                  $img_url =~ s/\\u([0-9a-fA-F]{4})/chr(hex($1))/eg;
                                  unless ($is_logo_or_icon->($img_url)) {
                                      print STDERR "Found image via Pattern 4 (script tag JSON): $img_url\n";
                                  } else {
                                      $img_url = '';
                                  }
                              }
                              
                              # Pattern 5: Look for <img> tags but skip logos/icons
                              if (!$img_url && $html =~ /<img[^>]+src=["']([^"']+\.(jpg|jpeg|png|gif|webp))["']/i) {
                                  $img_url = $1;
                                  # Convert relative URLs to absolute
                                  if ($img_url !~ /^https?:/) {
                                      $img_url = 'https://www.icloud.com' . $img_url if $img_url =~ /^\//;
                                  }
                                  unless ($is_logo_or_icon->($img_url)) {
                                      print STDERR "Found image via Pattern 5 (img tag): $img_url\n";
                                  } else {
                                      $img_url = '';
                                  }
                              }
                              
                              # Pattern 6: Look for base64 data URLs (less common but possible)
                              if (!$img_url && $html =~ /data:image\/(jpeg|jpg|png|gif|webp);base64,([A-Za-z0-9+\/]{100,}={0,2})/i) {
                                  # Only accept base64 if it's reasonably large (likely a photo, not an icon)
                                  $img_url = "data:image/$1;base64,$2";
                                  print STDERR "Found image via Pattern 6 (base64 data URL)\n";
                              }

                              if ($img_url) {
                                  # Successfully extracted image URL - render it inline
                                  # The image will be wrapped in an irclink anchor below for stats tracking
                                  print STDERR "Extracted image URL: $img_url\n";
                                  my $escaped_img_url = encode_entities($img_url, '<>"');
                                  $link_filler = '<img src="' . $escaped_img_url . '" alt="Apple Photos image" style="max-width: 100%; height: auto;">';
                              } else {
                                  # Couldn't extract image URL - fall back to showing the title as a link
                                  # This maintains functionality even if Apple changes their page structure
                                  print STDERR "Could not extract image URL from HTML. Sample HTML (first 500 chars): " . substr($html, 0, 500) . "\n";
                                  $link_filler = encode_entities($data->{$item}->{'title'});
                              }
                          }
                      } else {
                          # HTTP request failed - log error and fall back to default link display
                          my $status = $response->status_line;
                          my $error_msg = $response->message || 'Unknown error';
                          print STDERR "Failed to fetch Apple Photos URL: $apple_photos_url - Status: $status ($error_msg)\n";
                          # $link_filler already contains the escaped title from initialization, so no action needed
                      }
                    }
                    # fall back to normal linking of images if they could be nsfw
                    elsif (defined($data->{$item}->{'content_type'}) &&
                           defined($data->{$item}->{'user'}) &&
                           ($data->{$item}->{'content_type'} =~ /image/) &&
                           ($data->{$item}->{'user'} !~ /nsfw|otd/)) {
                      my $escaped_url = encode_entities($data->{$item}->{'url'}, '<>"');
                      $link_filler =  '<img src="' .  $escaped_url . '">';
                    }

                    if (defined($data->{$item}->{'url'}) && $data->{$item}->{'url'} =~ /twitter/) {
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

                    # Wrap all content in an irclink anchor for stats tracking
                    # The irclink endpoint increments click/view counters, then redirects to the original URL
                    # This preserves analytics while still allowing users to access the original content
                    my $escaped_link_url = encode_entities($link_url, '<>"');
                    $content =
                        '<a href="' . $escaped_link_url . qq{">} .
                        $link_filler  .
                        qq{</a>};

                };

                /image/ && do {
                    my $escaped_url = encode_entities($data->{$item}->{'url'}, '<>"');
                    $content =
                        qq{<img src="} .
                        $escaped_url .
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

    $c =~ s/\&/\&amp;/g if $c;

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

            my $escaped_title = encode_entities($hot->{$_}->{'title'});
            my $co =
                '<a href="http://' . $CONFIG->{'baseurl'} .  qq{/irclink/?} .
                $hot->{$_}->{'ircLinkID'} .
                qq{">} .
                $escaped_title .
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
