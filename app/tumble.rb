require 'sinatra'
require 'rest-client'
require 'json/pure'
require 'config/init'

Dir["models/**/*.rb"].each{|model|
  require model
}

set :haml, :format => :html5

t = Tumblelog.new('localhost', '5984', 'tumble')

get '/' do
  @quotes = [ { 'author' => 'foo', 'quote' => 'baz'}, { 'author' => 'foo', 'quote' => 'baz'}, { 'author' => 'foo', 'quote' => 'baz'}]
  @links =  [ { 'id' => 'e9e7e4e42e094db9b935450a12038c79','url' => 'http://www.google.com', 'author' => 'Aziz Shamim', 'title' => 'Google' }, { 'id' => 'e9e7e4e42e094db9b935450a12038c79','url' => 'http://www.google.com', 'author' => 'Aziz Shamim', 'title' => 'Google' } , { 'id' => 'e9e7e4e42e094db9b935450a12038c79','url' => 'http://www.google.com', 'author' => 'Aziz Shamim', 'title' => 'Google' }]
  haml :index
end

post '/quote' do
  request.body.rewind  # in case someone already read it
  data = { :quote => params[:quote], :author => params[:author], :type => 'quote' }
  resp = t.post(data)
  etag resp['rev'].to_s
  body "1"
  201
end

get '/quote/:id' do
  resp = t.find(params[:id])
  resp.to_json
end

post '/irclink' do
  request.body.rewind
  data = {:url => params[:url], :user => params[:user], :type => 'link' }
  resp = t.post(data)
  etag resp['rev'].to_s
  body resp['id'].to_s
  201
end

get '/irclink/:id' do
  resp = t.find(params[:id])
  headers['Location']= resp['url']
  headers['clicks']= resp['clicks']
  301
end

get %r{/page([\w]+)?} do
  number = ( params[:captures] ? params[:captures].first : 0 )
  resp = t.page(number)
  body resp.to_json
  200
end

