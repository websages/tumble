describe 'TumbleLog' do
  include Rack::Test::Methods

  def app
    @app ||= Sinatra::Application
  end

  # allow database
  FakeWeb.allow_net_connect = %r[^https?://localhost:5984]

  let(:uri)     { ENV['DATABASE_URL'] }

  # setup
  before(:all) do
    @db = TumbleLog::Setup.new(uri)
    @db.create!
    @db.install!
    setup_test_environment(30)
  end

  after(:all) do
    @db.destroy!
  end


# it 'should be backwards compatible with the original cgi (specification)'
  describe 'frontpage' do
    it 'should load the tracker for google analytics' do
      get '/'
      last_response.should be_ok
    end
#    it 'should allow a user to page forward and backward'
#    it 'should be able to search links, quotes, and photos'
    describe '_items' do
      matcher :include_div_class do |expected|
        match do |actual|
          b = Nokogiri::HTML::Document.parse(actual)
          b.root.xpath(".//div[@class='#{expected}']").length > 0
        end
      end

      before(:all) {
        get '/'
        @body = last_response.body
      }
      it { @body.should include_div_class('quote') } 
      it { @body.should include_div_class('link') } 
      it { @body.should include_div_class('image') } 
    end
  end
end
