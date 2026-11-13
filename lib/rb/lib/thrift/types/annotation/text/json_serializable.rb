require 'json'

module Thrift
  module Types
    module Annotation
      module Text
        # JSONSerializable renders a struct annotated with Format as its text
        # form in JSON. It is opt-in, annotating a struct does not change its
        # JSON by itself:
        #
        #   Base::Ownership::Ownership.include(
        #     Thrift::Types::Annotation::Text::JSONSerializable
        #   )
        #
        # as_json covers the encoders relying on it, such as ActiveSupport,
        # to_json the ones calling it, such as Oj in compat mode.
        module JSONSerializable
          def as_json(*)
            Text.dump(self)
          end

          def to_json(*args)
            as_json.to_json(*args)
          end
        end
      end
    end
  end
end
