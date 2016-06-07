require 'statsd'

class NullStatsd
  def method_missing(*)
  end
end

module Thrift
  module Metrics
    module Middleware
      class Timing
        class << self
          def send_duration(name, t0)
            Metrics.client.timing(
              "#{name}.duration",
              ((Time.now - t0) * 1_000).to_i
            )
          end

          def instrument(name, &block)
            t0 = Time.now

            begin
              r = block.call
              send_duration name, t0
              r
            rescue => e
              send_duration name, t0
              raise e
            end
          end
        end
      end

      class Exception
        class << self
          def instrument(name, &block)
            begin
              r = block.call
              Metrics.client.increment("#{name}.success")
              r
            rescue => e
              Metrics.client.increment("#{name}.exceptions.#{e.class.name.downcase}")
              raise e
            end
          end
        end
      end
    end

    MIDDLEWARES = [Middleware::Timing, Middleware::Exception]

    class << self
      def instrument(name, &block)
        MIDDLEWARES.reduce(block) do |acc, cur|
          Proc.new { cur.instrument(name) { acc.call } }
        end.call
      end

      def client
        @client ||= begin
                      if ENV['STATSD_URL']
                        ip, port = ENV['STATSD_URL'].split(':')
                        Statsd.new ip, port.to_i
                      else
                        NullStatsd.new
                      end
                    end
      end
    end
  end
end
