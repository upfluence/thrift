require 'spec_helper'
require 'thrift/types/known/timestamp/timestamp'

describe 'Thrift::Types::Known::Timestamp' do
  context 'from_time' do
    it do
      t = Time.at(137, 5, :nsec)

      tt = Thrift::Types::Known::Timestamp.from_time(t)

      tt.seconds.should == 137
      tt.nanos.should == 5

      tt.to_time.should == t
    end
  end
end
