require 'sinatra'
require 'rest-client'
require 'json/pure'
require 'config/init'
require 'base64'

set :haml, :format => :html5

t = TumbleLog.new(ENV['DATABASE_URL'])

get '/' do
  number = 0
  @items = t.page(0)
  number == 0 ? @nextpage = nil : @nextpage = number.to_i - 1
  @prevpage = number.to_i + 1
  haml :index
end

post '/quote' do
  request.body.rewind  # in case someone already read it
  data = { :quote => params[:quote], :author => params[:author] }
  resp = t.post_quote(data)
  headers['Location']="/quote/#{resp['id']}"
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
  data = {:url => params[:url], :user => params[:user] }
  resp = t.post_link(data)
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

get %r{/page/?([\d]+)?} do
  if params[:captures]
    number = params[:captures].first
  else
    number = 0
  end
  resp = t.page(number)
  body resp.to_json
  200
end

get '/:number' do
  number = params[:number] || 0
  @items = t.page(number)
  number == 0 ? @nextpage = nil : @nextpage = number.to_i - 1
  @prevpage = number.to_i + 1
  haml :index
end

post '/image' do
  file = params['file']
  resp = t.post_image(file)

 etag resp['rev'].to_s
  body resp['id'].to_s
  headers['Location']="/image/#{resp['id']}"
  201
end

