describe 'TumbleLog' do
  include Rack::Test::Methods

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

    it 'should allow a user to page forward and backward' do
      get '/'
      last_response.should be_ok
      #last_response.body.should contain_prev_nav
      #last_response.body.should_not contain_next_nav
    end

#       it 'should include quotes from the database'
#       it 'should include links from the database'
#       it 'should include fotos from phlicker'
#       it 'should be able to search links, quotes, and photos'
#       it 'should be backwards compatible with the original cgi (specification)'
  end

#      describe 'setup' do
#        it 'should create a database if none is available'
#        it 'should install the design docs into the database'
#      end

end
