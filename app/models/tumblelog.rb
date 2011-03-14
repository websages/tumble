require 'json/pure'
require 'rest-client'

class Tumblelog
  def initialize(host, port, database)
    @db = database 
    @host = host
    @port = port

    @uri = "http://#{@host}:#{@port}/#{@db}"
  end

  def post(data)
    data[:created_at] = Time.now.getutc.to_s
    if data[:url]
      data[:title] = link_title(data[:url]) || data[:url]
    end
    resp = RestClient.post @uri, data.to_json, :content_type => 'application/json'
    JSON.parse(resp)
  end

  def find(id)
    resp = RestClient.get @uri+"/#{id}?revs=true"
    data = JSON.parse(resp.body)
    if data['type'] == 'link'
      data[:timestamp] = Time.now.getutc.to_s
      resp = RestClient.put @uri+"/#{id}", data.to_json, :content_type => 'application/json'
      data[:clicks]=data['_revisions']['ids'].count.to_s
    end
    JSON.parse(data.to_json)
  end

  def page(number=0)
    resp = RestClient.get @uri+"/_design/items/_view/page?limit=10&skip=#{number}"
    JSON.parse(resp.body)
  end

  def request(req)
    res = Net::HTTP.start(@host, @port) { |http|http.request(req) }
    unless res.kind_of?(Net::HTTPSuccess)
      handle_error(req, res)
    end
    res
  end

  private

  def link_title(url)
    #resp = Net::HTTP.start(@host, @port) { |http| http.get(url) }
    resp = RestClient.head url
    if resp.code == 200 
      resp = RestClient.get url
      title = resp.match(/<title>([^<]*)/)[1]
    end
    title ? title : nil
  end

  def handle_error(req, res)
    e = RuntimeError.new("#{res.code}:#{res.message}\nMETHOD:#{req.method}\nURI:#{req.path}\n#{res.body}")
    raise e
  end

end
