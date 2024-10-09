require 'thrift/types/known/timestamp/timestamp_types'

module Thrift
  module Types
    module Known
      module Timestamp
        class << self
          def from_time(v)
            Timestamp.from_time(v)
          end

          def now
            from_time(Time.now)
          end
        end

        class Timestamp
          class << self
            def from_time(v)
              Timestamp.new(seconds: v.to_i, nanos: v.nsec)
            end
          end

          def to_time
            Time.at(seconds, nanos, :nsec)
          end
        end
      end
    end
  end
end
