#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements. See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership. The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License. You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied. See the License for the
# specific language governing permissions and limitations
# under the License.
#

module Thrift
  module Processor
    class BaseProcessorFunction
      def initialize(fname, middleware, args_klass, method)
        @fname = fname
        @middleware = middleware
        @args_klass = args_klass
        @method = method
      end

      protected

      def read_args(iprot)
        args = @args_klass.new

        args.read(iprot)
        iprot.read_message_end

        args
      end
    end

    class BinaryProcessorFunction < BaseProcessorFunction
      def process(seqid, iprot, oprot)
        execute(seqid, iprot, oprot)

        true
      end

      private

      def execute(seqid, iprot, oprot)
        args = read_args(iprot)

        result = @middleware.handle_binary(@fname, args) do |args|
          @method.call(args)
        end

        write_result(result, oprot, seqid)
      rescue => e
        write_exception(e, oprot, seqid)
      end

      def write_exception(exception, oprot, seqid)
        oprot.write_message_begin(@fname, MessageTypes::EXCEPTION, seqid)

        unless exception.is_a? ApplicationException
          exception = ApplicationException.new(
            ApplicationException::INTERNAL_ERROR,
            "Internal error processing #{@fname}: #{exception.class}: #{exception}"
          )
        end

        exception.write(oprot)
        oprot.write_message_end
        oprot.trans.flush
      end

      def write_result(result, oprot, seqid)
        oprot.write_message_begin(@fname, MessageTypes::REPLY, seqid)
        result.write(oprot)
        oprot.write_message_end
        oprot.trans.flush
      end
    end

    class UnaryProcessorFunction < BaseProcessorFunction
      def process(_seqid, iprot, _oprot)
        args = read_args(iprot)

        @middleware.handle_unary(@fname, args) do |args|
          @method.call(args)
        end

        true
      end
    end

    def initialize(handler, middlewares = [])
      @handler = handler
      @middleware = Middleware.wrap(middlewares)

      @processors = if self.class.const_defined? :METHODS
                      self.class::METHODS.reduce({}) do |acc, (name, info)|
                        acc.merge(name => build_processor(name, info))
                      end
                    else
                      {}
                    end
    end

    def process(iprot, oprot)
      name, type, seqid = iprot.read_message_begin

      mth = "process_#{name}"
      send(mth, seqid, iprot, oprot) if self.class.method_defined? mth

      (
        @processors[name] || UnkwonFunctionProcessor.new(name)
      ).process(seqid, iprot, oprot)
    end

    def build_processor(name, info)
      if info[:result_klass].nil?
        UnaryProcessor
      elsif info[:stream_klass].nil? && info[:sink_klass].nil?
        BinaryProcessor
      elsif info[:sink_klass].nil?
        OutboundStreamProcessor
      elsif info[:stream_klass].nil?
        InboundStreamProcessor
      else
        BidiStreamProcessor
      end.new(name, info, @middleware, @handler)
    end

    class BaseProcessor
      def initialize(name, args_class)
        @name = name
        @args_class = args_class
      end

      protected

      def read_args(iprot)
        args = @args_class.new
        args.read(iprot)
        iprot.read_message_end
        args
      end

      def write_exception(exception, oprot, name, seqid)
        oprot.write_message_begin(name, MessageTypes::EXCEPTION, seqid)

        unless exception.is_a? ApplicationException
          exception = ApplicationException.new(
            ApplicationException::INTERNAL_ERROR,
            "Internal error processing #{name}: #{exception.class}: #{exception}"
          )
        end

        exception.write(oprot)
        oprot.write_message_end
        oprot.trans.flush
      end

      def write_result(result, oprot, seqid)
        oprot.write_message_begin(@name, MessageTypes::REPLY, seqid)
        result.write(oprot)
        oprot.write_message_end
        oprot.trans.flush
      end
    end

    class UnaryProcessor
      def initialize(name, info, middleware, handler)
        @middleware = middleware
        @handler = handler
        @arg_keys = info[:args]

        super name, info[:args_klass]
      end

      def process(_seqid, iprot, _oprot)
        @middleware.handle_unary(@name, read_args(iprot)) do |args|
          @handler.send(@name, *@arg_keys.map { |k| args.send k })
          nil
        end

        true
      end
    end

    class BinaryProcessor < BaseProcessor
      def initialize(name, info, middleware, handler)
        @middleware = middleware
        @handler = handler
        @arg_keys = info[:args]
        @void_result = info[:void_result]
        @result_klass = info[:result_klass]
        @exceptions = info[:exceptions]

        super name, info[:args_klass]
      end

      def process(seqid, iprot, oprot)
        res = @middleware.handle_binary(@name, read_args(iprot)) do |args|
          execute(args)
        end

        write_result(res, oprot, seqid)

        true
      end

      protected

      def execute(args, *extra_args)
        res = @result_klass.new

        begin
          s = @handler.send(
            @name,
            *@arg_keys.map { |k| args.send k },
            *extra_args
          )

          res.success = s unless @void_result
        rescue => e
          k, = @exceptions.find { |(_, klass)| e.is_a? klass }

          raise e unless k

          res.send "#{k}=", e
        end

        res
      end
    end

    class BaseStreamProcessor < BinaryProcessor
      def initialize(name, info, middleware, handler)
        @mutex = Mutex.new
        @cond = ConditionVariable.new
        @closed = false

        super name, info, middleware, handler
      end

      protected

      def close
        @mutex.synchronize do
          @closed = true
          @cond.broadcast
        end
      end

      def wait
        @mutex.synchronize do
          @cond.wait(@mutex) unless @closed
        end
      end
    end

    class OutboundStreamProcessor < BaseStreamProcessor
      def initialize(name, info, middleware, handler)
        @stream_klass = info[:stream_klass]

        super name, info, middleware, handler
      end

      def process(seqid, iprot, oprot)
        @closed = false
        stream = TOutboundStream.new(
          iprot, oprot, @name, seqid, @stream_klass,
          MessageTypes::SERVER_STREAM_MESSAGE,
          method(:close)
        )
        res = @middleware.handle_outbound_stream(
          @name, read_args(iprot), stream
        ) { |args, stream| execute(args, stream) }

        write_result(res, oprot, seqid)
        stream.ready

        wait

        true
      end
    end

    class InboundStreamProcessor < BaseStreamProcessor
      def initialize(name, info, middleware, handler)
        @sink_klass = info[:sink_klass]

        super name, info, middleware, handler
      end

      def process(seqid, iprot, oprot)
        @closed = false
        stream = TInboundStream.new(
          iprot, oprot, @name, seqid, @sink_klass,
          MessageTypes::CLIENT_STREAM_MESSAGE,
          method(:close)
        )

        res = @middleware.handle_inbound_stream(
          @name, read_args(iprot), stream
        ) { |args, sink| execute(args, sink) }


        write_result(res, oprot, seqid)
        stream.ready

        wait

        true
      end
    end

    class BidiStreamProcessor < BaseStreamProcessor
      def initialize(name, info, middleware, handler)
        @sink_klass = info[:sink_klass]
        @stream_klass = info[:stream_klass]

        super name, info, middleware, handler
      end

      def process(seqid, iprot, oprot)
        @closed = false

        bidi_stream = TBidiStream.new(
          iprot, oprot, @name, seqid, @sink_klass, @stream_klass,
          MessageTypes::CLIENT_STREAM_MESSAGE,
          MessageTypes::SERVER_STREAM_MESSAGE,
          method(:close)
        )

        res = @middleware.handle_bidi_stream(
          @name, read_args(iprot),
          TBidiInboundStream.new(bidi_stream),
          TBidiOutboundStream.new(bidi_stream)
        ) do |args, sink, stream|
          execute(args, stream, sink)
        end

        write_result(res, oprot, seqid)
        bidi_stream.ready

        wait

        true
      end
    end

    class UnkwonFunctionProcessor < BaseProcessor
      def initialize(name)
        super name, nil
      end

      def process(seqid, iprot, oprot)
        iprot.skip(Types::STRUCT)
        iprot.read_message_end
        write_exception(
          ApplicationException.new(
            ApplicationException::UNKNOWN_METHOD,
            'Unknown function ' + @name,
          ),
          oprot,
          @name,
          seqid
        )

        false
      end
    end
  end
end
