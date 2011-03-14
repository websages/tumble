
configure :development do
  set :database, 'sqlite://tmp/development.sqlite'
end

configure :test do
  set :database, "sqlite::memory:"
end

# build the database
#require 'migrations/*.rb'

# add the models
Dir["models/**/*.rb"].each{|model|
  require model
}
