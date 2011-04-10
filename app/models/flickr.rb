class Flickr
  require 'time'
  require 'json/pure'
  attr_accessor :user_id

  #httptime is
  # Time.new.utc.strftime("%a, %m %b %Y %H:%M:%S GMT")

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

  def last_modified
    res = RestClient.head feeduri
    time = res.headers[:last_modified]
    Time.httpdate(time)
  end

  def last_update
    begin
      RestClient.get(@database+'/flickr') { |response, request, result|
        time = JSON.parse(response)['last_update']
        Time.httpdate(time)
      }
    end
  end

  def total
    res = JSON.parse(flickr('flickr.people.getInfo', {} ))
    res['person']['photos']['count']['_content']
  end

  def update_db
    data = {'user_id' => @user_id, 'stats' => {'last_update' => Time.now.httpdate } }
    RestClient.put @database+'/flickr', data, :content_type => 'application/json'
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
    RestClient.post(@database, data, :content_type => 'application/json') { |response, request, result|
      # something something
    }
  end

  def get(date_from, date_to)
    # find the dates for the photos
  end

end
