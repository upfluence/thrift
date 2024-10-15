require 'spec_helper'
require 'json'
require 'thrift/types/value/value'

class LocalObject
  def initialize(foo, bar)
    @foo, @bar = foo, bar
  end
end

describe 'Thrift::Types::Value' do
  describe 'from_object' do
    subject do
      Thrift::Serializer.new(
        Thrift::JsonProtocolFactory.new
      ).serialize(Thrift::Types::Value.from_object(obj))
    end

    describe 'utf-8 string' do
      let(:obj) { 'foo' }

      it do
        subject.should == "{\"2\":{\"str\":\"foo\"}}"
      end
    end

    describe 'complex hash' do
      let(:obj) { { foo: true, baz: 0.5, buz: 1 } }

      it do
        subject.should == "{\"8\":{\"rec\":{\"1\":{\"lst\":[\"rec\",3,{\"1\":{\"rec\":{\"2\":{\"str\":\"foo\"}}},\"2\":{\"rec\":{\"6\":{\"tf\":1}}}},{\"1\":{\"rec\":{\"2\":{\"str\":\"baz\"}}},\"2\":{\"rec\":{\"5\":{\"dbl\":0.5}}}},{\"1\":{\"rec\":{\"2\":{\"str\":\"buz\"}}},\"2\":{\"rec\":{\"4\":{\"i64\":1}}}}]}}}}"
      end
    end

    describe 'object' do
      let(:obj) { LocalObject.new(1, true) }

      it do
        subject.should == "{\"9\":{\"rec\":{\"1\":{\"map\":[\"str\",\"rec\",2,{\"foo\":{\"4\":{\"i64\":1}},\"bar\":{\"6\":{\"tf\":1}}}]}}}}"
      end
    end
  end
end
