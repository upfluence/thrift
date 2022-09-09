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
#

module Thrift
  module Client
    def initialize(iprot, oprot = nil)
      @iprot = iprot
      @oprot = oprot || iprot
      @seqid = 0
      @mutex = Mutex.new
      @cond = ConditionVariable.new
      @ready = true
    end

    def send_message_instance(data)
      begin
        data.write(@oprot)
      rescue StandardError => e
        @oprot.trans.close
        raise e
      end
      @oprot.write_message_end
      @oprot.trans.flush
    end

    def receive_message(result_klass, name=nil)
      fname, mtype, seqid = @iprot.read_message_begin
      handle_exception(mtype)

      raise ApplicationException.new(
        ApplicationException::BAD_SEQUENCE_ID,
        'out of seq'
      ) if seqid != @seqid

      raise ApplicationException.new(
        ApplicationException::INVALID_MESSAGE_TYPE,
        'invalid message type'
      ) if mtype != MessageTypes::REPLY

      raise ApplicationException.new(
        ApplicationException::WRONG_METHOD_NAME,
        'wrong method name'
      ) if !name.nil? && name != fname

      result = result_klass.new
      result.read(@iprot)
      @iprot.read_message_end
      result
    end

    def handle_exception(mtype)
      return if mtype != MessageTypes::EXCEPTION

      x = ApplicationException.new
      x.read(@iprot)
      @iprot.read_message_end
      raise x
    end
  end

  class BaseClient
    include Client

    def call_unary(name, req)
      @mutex.synchronize do
        @cond.wait(@mutex) until @ready

        @seqid += 1
        @oprot.write_message_begin(name, Thrift::MessageTypes::ONEWAY, @seqid)
        send_message_instance(req)
      end
    end

    def call_binary(name, req, resp_klass)
      @mutex.synchronize do
        @cond.wait(@mutex) until @ready

        rpc_call(name, req, resp_klass)
      end
    end

    def stream_client(name, req, resp_klass, sink_klass)
      resp = stream_call(name, req, resp_klass)
      stream = TOutboundStream.new(
        @iprot, @oprot, name, @seqid, sink_klass,
        MessageTypes::CLIENT_STREAM_MESSAGE,
        method(:finish_call)
      )

      stream.ready

      [resp, stream]
    end

    def stream_server(name, req, resp_klass, sink_klass)
      resp = stream_call(name, req, resp_klass)
      stream = TInboundStream.new(
        @iprot, @oprot, name, @seqid, sink_klass,
        MessageTypes::SERVER_STREAM_MESSAGE,
        method(:finish_call)
      )

      stream.ready

      [resp, stream]
    end

    def stream_bidi(name, req, resp_klass, stream_klass, sink_klass)
      resp = stream_call(name, req, resp_klass)
      stream = TBidiStream.new(
        @iprot, @oprot, name, @seqid,
        stream_klass, sink_klass,
        MessageTypes::SERVER_STREAM_MESSAGE,
        MessageTypes::CLIENT_STREAM_MESSAGE,
        method(:finish_call)
      )

      stream.ready

      [
        resp,
        TBidiInboundStream.new(stream),
        TBidiOutboundStream.new(stream)
      ]
    end

    private

    def stream_call(name, req, resp_klass)
      @mutex.synchronize do
        @cond.wait(@mutex) until @ready

        @ready = false
      end

      rpc_call(name, req, resp_klass)
    end

    def rpc_call(name, req, resp_klass)
      @seqid += 1
      @oprot.write_message_begin(name, Thrift::MessageTypes::CALL, @seqid)
      send_message_instance(req)
      receive_message(resp_klass)
    end

    def finish_call
      @mutex.synchronize do
        @ready = true
        @cond.signal
      end
    end
  end

  class << self
    def build_client(input)
      return BaseClient.new(input) if input.is_a? BaseProtocol

      input
    end
  end
end
