#!/usr/bin/env perl

use strict;
use warnings;
use Test::More tests => 8;
use FindBin;
use lib "$FindBin::Bin/../htdocs/lib";

# Test 1: Load the DB module
BEGIN { use_ok('tumble::DB') }

# Test 2: Load MySQL driver
BEGIN { use_ok('tumble::DB::MySQL') }

# Test 3: Load SQLite driver
BEGIN { use_ok('tumble::DB::SQLite') }

# Test 4: Load Base class
BEGIN { use_ok('tumble::DB::Base') }

# Test 5: MySQL date_interval_sql
{
    my $mysql = bless { dbi => undef }, 'tumble::DB::MySQL';
    my $sql = $mysql->date_interval_sql(6);
    is($sql, "DATE_SUB(CURDATE(), INTERVAL 6 DAY)", "MySQL date_interval_sql generates correct SQL");
}

# Test 6: SQLite date_interval_sql
{
    my $sqlite = bless { dbi => undef }, 'tumble::DB::SQLite';
    my $sql = $sqlite->date_interval_sql(6);
    is($sql, "date('now', '-6 days')", "SQLite date_interval_sql generates correct SQL");
}

# Test 7: MySQL fulltext_search_sql (mock DBI quote)
{
    package MockDBI;
    sub quote { my ($self, $str) = @_; return "'$str'"; }
    
    package main;
    my $mysql = bless { dbi => bless({}, 'MockDBI') }, 'tumble::DB::MySQL';
    my $sql = $mysql->fulltext_search_sql('title,url', 'test query');
    is($sql, "MATCH (title,url) AGAINST ('test query')", "MySQL fulltext_search_sql generates correct SQL");
}

# Test 8: SQLite fulltext_search_sql (mock DBI quote)
{
    package MockDBI2;
    sub quote { my ($self, $str) = @_; return "'$str'"; }
    
    package main;
    my $sqlite = bless { dbi => bless({}, 'MockDBI2') }, 'tumble::DB::SQLite';
    my $sql = $sqlite->fulltext_search_sql('title,url', 'test');
    like($sql, qr/title LIKE '%test%' OR url LIKE '%test%'/, "SQLite fulltext_search_sql generates LIKE-based SQL");
}

done_testing();
