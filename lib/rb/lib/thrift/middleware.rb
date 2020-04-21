module Thrift
  module Middleware
    class NopMiddleware
      def handle_binary(_mth, args = {}, &block)
        block.call(args)
      end

      def handle_unary(_mth, args = {},  &block)
        block.call(args)
      end
    end

    class MultiMiddleware
      def initialize(middlewares)
        @middlewares = middlewares
      end

      def handle_unary(mth, args = {}, &block)
        @middlewares.reverse.reduce(block) do |acc, m|
          Proc.new do |args|
            m.handle_unary(mth, args) { |args| acc.call(args) }
          end
        end.call(args)
      end

      def handle_binary(mth, args = {}, &block)
        @middlewares.reverse.reduce(block) do |acc, m|
          Proc.new do |args|
            m.handle_binary(mth, args) { |args| acc.call(args) }
          end
        end.call(args)
      end
    end

    NOP_MIDDLEWARE = NopMiddleware.new
  end
end
