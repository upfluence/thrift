require 'thrift/types/value/value_types'

module Thrift
  module Types
    module Value
      class ListValue
        def to_array
          values.map(&:to_object)
        end

        class << self
          def from_array(v)
            ListValue.new(values: v.map { |vv| Value.from_object(vv) })
          end
        end
      end

      class MapValue
        def to_hash
          entries.reduce({}) do |acc, e|
            acc.merge(e.key.to_object => e.value.to_object)
          end
        end

        class << self
          def from_hash(v)
            MapValue.new(
              entries: v.reduce([]) do |acc, (k, vv)|
                acc +  [
                  MapEntry.new(
                    key:   Value.from_object(k),
                    value: Value.from_object(vv)
                  )
                ]
              end
            )
          end
        end
      end

      class StructValue
        def to_hash
          fields.reduce({}) do |acc, (k, v)|
            acc.merge(k => v.to_object)
          end
        end

        class << self
          def from_object(v)
            StructValue.new(
              fields: v.instance_variables.reduce({}) do |acc, k|
                acc.merge(
                  k.to_s[1..-1] => Value.from_object(v.instance_variable_get(k))
                )
              end
            )
          end
        end
      end

      class << self
        def from_object(v)
          Value.from_object(v)
        end
      end

      class Value
        def to_object
          case get_value.class
          when NullValue
             nil
          when ListValue
            list_value.to_array
          when MapValue
            map_value.to_hash
          when StructValue
            struct_value.to_hash
          else
            get_value
          end
        end

        class << self
          def from_object(v)
            case v
            when NilClass
              Value.new(null_value: NullValue.new)
            when Symbol
              Value.new(string_value: v.to_s)
            when String
              if v.encoding.eql?(Encoding::UTF_8) && v.valid_encoding?
                Value.new(string_value: v)
              else
                Value.new(binary_value: v)
              end
            when Integer
              Value.new(integer_value: v)
            when Float
              Value.new(double_value: v)
            when TrueClass, FalseClass
              Value.new(bool_value: v)
            when Array
              Value.new(list_value: ListValue.from_array(v))
            when Hash
              Value.new(map_value: MapValue.from_hash(v))
            else
              Value.new(struct_value: StructValue.from_object(v))
            end
          end
        end
      end
    end
  end
end
