module Thrift
  module Middleware
    DEFAUL = Noop

    class Noop
      def self.build(*_args)
        Noop.new
      end

      def handle_binary(_, &block)
        block.call
      end

      def handle_unary(_, &block)
        block.call
      end
    end
  end
