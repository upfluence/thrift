#
# Autogenerated by Thrift Compiler (2.5.4-upfluence)
#
# DO NOT EDIT UNLESS YOU ARE SURE THAT YOU KNOW WHAT YOU ARE DOING
#

require 'thrift'

module Thrift
  module Types
    module Known
      module Any
        class Any; end

        class Any
          include ::Thrift::Struct, ::Thrift::Struct_Union

          NAME = 'Any'.freeze
          NAMESPACE = 'types.known.any'.freeze

          LEGACY_ANNOTATIONS = {
          }.freeze

          STRUCTURED_ANNOTATIONS = [
          ].freeze

          THRIFT_FIELD_INDEX_TYPE = 1
          THRIFT_FIELD_INDEX_VALUE = 2

          THRIFT_FIELD_TYPE_LEGACY_ANNOTATIONS = {
          }.freeze

          THRIFT_FIELD_TYPE_STRUCTURED_ANNOTATIONS = [
          ].freeze

          THRIFT_FIELD_VALUE_LEGACY_ANNOTATIONS = {
          }.freeze

          THRIFT_FIELD_VALUE_STRUCTURED_ANNOTATIONS = [
          ].freeze

          FIELDS = {
            THRIFT_FIELD_INDEX_TYPE => {type: ::Thrift::Types::STRING, name: 'type', legacy_annotations: THRIFT_FIELD_TYPE_LEGACY_ANNOTATIONS, structured_annotations: THRIFT_FIELD_TYPE_STRUCTURED_ANNOTATIONS},
            THRIFT_FIELD_INDEX_VALUE => {type: ::Thrift::Types::STRING, name: 'value', binary: true, legacy_annotations: THRIFT_FIELD_VALUE_LEGACY_ANNOTATIONS, structured_annotations: THRIFT_FIELD_VALUE_STRUCTURED_ANNOTATIONS}
          }

          def struct_fields; FIELDS; end

          def validate
            raise ::Thrift::ProtocolException.new(::Thrift::ProtocolException::UNKNOWN, 'Required field type is unset!') unless @type
            raise ::Thrift::ProtocolException.new(::Thrift::ProtocolException::UNKNOWN, 'Required field value is unset!') unless @value
          end

          ::Thrift::Struct.generate_accessors self
          ::Thrift.register_struct_type self
        end

      end
    end
  end
end
