#describe Flickr do
#  include Rack::Test::Methods
#
#  FakeWeb.allow_net_connect = %r[^https?://localhost:5984]
#
#  let(:key) {'e989f09213d48f7be061563511fd107e'}
#  let(:secret) {'d31541f61b737aa3'}
#  let(:user_id) { '30378931@N00' }
#  let(:database) { 'http://localhost:5984/tumble/'}
#
#  before(:all) do
#    @f = Flickr.new(user_id, key, secret, database )
#  end
#
#  describe '_photos' do
#    it 'should return the last update date' do
#      FakeWeb.register_uri(:any,'http://api.flickr.com/services/feeds/photos_public.gne?id=30378931@N00', {'Last-Modified' => "Fri, 18 Mar 2011 00:32:08 GMT", :status => [200, "OK"]})
#      date = @f.last_update
#      date.should be_a(Time)
#    end
#
#    it 'should get a list of photos from flickr' do
#      FakeWeb.register_uri(:get,"http://api.flickr.com/services/rest/?method=flickr.people.getPublicPhotos&api_key=e989f09213d48f7be061563511fd107e&user_id=#{user_id}&format=json&nojsoncallback=1&page=1&extras=description,date_upload,url_z,&per_page=10" ,:body => 'spec/fixtures/flickr_photos.json', :status => [200, "OK"])
#      @f.photos(10).should have(10).items
#      @f.photos(10).should be_an(Array)
#    end
#
#    it 'should tell you how many photos are available to slurping' do
#      FakeWeb.register_uri(:get, "http://api.flickr.com/services/rest/?method=flickr.people.getInfo&api_key=e989f09213d48f7be061563511fd107e&user_id=#{user_id}&format=json&nojsoncallback=1", :body => 'spec/fixtures/flickr_user.json')
#      @f.total.should_not be_nil
#      @f.total.should be_an(Integer)
#    end
#  end
#end
