describe TumbleLog::Setup do

  let(:uri) { ENV['DATABASE_URL'] }
  let(:all_design_docs) { '/_all_docs?startkey=%22_design/%22&endkey=%22_design0%22&include_docs=true' }

  matcher :have_database do
    match do |database|
      RestClient.get(database) do |response, request, result, &block|
        response.code == 200
      end
    end
  end

  matcher :have_design_docs do
    match do |database|
      query = database + '/_all_docs?startkey=%22_design/%22&endkey=%22_design0%22&include_docs=true'
      response = RestClient.get query
      JSON.parse(response)['total_rows'] > 0
    end
  end

  before(:each) do
    @db = TumbleLog::Setup.new(uri)
  end

  after(:each) do
    RestClient.delete uri
  end

  describe '_setup' do
    it 'should create a database if none is available' do
      uri.should_not have_database
      @db.create!
      uri.should have_database
    end

    it 'should install the design docs into the database' do
      @db.create!
      uri.should_not have_design_docs
      @db.install!
      uri.should have_design_docs
    end
  end

end
