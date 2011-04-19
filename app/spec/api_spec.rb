describe 'TumbleLog' do
  include Rack::Test::Methods

  def app
    @app ||= Sinatra::Application
  end
  # allow database
  FakeWeb.allow_net_connect = %r[^https?://localhost:5984]

  before(:all) do
    # get the database ready for the api tests
  end

  after(:all) do
    # database teardown
  end

  describe 'api' do
    let(:quoteid) { '46fe6a76d6688715649a7f5dce006a3c' }
    let(:linkid)  { '46fe6a76d6688715649a7f5dce006d8a' }
    let(:imageid) { '46fe6a76d6688715649a7f5dce00798d' }

    describe '_page' do
      it 'should return 10 items at a time' do
        get '/page'
        last_response.should be_ok
        j = JSON.parse(last_response.body)
        j.should be_an(Array)
        j.should have(10).items
      end
      it 'should skip items' do
        get '/page/0'
        list1 = JSON.parse(last_response.body).collect {|i| i['_id']}.compact
        get '/page/2'
        list2 = JSON.parse(last_response.body).collect {|i| i['_id']}.compact
        diff=list1&list2
        diff.should have(0).items
      end
    end

    describe '_quote' do
      # 'http://tumble.wcyd.org/quote/?quote=' . "$quote" . "&author=$author")
      it 'should accept a quote post' do
        post '/quote', { :author => 'Yoda', :quote => 'Do or do not, there is no try'}
        last_response.status.should eql(201)
        last_response.headers['ETag'].should_not be_nil
      end
      it 'should retrieve a quote' do
        get "/quote/#{quoteid}"
        last_response.should be_ok
        JSON.parse(last_response.body).should be_a(Hash)
      end
      it 'should return a 404 on a bad quote' do
        get '/quote/919191919'
        last_response.status.should eql(404)
      end
    end

    describe '_irclink' do
      it 'should accept a link' do
        testuri = 'http://www.google.com'
        FakeWeb.register_uri(:head, testuri, :status => [200, "OK"])
        FakeWeb.register_uri(:get, testuri, :status => [200, "OK"], :body => 'spec/fixtures/google.html')
        post "/irclink", {:user => 'Aziz Shamim', :url => testuri }
        last_response.status.should eql(201)
        last_response.headers['ETag'].should_not be_nil
        last_response.body.should_not be_empty
      end
      it 'should redirect to the link' do
        get "/irclink/#{linkid}"
        last_response.status.should eql(301)
        last_response.headers['Location'].should_not be_nil
      end
      it 'should track how many times the link is clicked (redirected)' do
        get "/irclink/#{linkid}"
        pre_click = last_response.headers['clicks'].to_i
        get "/irclink/#{linkid}"
        post_click = last_response.headers['clicks'].to_i
        diff = post_click - pre_click
        diff.should eql(1)
      end
    end

    describe '_photos' do
      it 'should accept a photo' do
        post '/image', 'file' => Rack::Test::UploadedFile.new('spec/fixtures/placekitten.jpg', 'image/jpeg')
        last_response.status.should eql(201)
        last_response.headers['ETag'].should_not be_nil
        last_response.headers['Location'].should_not be_nil
      end
      it 'should retrieve a photo' do
        get "/image/#{imageid}"
        last_response.status.should eql(200)
        last_response.headers['ETag'].should_not be_nil
        last_response.body.should_not be_empty
      end
#         it 'should include metadata culled from exif information'
#         it 'should include a caption'
    end
  end
end
