#!/usr/bin/perl -w

BEGIN { unshift @INC, '../lib'; }

use tumble::DB;

use CGI;
use DBI;
use URI::Escape;

use strict;

my $cgi = new CGI;
my $dbh = tumble::DB->new( config => '../config.yaml' );

if ( $cgi->param( 'quote' ) && $cgi->param( 'author' ) ) {
    my $quote  = $cgi->param( 'quote' );
    my $author = $cgi->param( 'author' );

    $quote=uri_unescape($quote);
    $author=uri_unescape($author);

    my $sth = $dbh->prepare( qq{
        INSERT INTO quote (
            quote, author
        ) VALUES (
            ?, ?
        )
    } );

    $sth->execute( $quote, $author );

    print "Content-type: text/plain\n\n";
    print "1";
}
