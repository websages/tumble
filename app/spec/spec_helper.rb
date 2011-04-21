require 'rubygems'
require File.join(File.dirname(__FILE__), "../", 'tumble.rb')

require 'rack/test'
require 'ruby-debug'
require 'rspec'
require 'rspec/expectations'
require 'fakeweb'
require 'nokogiri'
require 'libxml'

require File.join(File.dirname(__FILE__), "../", 'lib/setup.rb')

#require File.join(File.dirname(__FILE__), '/spec_matchers.rb'

def post_quote
  post '/quote', { :author => 'Yoda', :quote => 'Do or do not, there is no try'}
  @quoteid = last_response.body
end

def post_image
  post '/image', 'file' => Rack::Test::UploadedFile.new('spec/fixtures/placekitten.jpg', 'image/jpeg')
  @imageid = last_response.body
end

def post_link
  testuri = 'http://www.google.com'
  FakeWeb.register_uri(:head, testuri, :status => [200, "OK"])
  FakeWeb.register_uri(:get, testuri, :status => [200, "OK"], :body => 'spec/fixtures/google.html')
  post "/irclink", {:user => 'Aziz Shamim', :url => testuri }
  @linkid = last_response.body
end

def setup_test_environment(n)
  n.times do
    post_quote
    post_image
    post_link
  end
end



# set the test environment
set :environment, :bintest
set :run, false
set :raise_errors, true
set :logging, false

