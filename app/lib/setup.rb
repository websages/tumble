class TumbleLog::Setup
  require 'uri'

  attr_reader :db, :name, :host

  def initialize(uri)
    @db = URI::parse(uri)
  end

  def create!
    unless exists?
      RestClient.put @db.to_s, ''
    end
  end

  def destroy!
    unless !exists?
      RestClient.delete @db.to_s
    end
  end

  def install!
    resp = Array.new
    if exists?
      Dir['config/designs/*.json'].each do |design|
        doc = YAML::load(File.new(design))
        resp << RestClient.post(@db.to_s, doc.to_json, :content_type => :json, :accept => :json)
      end
    end
    resp
  end

  private
  def exists?
    RestClient.get(@db.to_s) do |response, request, result, &block|
      case response.code
      when 200
        true
      else
        false
      end
    end
  end

end
