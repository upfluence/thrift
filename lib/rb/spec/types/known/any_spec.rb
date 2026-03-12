require 'spec_helper'
require 'thrift/types/value/value'
require 'thrift/types/known/any/any'

describe 'Thrift::Types::Known::Any' do
  describe 'encode' do
    let(:codec) { '' }

    shared_examples 'idempotent' do
      it 'should be idempotent' do
        Thrift::Types::Known::Any.from_object(obj, codec).to_object.should == obj
      end
    end

    subject { Thrift::Types::Known::Any.from_object(obj, codec) }

    context 'thrift struct' do
      let(:obj) { Thrift::Types::Value.from_object("foo") }

      it { subject.type.should == 'thrift/types.value.Value' }
      it { subject.value.should == '{"2":{"str":"foo"}}' }

      include_examples 'idempotent'

      context 'yaml only' do
        let(:codec) { 'yaml' }

        it { subject.type.should == 'thrift-yaml/types.value.Value' }
        it { subject.value.should == "string_value: foo\n" }

        include_examples 'idempotent'
      end

      context 'json only' do
        let(:codec) { 'json' }

        it { subject.type.should == 'thrift-json/types.value.Value' }
        it { subject.value.should == '{"string_value":"foo"}' }

        include_examples 'idempotent'
      end
    end
  end
end
