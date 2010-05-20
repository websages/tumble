#!/usr/local/bin/perl -w

use CGI;
use DBI;

use strict;

my $cgi = new CGI;
my $dbh = DBI->connect( 'dbi:mysql:tumble:172.16.0.2', 'nobody' );

if ( $cgi->param( 'quote' ) && $cgi->param( 'author' ) ) {
    my $quote  = $cgi->param( 'quote' );
    my $author = $cgi->param( 'author' );

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
