
configure :development do
  ENV['DATABASE_URL']='http://localhost:5984/tumble_dev'
end

configure :test do
  ENV['DATABASE_URL']='http://localhost:5984/tumble_test'
end

configure :production do
  ENV['DATABASE_URL']='http://localhost:5984/tumble'
end

# build the database
#require 'migrations/*.rb'

# add the models
Dir["models/**/*.rb"].each { |model| require model }
