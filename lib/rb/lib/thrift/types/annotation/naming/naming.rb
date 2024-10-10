require 'thrift/types/annotation/naming/naming_types'

module Thrift
  module Types
    module Annotation
      module Naming
        class CanonicalNameExtractor
          class << self
            def extract(definition)
              definition.structured_annotations.select do |sa|
                sa.is_a? PreviouslyKnownAs
              end.map do |pka|
                "#{pka.namespace_ || definition.namespace}.#{pka.name || definition.name}"
              end
            end
          end

          ::Thrift.register_canonical_name_extractor self
        end
      end
    end
  end
end
