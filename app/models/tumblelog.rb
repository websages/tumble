require 'json/pure'
require 'rest-client'

class TumbleLog
  def initialize(uri)
    @uri = uri
    @limit = 10
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

  def page(number)
    resp = RestClient.get @uri+"/_design/items/_view/page?limit=#{@limit}&skip=#{number*@limit}"
    JSON.parse(resp.body)['rows'].collect { |r| r['value'] }.compact
  end

  private

  def link_title(url)
    resp = RestClient.head url
    if resp.code == 200 
      resp = RestClient.get url
      title = resp.match(/<title>([^<]*)/)[1]
    end
    title ? title : nil
  end

end
