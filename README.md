# Tumble

## History

Tumble was a "Wouldn't it be cool?" project handed to Scott Schnedier (https://github.com/sschneid) back in 2004. The idea was to create a website similar to a tumbleblog. Obviously, eventually tumblr ccame along and the rest was history.

## Deployment

The easiet way to deploy is type is to clone and type `make rpm` on an EL6 system. Things should justwork after that.

If you are not on EL, things should still work. Just `make install` or package it yourself.


## Tumble setup

1. Get a flickr account.
1. Set flickr account in scripts/flickr (since this is not abstracted yet)
1. Change the passwords/usernames in sql_setup
1. Change the passwords/usernames in the config.yaml
1. Change url, server configuration etc in /etc/httpd/conf.d/tumble.conf
1. Disable selinux or set proper context
1. Start up httpd
1. Setup database

```
     yum install mysql-server
     service mysqld start
     chkconfig mysqld on
     mysql < sql_setup
     mysql -u tumble tumble < migrations
```

## Bugs

    * fix user-agent being hardy for link verification
    * Should warn if unable to talk to databse or database is empty
    * abstract quantity of items to be in 'hot shit' category
    * Fix odd encoding bugs for web site titles
    * Probably lots of others, but it has been in production for 10 years.
