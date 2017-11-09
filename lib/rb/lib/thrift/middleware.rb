module Thrift
  module Middleware
    DEFAUL = Noop

    class Noop
      def self.build(*_args)
        Noop.new
      end

      def handle_binary(ctx, _mth, args = {}, &block)
        block.call(ctx, args)
      end

      def handle_unary(ctx, _mth, args = {},  &block)
        block.call(ctx, args)
      end
    end
  end
