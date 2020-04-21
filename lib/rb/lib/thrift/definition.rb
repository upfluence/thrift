module Thrift
  class StructDefinition
    attr_reader :namespace, :name, :klass
    def initialize(klass)
      @namespace = klass::NAMESPACE
      @name = klass::NAME
      @klass = klass
    end

    def struct_type
      "#{@namespace}.#{@name}"
    end
  end

  STRUCT_DEFINITIONS = {}

  class << self

    def register_struct_type(klass)
      definition = StructDefinition.new(klass)
      STRUCT_DEFINITIONS[definition.struct_type] = definition
    end
  end
end
