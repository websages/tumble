describe 'TumbleLog' do

  include Rack::Test::Methods
  require 'rspec/expectations'

  def app
    @app ||= Sinatra::Application
  end

  # allow database
  FakeWeb.allow_net_connect = %r[^https?://localhost:5984]

  let(:quoteid) { 'd150735ece7927beb570b830bd00414d' }
  let(:linkid)  { 'd150735ece7927beb570b830bd00456a' }
  let(:imageid) { '0f7dcd57dbde61a4bf6775babe030721' }

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
#       it 'should be backwards compatible with the original cgi (specification)'
  end

#      describe 'setup' do
#        it 'should create a database if none is available'
#        it 'should install the design docs into the database'
#      end

end
