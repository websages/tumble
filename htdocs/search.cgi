#!/usr/local/bin/perl -wT

BEGIN { unshift @INC, './lib'; } 

use strict;

eval {
    require tumble::search;

    my $search = tumble::search->new(
        tmpl_path => 'thtml/'
    );

    $search->run();
};

print $@ if $@;

