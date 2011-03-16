describe Flickr do
  include Rack::Test::Methods

  FakeWeb.allow_net_connect = %r[^https?://localhost:5984]

  let(:key) {'e989f09213d48f7be061563511fd107e'}
  let(:secret) {'d31541f61b737aa3'}


  describe '_photos' do
    FakeWeb.register_uri(:get,"http://api.flickr.com/services/rest/?method=flickr.people.getPublicPhotos&api_key=e989f09213d48f7be061563511fd107e&user_id=30378931@N00&extras=description,date_upload,url_z,geo" ,:body => 'spec/fixtures/flickr_photos.xml', :status => [200, "OK"])
                         
    it 'should store the last time it updated to the database'
#       f = Flickr.new(key,secret)
#       f.last_seen.should_not be_nil
#     end

    it 'should get a list of photos from flickr' do
      f = Flickr.new(key,secret)
      f.photos(10).should_not be_empty
      #f.photos(10).should be_an(Array)
    end

#       f = Flickr.new(key,secret)
#       f.photos(10)
#     end

    #it 'should respond with a list of photos' do
    #  f =Flickr.new(key,secret)
    #  p = f.photos(10)
    #  p.should_not be_nil
    #  p.should_not be_empty
    #end
  end
end
