#!/usr/bin/env ruby

require 'rubygems'
require 'open-uri'
require 'net/http'
require 'mysql'
require 'time'
require 'date'
require 'base64'
require 'yaml'
require 'progressbar'
require File.join(File.dirname(__FILE__), "../lib", 'tumble.rb')

MYSQL   = {:ip=>'127.0.0.1', :username=>'nobody',:password=>nil, :database=>'tumble'}

def time_cleanup(time)
  Time.parse(DateTime.parse(time).to_s).utc.strftime("%a, %m %b %Y %H:%M:%S GMT").to_s
end

def import_quotes()
  # Main Program Flow
  query = SDB.query("SELECT timestamp, quote, author FROM quote")
  pbar = ProgressBar.new("Quotes", query.num_rows)
  query.each do |row|
    @quote = TumbleLog::Quote.new(
      :created_at => time_cleanup(row[0]),
      :quote      => row[1],
      :author     => row[2],
      :type       => 'link'
    )
    @quote.save
    pbar.inc
  end
  pbar.finish
end

def import_links()
  query = SDB.query("SELECT timestamp, user, title, url, clicks from ircLink")
  pbar = ProgressBar.new("Links", query.num_rows)
  query.each do |row|
    @link = TumbleLog::Link.new(
      :created_at => time_cleanup(row[0]),
      :user       => row[1],
      :title      => row[2],
      :url        => row[3],
      :clicks     => row[4],
      :type       => 'quote'
    )
    @link.save
    pbar.inc
  end
  pbar.finish
end

def cache_images() 
  dir = "/tmp/imgcache/"
  logfile = dir + 'error.log'
  Dir::mkdir(dir) unless FileTest.directory?(dir)
  
  log = File.open(logfile, 'w') 
    
  query = SDB.query("SELECT timestamp, title, link, url, md5sum, imageID FROM image")
  pbar = ProgressBar.new("Cache", query.num_rows)
  query.each do |row|
    value = {
      'timestamp' => row[0],
      'title'     => row[1],
      'link'      => row[2],
      'url'       => row[3],
      'md5sum'    => row[4],
      'imageID'   => row[5]
    }    
    # Try to not download the file again. 
    # Need to change to MD5 Sum
    #next unless (File.exist?("#{value['imageID']}.jpg") and )) 
    
    #if File.exist?("#{value['imageID']}.yml") do
      
    #end
    
    # Also don't grab dailykitten shit. 
    next unless value['url'].include? "flickr"
    
    begin
      File.open(dir + value['imageID'] + '.jpg', 'w') { |output| 
        open(value['url']) { |input| output.write(input.read) }
      }
      File.open(dir + value['imageID'] + '.yml', 'w') { |yaml|
        yaml.write(YAML::dump(value))
      }
    rescue => e
      log.write e.message + "\n" 
      log.write e.backtrace + "\n"
      log.write value['url'] + "\n"
    end
    pbar.inc
  end
  
  log.close 
end

def import_images()
  query = SDB.query("SELECT timestamp, title, link, url, md5sum FROM image")
  pbar = ProgressBar.new("Images", query.num_rows)
  query.each do |row|
    next unless row[3].include? "flickr"
    
    # download image from row[3]
    tmpfile = open(row[3])
    name = "test" #file[:filename]
    type = `file -Ib #{tmpfile.path}`.gsub(/\n/,"")
    length = tmpfile.length
    data = Base64.encode64(tmpfile.read)

    @image = TumbleLog::Image.new(
      :created_at => time_cleanup(row[0]),
      :type       => 'image',
      :attachments => { 
        "#{name}" => { 
          :content_type => type , 
          :length       => length, 
          :data         => data
        } 
      }
    )
    @image.save

    pbar.inc
  end
  pbar.finish
end


# Main Program Flow
SDB = Mysql.init
SDB.options(Mysql::SET_CHARSET_NAME, 'utf8')
SDB.real_connect(MYSQL[:ip], MYSQL[:username], MYSQL[:password], MYSQL[:database])
SDB.query("SET NAMES utf8")

#import_quotes()
#import_links()
#import_images()
cache_images()

SDB.close if SDB


