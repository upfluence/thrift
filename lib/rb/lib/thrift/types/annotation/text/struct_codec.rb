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

          # Field is an integer, string or enum field of the pattern, once the
          # struct fields are inlined. Its name is its path joined by dots,
          # e.g. "entity.id".
          Field = ::Struct.new(:name, :kind, :bits, :enum) do
            def path
              name.split('.')
            end
          end

          attr_reader :name, :fields, :pattern

          class << self
            # Builds the codec of a generated struct class. visiting holds the
            # classes being compiled to reject the structs embedding
            # themselves.
            def for_class(klass, visiting = [])
              unless klass.is_a?(Class) && klass.include?(::Thrift::Struct_Union) && klass.const_defined?(:FIELDS)
                raise InvalidDefinitionError, "#{klass} is not a thrift struct"
              end

              raise InvalidDefinitionError, "#{klass} embeds itself" if visiting.include?(klass)

              fields = klass::FIELDS.values
              referenced = referenced_fields(klass)

              new(
                name:                   klass::NAME,
                structured_annotations: klass::STRUCTURED_ANNOTATIONS,
                fields:                 fields.map { |f| { name: f[:name], optional: f[:optional] || false } },
                types:                  fields.select { |f| referenced.include?(f[:name]) }.to_h do |f|
                  [f[:name], field_type(f, visiting + [klass])]
                end.compact,
                union:                  klass < ::Thrift::Union,
                exception:              klass < ::Thrift::Exception
              )
            end

            private

            # Only the types of the referenced fields are resolved, the
            # constructor reporting the invalid annotations.
            def referenced_fields(klass)
              format = Text.find_annotation(klass::STRUCTURED_ANNOTATIONS, Format)

              format ? Pattern.new(format.pattern).fields : []
            rescue InvalidDefinitionError
              []
            end

            def field_type(field, visiting)
              return { kind: :enum, enum: Text.enum_codec(field[:enum_class]) } if field[:enum_class]
              return { kind: :string } if field[:type] == ::Thrift::Types::STRING && !field[:binary]
              return struct_type(field[:class], visiting) if field[:class]

              bits = INTEGER_BITS[field[:type]]

              { kind: :integer, bits: bits } if bits
            end

            def struct_type(klass, visiting)
              return { kind: :struct, struct: nil } unless Text.find_annotation(klass::STRUCTURED_ANNOTATIONS, Format)

              { kind: :struct, struct: Text.struct_codec(klass, visiting) }
            end
          end

          # fields are hashes with the name and optional keys, types are the
          # types of the fields, keyed by name: a hash with the kind (:integer,
          # :string, :enum or :struct), the bits of an integer, the EnumCodec
          # of an enum and the StructCodec of a struct, nil when the struct has
          # no Format. A referenced field missing from types has an unsupported
          # type, an optional field can be left out of the pattern.
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
          # field name as a string or a symbol: an Integer for integer and
          # enum fields, a String for string fields and a Hash for struct
          # fields.
          def format(values)
            texts = fields.map do |f|
              found, value = lookup(values, f.path)

              raise InvalidValueError, "#{name}: missing field #{f.name.inspect}" unless found

              format_value(f, value)
            end

            @pattern.format(texts)
          rescue InvalidValueError => e
            raise InvalidValueError, "#{name}: #{e.message}"
          end

          # Returns the values of the struct of text form text, keyed by field
          # name, a struct field being a Hash.
          def parse(text)
            raise InvalidFormatError, "#{text.inspect} is not a string" unless text.is_a?(String)

            fields.zip(@pattern.scan(text)).each_with_object({}) do |(f, t), values|
              *parents, leaf = f.path

              parents.reduce(values) { |h, p| h[p] ||= {} }[leaf] = parse_value(f, t)
            end
          rescue InvalidFormatError => e
            raise InvalidFormatError, "#{name}: #{e.message}"
          end

          private

          def lookup(values, path)
            path.reduce([true, values]) do |(_, h), p|
              return [false, nil] unless h.is_a?(Hash)

              key = [p, p.to_sym].find { |k| h.key?(k) }

              return [false, nil] if key.nil?

              [true, h[key]]
            end
          end

          def compile_fields(fields, types)
            referenced = @pattern.fields

            fields.each do |f|
              next if f[:optional] || referenced.include?(f[:name])

              raise InvalidDefinitionError, "pattern does not reference field #{f[:name].inspect}"
            end

            by_name = fields.to_h { |f| [f[:name], f] }
            leaves = referenced.each_with_object({}) do |field_name, res|
              inline_field(by_name[field_name], field_name, types[field_name], res)
            end

            @pattern.fields.map { |n| Field.new(n, leaves[n][:kind], leaves[n][:bits], leaves[n][:enum]) }
          end

          def inline_field(field, field_name, type, leaves)
            raise InvalidDefinitionError, "pattern references unknown field #{field_name.inspect}" if field.nil?
            raise InvalidDefinitionError, "field #{field_name.inspect} must not be optional" if field[:optional]
            raise InvalidDefinitionError, "field #{field_name.inspect} has an unsupported type" if type.nil?
            return leaves[field_name] = type unless type[:kind] == :struct

            nested = type[:struct]

            if nested.nil?
              raise InvalidDefinitionError, "field #{field_name.inspect} is a struct without Format annotation"
            end

            nested.fields.each do |f|
              leaves["#{field_name}.#{f.name}"] = { kind: f.kind, bits: f.bits, enum: f.enum }
            end

            @pattern = @pattern.inline(field_name, nested.pattern)
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
