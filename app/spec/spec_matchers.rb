require 'rspec/expectations'

RSpec::Matchers.define :contain_next_nav do |expected|
  match do |actual|

