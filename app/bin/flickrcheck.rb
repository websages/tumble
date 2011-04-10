#!/usr/bin/env ruby

require 'rubygems'
require 'rest-client'

require 'yaml'
require File.join(File.dirname(__FILE__), '/../', 'models/flickr.rb')

auth = YAML.load_file(File.join(File.dirname(__FILE__), '../config/flickrauth.secret'))

f = Flickr.new(auth['flickr']['user_id'], auth['flickr']['API_KEY'], auth['flickr']['API_SECRET'], 'http://localhost:5984/tumble')

# get the last flickr update time from the database
#http://api.flickr.com/services/feeds/photos_public.gne?id=30378931@N00&format=json
p f.last_update

#
# get the head time from the flickr rss feed
p f.last_modified
# if there is an update, get the RSS feed and update the database with the links
