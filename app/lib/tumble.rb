# tumble.rb

require 'rubygems'
require 'couchrest'
require 'couchrest_model'

module TumbleLog
  COUCH = CouchRest.database!(ENV['DATABASE_URL'])
    
  class Link < CouchRest::Model::Base
    use_database COUCH
  
    property :created_at, :default => Time.new.utc.strftime("%a, %m %b %Y %H:%M:%S GMT")
    property :title
    property :url
    property :user
    property :clicks
  end

  class Quote < CouchRest::Model::Base
    use_database COUCH
 
    property :created_at, :default => Time.new.utc.strftime("%a, %m %b %Y %H:%M:%S GMT")
    property :quote
    property :author
  end
  
  class Image < CouchRest::Model::Base
    use_database COUCH
  
    property :created_at, :default => Time.new.utc.strftime("%a, %m %b %Y %H:%M:%S GMT")
    property :attachments
    property :sha256sum
  end
end