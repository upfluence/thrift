module Thrift
  module Types
    module Annotation
      module Text
        # EnumCodec encodes the values of an enum as text, following its Name
        # and EnumFormat annotations.
        class EnumCodec
          Value = ::Struct.new(:name, :value, :texts)

          attr_reader :name, :values

          class << self
            # Builds the codec of a generated enum module.
            def for_module(mod)
              new(
                name:                   mod::NAME,
                structured_annotations: mod::STRUCTURED_ANNOTATIONS,
                values:                 mod::THRIFT_NAMES.map do |value, name|
                  {
                    name:                   name,
                    value:                  value,
                    structured_annotations: mod::VALUE_STRUCTURED_ANNOTATIONS.fetch(value, [])
                  }
                end
              )
            rescue NameError => e
              raise InvalidDefinitionError, "#{mod} is not a thrift enum generated with annotations: #{e.message}"
            end
          end

          # values are hashes with the name, value and structured_annotations
          # keys.
          def initialize(name:, values:, structured_annotations: [])
            @name = name
            @values = []
            @texts = {}
            @from_texts = {}

            compile(values, Text.find_annotation(structured_annotations, EnumFormat) || EnumFormat.new)
          rescue InvalidDefinitionError => e
            raise InvalidDefinitionError, "enum #{name}: #{e.message}"
          end

          # Returns the text form of value.
          def format(value)
            @texts.fetch(value) { raise InvalidValueError, "#{name}(#{value.inspect})" }
          end

          # Returns the value of the text form text.
          def parse(text)
            @from_texts.fetch(text) { raise InvalidFormatError, "invalid #{name} #{text.inspect}" }
          end

          private

          def compile(values, format)
            unless Case::VALID_VALUES.include?(format.format)
              raise InvalidDefinitionError, "EnumFormat.format #{format.format.inspect} is unknown"
            end

            caser = Caser.new(format.format)
            owners = {}

            values.each do |v|
              add_value(v, value_texts(v, format, caser), owners)
            end

            @values.freeze
          end

          def add_value(definition, texts, owners)
            if @texts.key?(definition[:value])
              other = owners[@texts[definition[:value]]]

              raise InvalidDefinitionError,
                    "values #{other} and #{definition[:name]} share the id #{definition[:value]}"
            end

            texts.each { |text| add_text(definition, text, owners) }

            @texts[definition[:value]] = texts.first
            @values << Value.new(definition[:name], definition[:value], texts)
          end

          def add_text(definition, text, owners)
            if owners.key?(text)
              raise InvalidDefinitionError,
                    "values #{owners[text]} and #{definition[:name]} share the text #{text.inspect}"
            end

            owners[text] = definition[:name]
            @from_texts[text] = definition[:value]
          end

          # Returns the text forms of the value, the canonical one first.
          def value_texts(definition, format, caser)
            text_name = Text.find_annotation(definition[:structured_annotations], Name)

            return [derived_text(definition, format, caser)] if text_name.nil?

            raise InvalidDefinitionError, "value #{definition[:name]} has an empty text" if text_name.value.to_s.empty?

            (text_name.accepted || []).each_with_object([text_name.value]) do |accepted, texts|
              if accepted.to_s.empty?
                raise InvalidDefinitionError, "value #{definition[:name]} has an empty accepted text"
              end

              texts << accepted unless texts.include?(accepted)
            end
          end

          def derived_text(definition, format, caser)
            trimmed = trim(definition[:name], format)

            raise InvalidDefinitionError, "value #{definition[:name]} is empty once trimmed" if trimmed.empty?

            caser.call(trimmed)
          end

          def trim(name, format)
            trimmed = name
            prefix = (format.trim_prefixes || []).find { |p| trimmed.start_with?(p) }
            trimmed = trimmed.delete_prefix(prefix) if prefix
            suffix = (format.trim_suffixes || []).find { |s| trimmed.end_with?(s) }
            trimmed = trimmed.delete_suffix(suffix) if suffix
            trimmed
          end
        end
      end
    end
  end
end
