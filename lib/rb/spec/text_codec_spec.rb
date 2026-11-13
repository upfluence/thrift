require 'spec_helper'
require 'texttest/text_test_types'

# Exercises the text codec through code generated from test/TextTest.thrift.
describe Thrift::Types::Annotation::Text do
  text = Thrift::Types::Annotation::Text
  tt = Texttest

  describe '.dump' do
    shared_examples 'a dumped struct' do |obj, want|
      it "dumps #{obj.inspect}" do
        expect(text.dump(obj)).to eq(want)
      end
    end

    shared_examples 'a struct without text form' do |obj, error|
      it "rejects #{obj.inspect}" do
        expect { text.dump(obj) }.to raise_error(error)
      end
    end

    it_behaves_like 'a dumped struct', tt::Ownership.new(type: tt::Type::PaypalAPI, entity_id: 42), 'paypal_api:42'
    it_behaves_like 'a dumped struct',
                    tt::Handle.new(media: tt::MediaType::MediaYoutubeShort, name: 'bob', shard: -3),
                    'youtube/bob@{-3}'

    it_behaves_like 'a struct without text form',
                    tt::Handle.new(media: 0, name: 'a@{b', shard: 0),
                    text::InvalidValueError
    it_behaves_like 'a struct without text form', tt::Ownership.new(type: 42, entity_id: 1), text::InvalidValueError
    it_behaves_like 'a struct without text form', tt::Ownership.new(type: 0, entity_id: '1'), text::InvalidValueError
    it_behaves_like 'a struct without text form', tt::Ownership.new(type: 0), text::InvalidValueError
    it_behaves_like 'a struct without text form', tt::Unannotated.new(id: 1), text::InvalidDefinitionError

    it 'raises argument errors' do
      expect { text.dump(tt::Unannotated.new(id: 1)) }.to raise_error(ArgumentError)
    end
  end

  describe '.load' do
    shared_examples 'a loaded struct' do |have, want|
      it "loads #{have.inspect}" do
        expect(text.load(want.class, have)).to eq(want)
      end
    end

    shared_examples 'an invalid text' do |klass, have, error|
      it "rejects #{have.inspect}" do
        expect { text.load(klass, have) }.to raise_error(error)
      end
    end

    it_behaves_like 'a loaded struct', 'gmail:7', tt::Ownership.new(type: tt::Type::GoogleOAuth, entity_id: 7)
    it_behaves_like 'a loaded struct', 'Type_Team:-1', tt::Ownership.new(type: tt::Type::Team, entity_id: -1)
    it_behaves_like 'a loaded struct',
                    'tt/x@{2}',
                    tt::Handle.new(media: tt::MediaType::MediaTiktok, name: 'x', shard: 2)

    it_behaves_like 'an invalid text', tt::Ownership, 'nope:1', text::InvalidFormatError
    it_behaves_like 'an invalid text', tt::Ownership, 'team:1.5', text::InvalidFormatError
    it_behaves_like 'an invalid text', tt::Ownership, 'team', text::InvalidFormatError
    it_behaves_like 'an invalid text', tt::Ownership, nil, text::InvalidFormatError
    it_behaves_like 'an invalid text', tt::Unannotated, '1', text::InvalidDefinitionError
  end

  describe 'enums' do
    it 'dumps and loads values' do
      expect(text.dump_enum(tt::MediaType, tt::MediaType::MediaInstagram)).to eq('instagram')
      expect(text.load_enum(tt::MediaType, 'tt')).to eq(tt::MediaType::MediaTiktok)
      expect(text.dump_enum(tt::Plain, tt::Plain::FooBar)).to eq('foo_bar')
    end

    it 'rejects the thrift names unless accepted' do
      expect { text.load_enum(tt::MediaType, 'MediaTiktok') }.to raise_error(text::InvalidFormatError)
      expect(text.load_enum(tt::Type, 'Type_Company')).to eq(tt::Type::Company)
    end
  end

  describe '.validate' do
    shared_examples 'a valid type' do |type|
      it "accepts #{type}" do
        expect(text.validate(type)).to be_nil
      end
    end

    shared_examples 'an invalid type' do |type|
      it "rejects #{type}" do
        expect { text.validate(type) }.to raise_error(text::InvalidDefinitionError)
      end
    end

    [tt::Handle, tt::Ownership, tt::Type, tt::Plain].each { |t| it_behaves_like 'a valid type', t }

    [tt::Unannotated, tt::OptionalField, tt::Union, tt::SeparatorConflict, String, Comparable].each do |t|
      it_behaves_like 'an invalid type', t
    end
  end

  describe text::JSONSerializable do
    let(:klass) { Class.new(tt::Ownership) { include Thrift::Types::Annotation::Text::JSONSerializable } }
    let(:owner) { klass.new(type: tt::Type::Team, entity_id: 3) }

    it 'renders the text form in as_json' do
      expect(owner.as_json).to eq('team:3')
    end

    it 'renders the text form in to_json' do
      expect(JSON.generate(owner: owner)).to eq('{"owner":"team:3"}')
    end

    it 'leaves the structs without the module unchanged' do
      expect(tt::Ownership.ancestors).not_to include(text::JSONSerializable)
    end

    it 'raises when the struct has no text form' do
      expect { klass.new(type: tt::Type::Team).to_json }.to raise_error(text::InvalidValueError)
    end
  end
end
