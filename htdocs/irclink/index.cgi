#!/usr/local/bin/perl -w

use CGI;
use DBI;
use LWP::UserAgent;

use strict;

my $cgi = new CGI;
my $dbh = DBI->connect( 'dbi:mysql:tumble', 'nobody' );

if ( $cgi->param( 'user' ) && $cgi->param( 'url' ) ) {
    my $user = $cgi->param( 'user' );
    my $url  = $cgi->param( 'url' );

    my $agent = LWP::UserAgent->new();
    $agent->agent( 'Mozilla/5.0' );

    unless ( $agent->head( $url )->{'_rc'} eq '200' ) {
        print "Content-type: text/plain\n\n";
        print '0';
        exit( 0 );
    }

    my $title = $agent->get( $url )->title();

    $title ||= $url;

    my $sth = $dbh->prepare( qq{
        INSERT INTO ircLink (
            user, title, url
        ) VALUES (
            ?, ?, ?
        )
    } );

    $sth->execute( $user, $title, $url );

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

    <script src="http://www.google-analytics.com/urchin.js" type="text/javascript">
    </script>
    <script type="text/javascript">
        _uacct = "UA-161892-2";
        urchinTracker();
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

