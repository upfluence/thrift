require 'thrift/types/known/any_types'
require 'json'
require 'yaml'

module Thrift
  module Types
    module Known
      module Any
        class CustomYAML
          class << self
            def dump(obj)
              YAML.dump(obj)[4..-1]
            end

            def load(val)
              YAML.safe_load(val)
            end
          end
        end

        class MetaCodec
          def initialize(klass)
            @klass = klass
          end

          def encode(obj)
            @klass.dump(to_hash(obj))
          end

          def decode(buf, obj)
            from_hash(obj, @klass.load(buf))
          end

          private

          def to_hash(obj)
            to_hash_field(obj, type: Thrift::Types::STRUCT)
          end

          def to_hash_field(value, field)
            case field[:type]
            when Thrift::Types::STRUCT
              value.struct_fields.values.reduce({}) do |acc, sfield|
                name = sfield[:name]
                vv = value.send("#{name}?") ? to_hash_field(value.send(name), sfield) : nil
                vv.nil? ? acc : acc.merge(name => vv)
              end
            when Thrift::Types::LIST
              value.map { |vv| to_hash_field(vv, field[:element]) }
            when Thrift::Types::MAP
              value.reduce({}) do |acc, (k, v)|
                kv = to_hash_field(k, field[:key])
                vv = to_hash_field(v, field[:value])

                acc.merge(kv => vv)
              end
            else
              value
            end
          end

          def from_hash(obj, hash)
            return nil if hash.nil?

            obj.struct_fields.values.each do |field|
              name = field[:name]
              value = hash[name]

              obj.send("#{name}=", from_value_field(value, field)) if value
            end

            obj
          end

          def from_value_field(value, field)
            case field[:type]
            when Thrift::Types::STRUCT
              from_hash(field[:class].new, value)
            when Thrift::Types::LIST
              value.map { |vv| from_value_field(vv, field[:element]) }
            when Thrift::Types::MAP
              value.reduce({}) do |acc, (k, v)|
                kv = from_value_field(k, field[:key])
                vv = from_value_field(v, field[:value])
                acc.merge(kv => vv)
              end
            else
              value
            end
          end
        end

        class ProtocolCodec
          def initialize(protocol_factory)
            @protocol_factory = protocol_factory
          end

          def encode(obj)
            Serializer.new(@protocol_factory).serialize(obj)
          end

          def decode(buf, obj)
            obj.read(
              @protocol_factory.get_protocol(MemoryBufferTransport.new(buf))
            )
          end
        end

        DEFAULT_CODEC = ProtocolCodec.new(JsonProtocolFactory.new)
        JSON_CODEC = MetaCodec.new(JSON)
        YAML_CODEC = MetaCodec.new(CustomYAML)

        CODECS = {
          ''     => DEFAULT_CODEC,
          'json' => JSON_CODEC,
          'yaml' => YAML_CODEC,
          'yml'  => YAML_CODEC
        }


        class << self
          def from_object(obj, codec_key = '')
            Any.from_object(obj, codec_key)
          end
        end

        class Any
          class TypeNotHandled < StandardError; end
          class CodecNotHandled < StandardError; end

          class << self
            def from_object(obj, codec_key = '')
              struct_def = Thrift::STRUCT_DEFINITIONS.values.find do |v|
                obj.is_a? v.klass
              end

              raise TypeNotHandled unless struct_def
              raise CodecNotHandled unless CODECS[codec_key]

              key = codec_key.eql?('') ? '' : "-#{codec_key}"

              Any.new(
                type:  "thrift#{key}/#{struct_def.struct_type}",
                value: CODECS[codec_key].encode(obj)
              )
            end
          end

          def to_object
            codec_key, struct_type = parse_type

            raise TypeNotHandled unless Thrift::STRUCT_DEFINITIONS[struct_type]
            raise CodecNotHandled unless CODECS[codec_key]

            res = Thrift::STRUCT_DEFINITIONS[struct_type].klass.new
            CODECS[codec_key].decode(value, res)

            res
          end

          private

          def parse_type
            ts = type.split('/')

            if ts.length != 2 || (ts[0] =~ /^thrift-?/) != 0
              raise TypeNotHandled.new
            end

            codec = ts[0][6..-1]
            codec = codec[1..-1] if codec[0].eql?('-')

            [codec, ts[1]]
          end
        end
      end
    end
  end
end
