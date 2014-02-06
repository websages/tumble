# Tumble setup

1. Get a flickr account.
1. Set flickr account in scripts/flickr (since this is not abstracted yet)
1. Change the passwords/usernames in sql_setup
1. Change the passwords/usernames in the config.yaml
1. Change url, server configuration etc in /etc/httpd/conf.d/tumble.conf
1. Disable selinux or set proper context
1. Start up httpd
1. Setup database

     yum install mysql-server
     service mysqld start
     chkconfig mysqld on
     mysql < sql_setup
     mysql -u tumble tumble < migrations




# Todo Items
Get a flickr account
hack kerminator to work on this
move logs into local directory
make templatize configuration stuff to work on Apache 2.2 and 2.4?
create deployment methodology

# Bugs

    * fix user-agent being hardy
    * Should warn if unable to talk to databse or database is empty
    * Abstract hard-coded stuff into variables...even if global

# Stuff to check
Buttons
Quotes
Images
Hubot
Flickr


# Package Requirements
See tumble.spec file
  

