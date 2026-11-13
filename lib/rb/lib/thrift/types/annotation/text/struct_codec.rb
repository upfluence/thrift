module Thrift
  module Types
    module Annotation
      module Text
        # StructCodec encodes the values of a struct as text, following its
        # Format annotation.
        class StructCodec
          INTEGER = /\A[-+]?[0-9]+\z/.freeze

          INTEGER_BITS = {
            ::Thrift::Types::BYTE => 8,
            ::Thrift::Types::I16  => 16,
            ::Thrift::Types::I32  => 32,
            ::Thrift::Types::I64  => 64
          }.freeze

          Field = ::Struct.new(:name, :kind, :bits, :enum)

          attr_reader :name, :fields

          class << self
            # Builds the codec of a generated struct class.
            def for_class(klass)
              unless klass.is_a?(Class) && klass.include?(::Thrift::Struct_Union) && klass.const_defined?(:FIELDS)
                raise InvalidDefinitionError, "#{klass} is not a thrift struct"
              end

              fields = klass::FIELDS.values

              new(
                name:                   klass::NAME,
                structured_annotations: klass::STRUCTURED_ANNOTATIONS,
                fields:                 fields.map { |f| { name: f[:name], optional: f[:optional] || false } },
                types:                  fields.each_with_object({}) do |f, types|
                  type = field_type(f)
                  types[f[:name]] = type if type
                end,
                union:                  klass < ::Thrift::Union,
                exception:              klass < ::Thrift::Exception
              )
            end

            private

            def field_type(field)
              return { kind: :enum, enum: Text.enum_codec(field[:enum_class]) } if field[:enum_class]
              return { kind: :string } if field[:type] == ::Thrift::Types::STRING && !field[:binary]

              bits = INTEGER_BITS[field[:type]]

              { kind: :integer, bits: bits } if bits
            end
          end

          # fields are hashes with the name and optional keys, types are the
          # types of the fields, keyed by name: a hash with the kind (:integer,
          # :string or :enum), the bits of an integer and the EnumCodec of an
          # enum. A field missing from types has an unsupported type.
          def initialize(name:, structured_annotations:, fields:, types:, union: false, exception: false)
            @name = name

            format = Text.find_annotation(structured_annotations, Format)

            raise InvalidDefinitionError, 'no Format annotation' if format.nil?
            raise InvalidDefinitionError, 'Format can only be set on a struct' if union || exception

            @pattern = Pattern.new(format.pattern)
            @fields = compile_fields(fields, types).freeze

            validate_separators
          rescue InvalidDefinitionError => e
            raise InvalidDefinitionError, "struct #{name}: #{e.message}"
          end

          # Returns the text form of the struct holding the values, keyed by
          # field name: an Integer for integer and enum fields, a String for
          # string fields.
          def format(values)
            texts = fields.map do |f|
              raise InvalidValueError, "#{name}: missing field #{f.name.inspect}" unless values.key?(f.name)

              format_value(f, values[f.name])
            end

            @pattern.format(texts)
          rescue InvalidValueError => e
            raise InvalidValueError, "#{name}: #{e.message}"
          end

          # Returns the values of the struct of text form text, keyed by field
          # name.
          def parse(text)
            raise InvalidFormatError, "#{text.inspect} is not a string" unless text.is_a?(String)

            fields.zip(@pattern.scan(text)).to_h { |f, t| [f.name, parse_value(f, t)] }
          rescue InvalidFormatError => e
            raise InvalidFormatError, "#{name}: #{e.message}"
          end

          private

          def compile_fields(fields, types)
            by_name = fields.to_h { |f| [f[:name], f] }

            res = @pattern.fields.map do |field_name|
              field = by_name[field_name]

              raise InvalidDefinitionError, "pattern references unknown field #{field_name.inspect}" if field.nil?
              raise InvalidDefinitionError, "field #{field_name.inspect} must not be optional" if field[:optional]

              type = types[field_name]

              raise InvalidDefinitionError, "field #{field_name.inspect} has an unsupported type" if type.nil?

              Field.new(field_name, type[:kind], type[:bits], type[:enum])
            end

            fields.each do |f|
              unless @pattern.fields.include?(f[:name])
                raise InvalidDefinitionError, "pattern does not reference field #{f[:name].inspect}"
              end
            end

            res
          end

          # Ensures every enum and integer value can be scanned back: the
          # literal following such a field can not occur in its text form.
          # String values are checked when formatted.
          def validate_separators
            segments = @pattern.segments
            field = -1

            segments.each_with_index do |segment, i|
              next unless segment.field?

              field += 1

              next if i == segments.length - 1

              validate_separator(fields[field], segments[i + 1].literal)
            end
          end

          def validate_separator(field, sep)
            case field.kind
            when :integer
              if '+-0123456789'.include?(sep[0])
                raise InvalidDefinitionError,
                      "integer field #{field.name.inspect} can not be followed by #{sep.inspect}"
              end
            when :enum
              field.enum.values.each do |v|
                v.texts.each do |t|
                  next if (t + sep).index(sep) == t.length

                  raise InvalidDefinitionError,
                        "text #{t.inspect} of #{field.enum.name}.#{v.name} conflicts with the separator #{sep.inspect}"
                end
              end
            end
          end

          def format_value(field, value)
            case field.kind
            when :string
              return value if value.is_a?(String)
            when :integer
              if value.is_a?(Integer)
                raise InvalidValueError, "#{value} overflows i#{field.bits}" unless fits?(value, field.bits)

                return value.to_s
              end
            when :enum
              return field.enum.format(value) if value.is_a?(Integer)
            end

            raise InvalidValueError, "#{field.name}: unexpected #{value.inspect} value"
          end

          def parse_value(field, text)
            case field.kind
            when :string
              text
            when :integer
              value = INTEGER.match?(text) ? Integer(text, 10) : nil

              unless value && fits?(value, field.bits)
                raise InvalidFormatError, "#{text.inspect} is not an i#{field.bits}"
              end

              value
            when :enum
              field.enum.parse(text)
            end
          end

          def fits?(value, bits)
            limit = 1 << (bits - 1)

            value >= -limit && value < limit
          end
        end
      end
    end
  end
end
