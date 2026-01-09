use strict;
use warnings;
use Test::More;
use FindBin;
use lib "$FindBin::Bin/../htdocs/lib";

BEGIN { use_ok('tumble::Content') };

# Mock fetcher for Twitter
sub mock_fetcher {
    my ($url) = @_;
    if ($url =~ /api.twitter.com/) {
        return '{"html": "<blockquote>Mock Tweet</blockquote>"}';
    }
    return undef;
}

my $config = { baseurl => 'tumble.example.com' };
my $processor = tumble::Content->new(
    config => $config,
    fetcher => \&mock_fetcher
);

subtest 'process_item: date formatting' => sub {
    my $item = {
        timestamp => '2023-10-27 10:00:00',
        type => 'text',
        title => 'Test Title',
    };
    
    my $processed = $processor->process_item($item);
    
    ok($processed->{timestamp}, 'Timestamp converted');
    like($processed->{timestamp}, qr/^\w+, \d+ \w+ \d{4}/, 'Timestamp looks like RFC 822');
};

subtest 'process_item: youtube embed' => sub {
    my $item = {
        type => 'ircLink',
        url => 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
        title => 'Rick Roll',
    };
    
    my $processed = $processor->process_item($item);
    
    like($processed->{content}, qr/iframe/, 'Contains iframe for YouTube');
    like($processed->{content}, qr/dQw4w9WgXcQ/, 'Contains video ID');
};

subtest 'process_item: twitter embed' => sub {
    my $item = {
        type => 'ircLink',
        url => 'https://twitter.com/user/status/1234567890',
        title => 'Tweet',
    };
    
    my $processed = $processor->process_item($item);
    
    # logic: if twitter, content is the oembed html wrapped in a link? 
    # Original code: $link_filler = $stuff->{'html'};
    # Then: $content = <a ...>$link_filler</a>
    # Wait, the original code sets $link_filler.
    
    # We need to verify that our new module produces the 'content' field similarly.
    # But wait, original code constructs the <a href...> wrapper around $link_filler.
    # So we expect the content to contain the mocked HTML "<blockquote>Mock Tweet</blockquote>"
    
    like($processed->{content}, qr/Mock Tweet/, 'Contains mocked tweet content');
};

subtest 'process_item: normal link construction' => sub {
    my $item = {
        type => 'ircLink',
        url => 'http://example.com',
        title => 'Example',
        ircLinkID => 123,
    };
    
    my $processed = $processor->process_item($item);
    # original: <a href="http://$baseurl/irclink/?$id">$title</a>
    
    like($processed->{content}, qr/href="http:\/\/tumble.example.com\/irclink\/\?123"/, 'Link constructed with baseurl');
    like($processed->{content}, qr/>Example<\/a>/, 'Link text is title');
};

done_testing();
