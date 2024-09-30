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

      @functions = if self.class.const_defined?(:METHODS)
        self.class::METHODS.reduce({}) do |acc, (key, args)|
          klass = args[:oneway] ? UnaryProcessorFunction : BinaryProcessorFunction

          acc.merge key => klass.new(
            key,
            @middleware,
            args[:args_klass],
            method("execute_#{key}")
          )
        end
      end || {}
    end

    def process(iprot, oprot)
      name, _type, seqid = iprot.read_message_begin

      func = @functions[name]

      return func.process(seqid, iprot, oprot) if func

      # TODO: once all the stubs will be generated w thrift >=2.5 the next lines
      # can be deleted
      if respond_to?("process_#{name}")
        begin
          send("process_#{name}", seqid, iprot, oprot)
        rescue => e
          write_exception(e, oprot, name, seqid)
        end
        true
      else
        iprot.skip(Types::STRUCT)
        iprot.read_message_end
        write_exception(
          ApplicationException.new(
            ApplicationException::UNKNOWN_METHOD,
            'Unknown function ' + name,
          ),
          oprot,
          name,
          seqid
        )
        false
      end
    end

    def read_args(iprot, args_class)
      args = args_class.new
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

    def write_result(result, oprot, name, seqid)
      oprot.write_message_begin(name, MessageTypes::REPLY, seqid)
      result.write(oprot)
      oprot.write_message_end
      oprot.trans.flush
    end
  end

  class << self
    def build_processor_from_provider(klass, provider, handler)
      sdef = ServiceDefinition.new(klass)

      [
        { namespace: sdef.namespace, service: sdef.service },
        *sdef.legacy_names
      ].reduce(nil) do |acc, n|
        acc || begin
          provider.build(
            n[:namespace], n[:service], sdef.processor_class, handler
          )
        rescue ProcessorNotDefined
          nil
        end
      end || raise(ProcessorNotDefined)
    end
  end
end
