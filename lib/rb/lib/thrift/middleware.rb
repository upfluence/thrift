module Thrift
  module Middleware
    class NopMiddleware
      def handle_binary(_mth, args = {}, &block)
        block.call(args)
      end

      def handle_unary(_mth, args = {},  &block)
        block.call(args)
      end

      def handle_bidi_stream(_mth, args, istream, ostream, &block)
        block.call(args, istream, ostream)
      end

      def handle_inbound_stream(_mth, args, istream, &block)
        block.call(args, istream)
      end

      def handle_outbound_stream(_mth, args, ostream, &block)
        block.call(args, ostream)
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

      def handle_bidi_stream(_mth, args, istream, ostream, &block)
        @middlewares.reverse.reduce(block) do |acc, m|
          Proc.new do |args, istream, ostream|
            m.handle_bidi_stream(mth, args, istream, ostream) do |args, istream, ostream|
              acc.call(args, istream, ostream)
            end
          end
        end.call(args, istream, ostream)
      end

      def handle_inbound_stream(_mth, args, istream, &block)
        @middlewares.reverse.reduce(block) do |acc, m|
          Proc.new do |args, istream|
            m.handle_bidi_stream(mth, args, istream) do |args, istream|
              acc.call(args, istream)
            end
          end
        end.call(args, istream)
      end

      def handle_outbound_stream(_mth, args, ostream, &block)
        @middlewares.reverse.reduce(block) do |acc, m|
          Proc.new do |args, ostream|
            m.handle_bidi_stream(mth, args, ostream) do |args, ostream|
              acc.call(args, ostream)
            end
          end
        end.call(args, ostream)
      end
    end

    NOP_MIDDLEWARE = NopMiddleware.new

    class << self
      def wrap(middlewares)
        case middlewares.length
        when 0
          NOP_MIDDLEWARE
        when 1
          middlewares.first
        else
          MultiMiddleware.new(middlewares)
        end
      end
    end
  end
end
