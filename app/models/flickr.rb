require 'nokogiri'

class Flickr
  attr_accessor :user_id

  def initialize(userid, key, secret, database_uri)
    @user_id =userid
    @api_key = key
    @api_secret = secret
    @database = database_uri
  end

  def photos(number)
    pages = 1
    if number > 500
      pages = ( number / 500 ) + 1
      number = 500
    end
    method='flickr.people.getPublicPhotos'
    options = {'extras' => "description,date_upload,url_z,", 'page' => pages, 'per_page' => number}
    resp = JSON.parse(flickr(method,options))
    resp['photos']['photo'].collect {|p| 
        { :flickrid => p['id'], :url_z => p['url_z'], :created_at => p['date_upload'], :title => p['title'], :description => p['description'] }
      }.compact
  end

  ##check if the data is newer than when we last saw it
  def lastseen
    res = RestClient.head feeduri
    time = res.headers[:last_modified]
    Time.httpdate(time)
  end

  def total
    res = JSON.parse(flickr('flickr.people.getInfo', {} ))
    res['person']['photos']['count']['_content']
  end

  def new?
    flickrdata = RestClient.get @database+'/flickr'
  end


  private
  def flickr(method, options)
    uri = "http://api.flickr.com/services/rest/?method=#{method}&api_key=#{@api_key}&user_id=#{user_id}&format=json&nojsoncallback=1"
    options.each do |k,v| 
      uri += "&#{k}=#{v}"
    end
    RestClient.get uri
  end

  def feeduri
    "http://api.flickr.com/services/feeds/photos_public.gne?id=#{@user_id}"
  end

  def store(data)
    # store the photo data in the database
  end

  def get(date_from, date_to)
    # find the dates for the photos
  end

end
