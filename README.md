# Tumble setup

1. Get a flickr account.
1. Set flickr account in scripts/flickr (since this is not abstracted yet)
1. Change the passwords/usernames in sql_setup
1. Change the passwords/usernames in the config.yaml
1. Setup database

     yum install mysql-server
     service mysqld start
     chkconfig mysqld on
     mysql < sql_setup
     mysql -u tumble -p tumble < migrations


# Other Todo Items
Get a flickr account
hack kerminator to work on this
move logs into local directory
package this shit
make templatize configuration stuff to work on Apache 2.2 and 2.4?
Work on debian and Fedora?
Abstract hard-coded stuff into variables...even if global
migrations for database
create deployment methodology
Have it warn if the database has no tables or something
Fix /usr/local/bin/perl
fix user-agent being hardy


# Stuff to check
Buttons
Quotes
Images
Hubot
Flickr


# Package Requirements
  perl-mysql mysql-server
  mod_perl
  perl-CGI-Application
  perl-LWP-UserAgent-Determined
  httpd
  

