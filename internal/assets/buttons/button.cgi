#!/usr/bin/perl -w

use CGI;

use YAML qw( LoadFile );

use strict;

my $cgi = new CGI;

my $user = $cgi->param( 'user' );

my $config = LoadFile( '../config.yaml' );
my $url = $config->{'baseurl'};

if ( $user ) {
    print "Content-type: text/html\n\n";

    print qq(<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en">

<head>
    <title>tumblefish buttons</title>
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
    <link rel="stylesheet" href="http://$url/css/screen.css" type="text/css" media="screen" />
</head>

<body>
    <div id="page">
        <div id="masthead">
            tumblefish.
        </div>

        <div id="content">
            <div class="tumble_date">
            <div class="tumble_date_date">!!</div>
            <div class="tumble_date_mon">buttons</div>
            <div class="tumble_date_day">yay!</div>
        </div>
        <div class="tumble_item_quote">
            <div class="tumble_item_top"></div>
            <span class="tumble_item_quote_quote">So how do I install this crap??</span>
            <div class="tumble_item_bottom"></div>
        </div>
        <div class="tumble_item_ircLink">
            <div class="tumble_item_top"></div>
            <span class="tumble_item_ircLink_title">);
    print qq(Drag this link: <a href="javascript:location.href='http://$url/irclink/?user=$user&source=web&url='+encodeURIComponent(location.href)" onclick="window.alert('No clicky! Drag this link to your Bookmarks toolbar or menu, or right-click it and choose Bookmark This Link...');return false;">post to tumblefish!</a> up to your Bookmarks toolbar or menu.);
    print qq(</span>
            <div class="tumble_item_bottom"></div>
        <div>
            <div class="tumble_item_ircLink">
            <div class="tumble_item_top"></div>
            <span class="tumble_item_ircLink_title">PS - Unfortunately, tumblebuttons don't work with Microsoft Internet Explorer.  MSIE sucks.  Stop using it.</span>
            <div class="tumble_item_bottom"></div>
        </div>
        <div>
            <div class="tumble_item_ircLink">
            <div class="tumble_item_top"></div>
            <span class="tumble_item_ircLink_title">PPS - If your name is Greg Buchanan and you just read the above postscript, you can suck my ass.</span>
            <div class="tumble_item_bottom"></div>

        </div>
    </div>
</body>

</html>
);
}
else {
    print "Content-type: text/plain\n\n";
    print "Oh no!  You didn't enter your name!";
}
