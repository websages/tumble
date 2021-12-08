#!/usr/bin/perl -w

BEGIN { unshift @INC, '../lib'; }

use lsrfsh::MySQL;

use CGI;
use DBI;
use LWP::UserAgent;

use strict;

my $cgi = new CGI;
my $dbh = lsrfsh::MySQL->new( config => '../config.yaml' );

if ( $cgi->param( 'user' ) && $cgi->param( 'url' ) ) {
    my $user = $cgi->param( 'user' );
    my $url  = $cgi->param( 'url' );
    my $agentString = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10.16; rv:84.0) Gecko/20100101 Firefox/84.0';

    my $agent = LWP::UserAgent->new(
        ssl_opts => { verify_hostname => 0 },
        protocols_allowed => ['https', 'http'],
    );
    $agent->agent( $agentString );

    my $response = $agent->get( $url );
    if($response->{'_rc'} eq "302"){
        print STDERR "Redirect: ".$response->{'_headers'}->{'location'}."\n";
        $url = $response->{'_headers'}->{'location'};
        my $redir_response = $agent->get($url);
        if($redir_response->{'_rc'} ne "200"){
            print STDERR "Redirect got ".$redir_response->{'_rc'}."\n";
            print "Content-type: text/plain\n\n";
            print '0';
            exit( 0 );
        }
    }elsif( $response->{'_rc'} ne "200"){
        print "Content-type: text/plain\n\n";
        print '0';
        exit( 0 );
    }

    my $title = $agent->get( $url )->title();

    my $content_type;
    if ($response->{'_headers'}->{'content-type'} =~ /image/)
    {
        $content_type = 'image';
    } else {
        $content_type = 0;
    }

    $title ||= $url;

    my $sth = $dbh->prepare( qq{
        INSERT INTO ircLink (
            user, title, url, content_type
        ) VALUES (
            ?, ?, ?, ?
        )
    } );

    $sth->execute( $user, $title, $url, $content_type );

    if ( $cgi->param( 'source' ) eq 'irc' ) {
        print "Content-type: text/plain\n\n";

        $url =~ s/'/\\'/g;
        $url =~ s/"/\\"/g;

        print $dbh->selectrow_array( qq{
            SELECT ircLinkID FROM ircLink
            WHERE url = '$url'
        } );
    }
    else {
        print "Content-type: text/html\n\n";

        print qq(<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/><html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en">
<head>
    <title>tumblefish link posted</title>
    <script type="text/javascript">
      var _gaq = _gaq || [];
      _gaq.push(['_setAccount', 'UA-24593498-1']);
      _gaq.push(['_trackPageview']);
      (function() {
        var ga = document.createElement('script'); ga.type = 'text/javascript'; ga.async = true;
        ga.src = ('https:' == document.location.protocol ? 'https://ssl' : 'http://www') + '.google-analytics.com/ga.js';
        var s = document.getElementsByTagName('script')[0]; s.parentNode.insertBefore(ga, s);
      })();
    </script>
    <META HTTP-EQUIV="Refresh"
          CONTENT="5; URL=$url">

</head>

<body>
    <font size="14px" color="#aaa" face="Helvetica, Arial, sand-serif">
    <b>Your link has been posted!</b><br /><br />

    Redirecting back to <b>$url</b> in 5 seconds...
    </font>
</body>

</html>
);
    }
}
else {
    my $id = $ENV{'QUERY_STRING'};

    my $sth = $dbh->prepare( qq{
        UPDATE ircLink SET timestamp = timestamp, clicks = clicks + 1
        WHERE ircLinkID = ?
    } );

    my $q = $sth->execute( $id );

    my $clicks = $dbh->selectrow_array( qq{
        SELECT clicks FROM ircLink
        WHERE ircLinkID = '$id'
    } );

    print "id: [$id]\n";

    my $url = $dbh->selectrow_array( qq{
        SELECT url FROM ircLink
        WHERE ircLinkID = '$id'
    } );

    print "Location: $url\n\n";
}

