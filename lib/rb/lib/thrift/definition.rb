module Thrift
  class StructDefinition
    attr_reader :klass

    def initialize(klass)
      @klass = klass
    end

    def namespace
      @klass::NAMESPACE
    end

    def name
      @klass::NAME
    end

    def struct_type
      "#{namespace}.#{name}"
    end
  end

  class ServiceDefinition < StructDefinition
    attr_reader :klass

    def initialize(klass)
      @klass = klass
    end

    def client_class
      @klass::Client
    end

    def processor_class
      @klass::Processor
    end

    def namespace
      @klass::NAMESPACE
    end

    def service
      @klass::SERVICE
    end

    def service_type
      "#{namespace}.#{service}"
    end
  end

  STRUCT_DEFINITIONS = {}
  SERVICE_DEFINITIONS = {}

  class << self
    def register_struct_type(klass)
      definition = StructDefinition.new(klass)
      STRUCT_DEFINITIONS[definition.struct_type] = definition
    end

    def register_service_type(klass)
      definition = ServiceDefinition.new(klass)
      SERVICE_DEFINITIONS[definition.service_type] = definition
    end
  end
end
