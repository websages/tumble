require 'json/pure'
require 'rest-client'
require 'base64'

class TumbleLog
  def initialize(uri)
    @uri = uri
    @limit = 10
  end

  def post_quote(doc)
    doc['type'] = 'quote'

    resp = post(doc)
    JSON.parse(resp)
  end

  def image(ref)
    if ref.class == String
      resp = find(ref)
      data = resp['attachments']
      name = data.keys.first

      data[name]['name'] = name
      data[name]['_id'] = data['_id']
      data[name]['_rev'] = data['_rev']
      data[name]['data'] = Base64.decode64(data[name]['data'])
      return data[name]
    else
      resp = post_image(ref)
      return resp
    end
  end

  def post_link(doc)
    doc[:title] = link_title(doc[:url]) || doc[:url]
    doc[:clicks] = 0
    doc['type'] = 'link'

    resp = post(doc)
    JSON.parse(resp)
  end

  def post_image(file)
    tmpfile = file[:tempfile]
    name = file[:filename]
    type = file[:type]
    length = tmpfile.length
    data = Base64.encode64(tmpfile.read)
    attachments = { "#{name}" => { :content_type => type , :length => length, :data => data } }

    doc = Hash.new
    doc['type'] = 'image'
    doc[:attachments] = attachments

    resp = post(doc)
    JSON.parse(resp)
  end

  def find(id)
    resp = RestClient.get @uri+"/#{id}"
    data = JSON.parse(resp.body)
    if data['type'] == 'link'
      data[:timestamp] = Time.now.getutc.to_s
      clicks = data['clicks'].to_i + 1
      data['clicks'] = clicks
      resp = RestClient.put @uri+"/#{id}", data.to_json, :content_type => 'application/json'
      data['clicks'] = clicks.to_s
    end
    JSON.parse(data.to_json)
  end

  def page(number)
    skip = number.to_i * @limit
    resp = RestClient.get @uri+"/_design/items/_view/page?limit=#{@limit}&skip=#{skip}"
    JSON.parse(resp.body)['rows'].collect { |r| r['value'] }.compact
  end

  private
  def post(doc)
    doc[:created_at] = Time.now.getutc.to_s
    resp = RestClient.post @uri, doc.to_json, :content_type => 'application/json'
    resp
  end

  def link_title(url)
    resp = RestClient.head url
    if resp.code == 200 
      resp = RestClient.get url
      title = resp.match(/<title>([^<]*)/)[1]
    end
    title ? title : nil
  end

  def shortcode(longcode)
    Base64.encode64(longcode)
  end

  def longcode(shortcode)
    Base64.decode64(shortcode)
  end

end
