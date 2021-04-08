require 'spec_helper'
require 'thrift/types/known/duration'

describe 'Thrift::Types::Known::Duration' do
  context 'from_number' do
    it 'from float' do
      d = Thrift::Types::Known::Duration.from_number(2.0005)

      d.seconds.should == 2
      d.nanos.should == 500_000
    end

    it 'from int' do
      d = Thrift::Types::Known::Duration.from_number(127)

      d.seconds.should == 127
      d.nanos.should == 0
    end
  end
end
