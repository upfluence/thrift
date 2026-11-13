require 'spec_helper'
require 'json'

# The cases are shared with the go library.
TEXT_CASES = JSON.parse(File.read(File.expand_path('../../../../../test/TextTestCases.json', __dir__)))

describe Thrift::Types::Annotation::Text do
  text = Thrift::Types::Annotation::Text

  def enum_codec(text, enum_case)
    format = enum_case['format']
    annotations = []

    if format
      kase = format['format']
      kase = text::Case.const_get(kase) if kase.is_a?(String)

      annotations << text::EnumFormat.new(
        {
          'format'              => kase,
          'trim_prefixes'       => format['trim_prefixes'],
          'trim_suffixes'       => format['trim_suffixes'],
          'accept_thrift_names' => format['accept_thrift_names']
        }.compact
      )
    end

    text::EnumCodec.new(
      name:                   enum_case['name'],
      structured_annotations: annotations,
      values:                 enum_case['values'].map do |v|
        {
          name:                   v['name'],
          value:                  v['value'],
          structured_annotations: v.key?('text') ? [text::Name.new(value: v['text'])] : []
        }
      end
    )
  end

  def struct_codec(text, struct_case, enums)
    bits = { 'i8' => 8, 'i16' => 16, 'i32' => 32, 'i64' => 64 }
    pattern = struct_case['pattern']
    fields = struct_case['fields'].map { |f| { name: f['name'], optional: f['requiredness'] == 'optional' } }
    types = struct_case['fields'].to_h do |f|
      type =
        case f['type']
        when 'string' then { kind: :string }
        when 'enum' then { kind: :enum, enum: enums.fetch(f['enum']) }
        else bits[f['type']] && { kind: :integer, bits: bits[f['type']] }
        end

      [f['name'], type]
    end

    text::StructCodec.new(
      name:                   struct_case['name'],
      structured_annotations: pattern ? [text::Format.new(pattern: pattern)] : [],
      union:                  struct_case['union'] || false,
      fields:                 fields,
      types:                  types.compact
    )
  end

  describe 'shared cases' do
    shared_examples 'split words' do |have, want|
      it "splits #{have.inspect}" do
        expect(text::Caser.split_words(have)).to eq(want)
      end
    end

    shared_examples 'a cased name' do |have, kase, want|
      it "renders #{have} in #{kase}" do
        caser = text::Caser.new(text::Case.const_get(kase))

        expect(caser.call(have)).to eq(want)
      end
    end

    shared_examples 'a pattern' do |pattern_case|
      it "handles #{pattern_case['pattern'].inspect}" do
        if pattern_case['error']
          expect { text::Pattern.new(pattern_case['pattern']) }.to raise_error(text::InvalidDefinitionError)
          next
        end

        pattern = text::Pattern.new(pattern_case['pattern'])

        expect(pattern.fields).to eq(pattern_case['fields'])

        pattern_case['scans'].each do |sc|
          if sc['error']
            expect { pattern.scan(sc['have']) }.to raise_error(text::InvalidFormatError)
          else
            expect(pattern.scan(sc['have'])).to eq(sc['want'])
          end
        end

        pattern_case['formats'].each do |fc|
          if fc['error']
            expect { pattern.format(fc['have']) }.to raise_error(text::InvalidValueError)
          else
            expect(pattern.format(fc['have'])).to eq(fc['want'])
          end
        end
      end
    end

    shared_examples 'an enum' do |enum_case|
      it "handles #{enum_case['id']}" do
        if enum_case['error']
          expect { enum_codec(text, enum_case) }.to raise_error(text::InvalidDefinitionError)
          next
        end

        codec = enum_codec(text, enum_case)

        enum_case['want'].each do |w|
          expect(codec.format(w['value'])).to eq(w['texts'].first)

          w['texts'].each { |t| expect(codec.parse(t)).to eq(w['value']) }
        end

        expect { codec.parse('unknown') }.to raise_error(text::InvalidFormatError)
        expect { codec.format(42) }.to raise_error(text::InvalidValueError)
      end
    end

    shared_examples 'a struct' do |struct_case|
      it "handles #{struct_case['name']}" do
        enums = TEXT_CASES['enums'].reject { |e| e['error'] }.to_h { |e| [e['id'], enum_codec(text, e)] }

        if struct_case['error']
          expect { struct_codec(text, struct_case, enums) }.to raise_error(text::InvalidDefinitionError)
          next
        end

        codec = struct_codec(text, struct_case, enums)

        struct_case['formats'].each do |fc|
          if fc['error']
            expect { codec.format(fc['have']) }.to raise_error(text::InvalidValueError)
          else
            expect(codec.format(fc['have'])).to eq(fc['want'])
          end
        end

        struct_case['scans'].each do |sc|
          if sc['error']
            expect { codec.parse(sc['have']) }.to raise_error(text::InvalidFormatError)
          else
            expect(codec.parse(sc['have'])).to eq(sc['want'])
          end
        end
      end
    end

    TEXT_CASES['words'].each { |c| it_behaves_like 'split words', c['have'], c['want'] }
    TEXT_CASES['cases'].each { |c| it_behaves_like 'a cased name', c['have'], c['case'], c['want'] }
    TEXT_CASES['patterns'].each { |c| it_behaves_like 'a pattern', c }
    TEXT_CASES['enums'].each { |c| it_behaves_like 'an enum', c }
    TEXT_CASES['structs'].each { |c| it_behaves_like 'a struct', c }
  end
end
