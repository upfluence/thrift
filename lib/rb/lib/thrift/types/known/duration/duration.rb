require 'thrift/types/known/duration/duration_types'

module Thrift
  module Types
    module Known
      module Duration
        class << self
          def from_number(v)
            Duration.from_number(v)
          end
        end

        class Duration
          class << self
            def from_number(v)
              Duration.new(seconds: v.to_i, nanos: ((v % 1) * 1e9).to_i)
            end
          end

          def to_number
            seconds + nanos / 1e9
          end
        end
      end
    end
  end
end
