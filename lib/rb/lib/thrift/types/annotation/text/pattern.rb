module Thrift
  module Types
    module Annotation
      module Text
        # Pattern is a compiled pattern following a subset of the python
        # format string syntax: "{name}" is a field placeholder, "{{" and "}}"
        # are the escaped braces, everything else is literal. Format specs and
        # conversions ("{name:spec}", "{name!r}") are not supported.
        #
        # A pattern can be formatted and scanned back: a field holds the text
        # up to the first occurrence of the literal following it, the last
        # field holds the rest of the text. Consequently, two fields must be
        # separated by a literal and formatting a value containing the literal
        # following it fails.
        #
        # It mirrors the internal fstring package of the go library.
        #
        # @api private
        class Pattern
          Segment = ::Struct.new(:literal, :field) do
            def field?
              !field.nil?
            end
          end

          IDENTIFIER = /\A[A-Za-z_][A-Za-z0-9_]*\z/.freeze
          TOKEN = /\{\{|\}\}|\{[^{}]*\}|[{}]|[^{}]+/.freeze

          attr_reader :segments, :fields

          def initialize(pattern)
            @pattern = pattern.dup.freeze
            @segments = compile.freeze
            @fields = @segments.select(&:field?).map(&:field).freeze
          end

          def to_s
            @pattern
          end

          # Returns the text of every field of text, in pattern order.
          def scan(text)
            values = []
            rest = text

            segments.each_with_index do |segment, i|
              unless segment.field?
                raise mismatch(text) unless rest.start_with?(segment.literal)

                rest = rest[segment.literal.length..]
                next
              end

              next_segment = segments[i + 1]
              j = next_segment ? rest.index(next_segment.literal) : rest.length

              raise mismatch(text) if j.nil?

              values << rest[0...j]
              rest = rest[j..]
            end

            raise mismatch(text) unless rest.empty?

            values
          end

          # Formats the pattern with the text of every field, in pattern order.
          def format(values)
            unless values.length == fields.length
              raise ArgumentError,
                    "#{values.length} values given for the #{fields.length} fields of #{@pattern.inspect}"
            end

            field = -1

            segments.each_with_index.map do |segment, i|
              next segment.literal unless segment.field?

              field += 1
              format_field(segment.field, values[field], segments[i + 1])
            end.join
          end

          private

          def format_field(field, value, next_segment)
            return value if next_segment.nil?

            # The value is scanned back up to the first occurrence of the next
            # literal, which must be the one following the value.
            return value if (value + next_segment.literal).index(next_segment.literal) == value.length

            raise InvalidValueError, "field #{field.inspect}: #{value.inspect} contains #{next_segment.literal.inspect}"
          end

          def mismatch(text)
            InvalidFormatError.new("#{text.inspect} does not match #{@pattern.inspect}")
          end

          def compile
            @pattern.scan(TOKEN).each_with_object([]) do |token, segments|
              case token
              when '{{', '}}' then append_literal(segments, token[0])
              when '{' then raise invalid_pattern('unclosed {')
              when '}' then raise invalid_pattern('single } must be escaped')
              when /\A\{(.*)\}\z/ then append_field(segments, Regexp.last_match(1))
              else append_literal(segments, token)
              end
            end
          end

          def append_literal(segments, literal)
            if segments.last.nil? || segments.last.field?
              segments << Segment.new(literal, nil)
            else
              segments[-1] = Segment.new(segments.last.literal + literal, nil)
            end
          end

          def append_field(segments, name)
            raise invalid_pattern("invalid field name #{name.inspect}") unless IDENTIFIER.match?(name)
            raise invalid_pattern("field #{name.inspect} is repeated") if segments.any? { |s| s.field == name }
            raise invalid_pattern('fields must be separated by a literal') if segments.last&.field?

            segments << Segment.new(nil, name)
          end

          def invalid_pattern(reason)
            InvalidDefinitionError.new("invalid pattern #{@pattern.inspect}: #{reason}")
          end
        end
      end
    end
  end
end
