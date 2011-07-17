#!/usr/bin/env ruby

require 'rubygems'
require 'open-uri'
require 'net/http'
require 'mysql'
require 'time'
require 'date'
require 'base64'
require 'yaml'
require 'digest'
require 'progressbar'
require File.join(File.dirname(__FILE__), "../lib", 'tumble.rb')

MYSQL   = {:ip=>'127.0.0.1', :username=>'nobody',:password=>nil, :database=>'tumble'}

def time_cleanup(time)
  Time.parse(DateTime.parse(time).to_s).utc.strftime("%a, %m %b %Y %H:%M:%S GMT").to_s
end

def init_database()
  
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

def cache_images(dir = '/tmp/imgcache/')
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
    cache = dir + value['imageID'] + '.jpg'
    
    # Also don't grab dailykitten shit. 
    skip = true unless value['url'].include? "flickr"
    
    # Don't download the same file twice, damn it. 
    skip = true if File.exists?(cache) && File.size(cache) == get_content_length(value['url']) 

    if skip != true
      begin
        open(value['url']) { |input|
          value['content-type'] = input.content_type
          File.open(cache, 'w') { |output| 
            output.write(input.read)
          } 
        }
        
        value['sha256sum'] = generate_sha256_sum(cache)
        value['filename'] = cache

        File.open(dir + value['imageID'] + '.yml', 'w') { |yaml|
          yaml.write(YAML::dump(value))
        }
      rescue => e
        log.write e.message + "\n" 
        log.write e.backtrace + "\n"
        log.write value['url'] + "\n"
      end
    end
    pbar.inc
  end
  
  pbar.finish
  log.close 
end

def import_images(dir = '/tmp/imgcache')
  pbar = ProgressBar.new("Images", Dir.glob(dir+'/*.yml').size)
  
  Dir.glob(dir+'/*.yml') do |entry|
    yml = YAML::load(File.open(entry))
    
    @image = TumbleLog::Image.new(
      :created_at  => time_cleanup(yml['timestamp']),
      :type        => 'image',
      :sha256sum   => yml['sha256sum'],
      :attachments => {
        "#{yml['sha256sum']}.jpg" => {
          :content_type => yml['content-type'],
          :length       => File.size(yml['filename']),
          :data         => Base64.encode64(File.read(yml['filename'])),
        }
      }
    )
    @image.save
    pbar.inc
  end
  pbar.finish
end

def get_content_length(uri_str)
  response = nil

  url = URI.parse(uri_str)
  res = Net::HTTP.start(url.host, url.port) do |http|
    response = http.head(uri_str)
  end

  case response
    when Net::HTTPSuccess     then response['content-length'].to_i
    when Net::HTTPRedirection then get_content_length(response['location'])
  end
end

def generate_sha256_sum(file)
  Digest::SHA2.new(256).file(file).hexdigest
end

# Main Program Flow
SDB = Mysql.init
SDB.options(Mysql::SET_CHARSET_NAME, 'utf8')
SDB.real_connect(MYSQL[:ip], MYSQL[:username], MYSQL[:password], MYSQL[:database])
SDB.query("SET NAMES utf8")

cache_images()
import_quotes()
import_links()
import_images()


SDB.close if SDB


