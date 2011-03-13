require 'rubygems'
require 'rack/reloader'
require 'tumble'


use Rack::Reloader, 0 if development?
run Sinatra::Application
