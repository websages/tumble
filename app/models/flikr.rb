class Flickr
  attr_accessor :user_id

  def initialize(key, secret)
    @user_id ='30378931@N00'
    @api_key = key
    @api_secret = secret
  end

  def photos(number)
    method='flickr.people.getPublicPhotos'
    options = {'extras' => "description,date_upload,url_z,geo" }
    limits=""
    resp = flickr(method,options)
  end


  #return the feed
  # flickr feed 
  # http://api.flickr.com/services/feeds/photos_public.gne?id=#{@user_id}
  def feed
    RestClient.get uri
  end

  #return link data for a uri
  def photolink(id)
  end

  ##check if the data is newer than when we last saw it
  #def new?
  #  res = RestClient.head uri
  #  last_modified = Time.httpdate(res.headers[:last_modified])
  #  last_modified <=> last_seen
  #end

  #def last_seen
  #  r = RestClient.get ENV['DATABASE_URL']+'/flickr'
  #  resp =  JSON.parse(r)
  #  Time.httpdate(resp['last_seen'])
  #end

  ##check if it is the first time
  #def first?
  #  r = RestClient.get ENV['DATABASE_URL']+'/flickr'
  #  p r
  #  if r.status == 404
  #end

  private
  def flickr(method, options)
    uri = "http://api.flickr.com/services/rest/?method=#{method}&api_key=#{@api_key}&user_id=#{user_id}"
    options.each do |k,v| 
      uri += "&#{k}=#{v}"
    end
    RestClient.get uri
  end

  def feeduri
    "http://api.flickr.com/services/feeds/photos_public.gne?id=#{@user_id}"
  end

end
