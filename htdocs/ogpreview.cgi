#!/usr/bin/perl -w

BEGIN { unshift @INC, '../lib'; }

use CGI;
use LWP::UserAgent;
use JSON;

use strict;

# Try to use HTML::TreeBuilder if available, otherwise fall back to regex parsing
eval {
    require HTML::TreeBuilder;
    HTML::TreeBuilder->import();
};
my $has_treebuilder = !$@;

my $cgi = new CGI;
my $url = $cgi->param('url');

unless ($url) {
    print "Content-type: application/json\n\n";
    print encode_json({ error => 'No URL provided' });
    exit;
}

# Validate URL
unless ($url =~ /^https?:\/\//) {
    print "Content-type: application/json\n\n";
    print encode_json({ error => 'Invalid URL' });
    exit;
}

my $agentString = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10.16; rv:84.0) Gecko/20100101 Firefox/84.0';

my $agent = LWP::UserAgent->new(
    ssl_opts => { verify_hostname => 0 },
    protocols_allowed => ['https', 'http'],
    timeout => 10,
);
$agent->agent($agentString);

my $response = $agent->get($url);

unless ($response->is_success) {
    print "Content-type: application/json\n\n";
    print encode_json({ error => 'Failed to fetch URL' });
    exit;
}

my $content = $response->decoded_content;
my $result = {};

if ($has_treebuilder) {
    # Use HTML::TreeBuilder if available
    my $tree = HTML::TreeBuilder->new;
    $tree->parse($content);
    $tree->eof();

    # Extract Open Graph tags
    my @og_tags = $tree->look_down(_tag => 'meta', sub {
        my $attr = $_[0]->attr('property');
        return $attr && $attr =~ /^og:/;
    });

    foreach my $tag (@og_tags) {
        my $property = $tag->attr('property');
        my $content = $tag->attr('content');
        if ($property && $content) {
            $property =~ s/^og://;
            $result->{$property} = $content;
        }
    }

    # Extract Twitter Card tags
    my @twitter_tags = $tree->look_down(_tag => 'meta', sub {
        my $attr = $_[0]->attr('name');
        return $attr && $attr =~ /^twitter:/;
    });

    foreach my $tag (@twitter_tags) {
        my $name = $tag->attr('name');
        my $content = $tag->attr('content');
        if ($name && $content) {
            $name =~ s/^twitter://;
            $result->{"twitter_$name"} = $content;
        }
    }

    # Fallback to standard meta tags if no OG/Twitter tags found
    unless (keys %$result) {
        my $title_tag = $tree->look_down(_tag => 'title');
        if ($title_tag) {
            $result->{title} = $title_tag->as_text;
        }
        
        my $desc_tag = $tree->look_down(_tag => 'meta', sub {
            $_[0]->attr('name') && lc($_[0]->attr('name')) eq 'description';
        });
        if ($desc_tag) {
            $result->{description} = $desc_tag->attr('content');
        }
    }

    $tree->delete();
} else {
    # Fallback to regex parsing if HTML::TreeBuilder is not available
    # Extract Open Graph tags - try both attribute orders
    while ($content =~ /<meta\s+property=["']og:([^"']+)["']\s+content=["']([^"']+)["']/gi ||
           $content =~ /<meta\s+content=["']([^"']+)["']\s+property=["']og:([^"']+)["']/gi) {
        my $property = $1 || $4;
        my $value = $2 || $3;
        if ($property && $value) {
            $result->{$property} = $value;
        }
    }
    
    # Extract Twitter Card tags - try both attribute orders
    while ($content =~ /<meta\s+name=["']twitter:([^"']+)["']\s+content=["']([^"']+)["']/gi ||
           $content =~ /<meta\s+content=["']([^"']+)["']\s+name=["']twitter:([^"']+)["']/gi) {
        my $name = $1 || $4;
        my $value = $2 || $3;
        if ($name && $value) {
            $result->{"twitter_$name"} = $value;
        }
    }
    
    # Extract title
    if ($content =~ /<title[^>]*>([^<]+)<\/title>/i) {
        $result->{title} = $1 unless $result->{title};
    }
    
    # Extract description meta tag - try both attribute orders
    if ($content =~ /<meta\s+name=["']description["']\s+content=["']([^"']+)["']/i ||
        $content =~ /<meta\s+content=["']([^"']+)["']\s+name=["']description["']/i) {
        my $desc = $1;
        $result->{description} = $desc unless $result->{description} || !$desc;
    }
}

print "Content-type: application/json\n\n";
print encode_json($result);

