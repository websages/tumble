# Migrations will run automatically. The DSL like wrapper syntax is courtesy
# of sinatra-sequel
#
# For details on sequel's schema modifications, check out:
# http://sequel.rubyforge.org/rdoc/files/doc/schema_rdoc.html
 
migration "create quotes table" do
  database.create_table :quote do
    primary_key :quoteID
    string :quote
    string :author
    datetime :timestamp, :default => 'CURRENT_TIMESTAMP'
  end
end

migration "create links table" do
  database.create_table :irclink do
    primary_key :ircLinkID
    string :user
    string :title
    text :url
    int :clicks
    datetime :timestamp, :default => 'CURRENT_TIMESTAMP'
  end
end

migration "create the image table" do
  database.create_table :image do
    primary_key :imageID
    datetime :timestamp, :default => 'CURRENT_TIMESTAMP'
    string :title
    string :link
    text :url
    text :md5sum
  end
end
    


