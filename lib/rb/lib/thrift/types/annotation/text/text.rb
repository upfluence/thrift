require 'thrift/types/annotation/text/text_types'

module Thrift
  module Types
    module Annotation
      # Text encodes the structs annotated with Format, and the enums, as
      # their text form, see types/annotation/text.thrift. It accepts and
      # produces exactly the same texts as the text package of the go library.
      module Text
        class Error < ArgumentError; end

        # Raised when the text annotations of a type are invalid, or when the
        # type has none.
        class InvalidDefinitionError < Error; end

        # Raised when a text can not be decoded.
        class InvalidFormatError < Error; end

        # Raised when a value has no text form.
        class InvalidValueError < Error; end

        @codecs = {}
        @mutex = Mutex.new

        class << self
          # Returns the text form of obj, an instance of a generated struct
          # annotated with Format.
          def dump(obj)
            codec = struct_codec(obj.class)

            codec.format(codec.fields.to_h { |f| [f.name, obj.public_send(f.name)] })
          end

          # Returns the instance of klass, a generated struct annotated with
          # Format, of text form text.
          def load(klass, text)
            klass.new(struct_codec(klass).parse(text))
          end

          # Returns the text form of value, a value of the generated enum
          # module mod.
          def dump_enum(mod, value)
            enum_codec(mod).format(value)
          end

          # Returns the value of the generated enum module mod of text form
          # text.
          def load_enum(mod, text)
            enum_codec(mod).parse(text)
          end

          # Raises an InvalidDefinitionError if the text annotations of type,
          # a generated struct class or enum module, are invalid.
          def validate(type)
            type.is_a?(Class) ? struct_codec(type) : enum_codec(type)

            nil
          end

          # @api private
          def struct_codec(klass)
            cached(klass) { StructCodec.for_class(klass) }
          end

          # @api private
          def enum_codec(mod)
            cached(mod) { EnumCodec.for_module(mod) }
          end

          # Returns the structured annotation of type klass of annotations,
          # nil if there is none.
          #
          # @api private
          def find_annotation(annotations, klass)
            found = annotations.select { |a| a.is_a?(klass) }

            raise InvalidDefinitionError, "#{klass::NAMESPACE}.#{klass::NAME} is set twice" if found.length > 1

            found.first
          end

          private

          # Invalid definitions are not cached, they raise on every call.
          def cached(type)
            codec = @mutex.synchronize { @codecs[type] }

            return codec if codec

            codec = yield

            @mutex.synchronize { @codecs[type] = codec }
          end
        end
      end
    end
  end
end

require 'thrift/types/annotation/text/pattern'
require 'thrift/types/annotation/text/caser'
require 'thrift/types/annotation/text/enum_codec'
require 'thrift/types/annotation/text/struct_codec'
