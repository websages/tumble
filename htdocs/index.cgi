#!/usr/local/bin/perl -wT

BEGIN { unshift @INC, './lib'; } 

use strict;

eval {
    require tumble;

    my $tumble = tumble->new(
        tmpl_path => 'thtml/'
    );

    $tumble->run();
};

print $@ if $@;

