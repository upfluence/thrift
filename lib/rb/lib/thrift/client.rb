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
    # stream_handshake_timeout and stream_idle_timeout are handed to the
    # streams of the client, see DEFAULT_STREAM_HANDSHAKE_TIMEOUT and
    # DEFAULT_STREAM_IDLE_TIMEOUT. A value of zero or less disables them.
    attr_accessor :stream_handshake_timeout, :stream_idle_timeout

    def initialize(iprot, oprot = nil)
      @iprot = iprot
      @oprot = oprot || iprot
      @stream_handshake_timeout = DEFAULT_STREAM_HANDSHAKE_TIMEOUT
      @stream_idle_timeout = DEFAULT_STREAM_IDLE_TIMEOUT
      @seqid = 0
      @mutex = Mutex.new
      @cond = ConditionVariable.new
      @ready = true
    end

    def send_message(name, args_class, args = {})
      @seqid += 1
      @oprot.write_message_begin(name, MessageTypes::CALL, @seqid)
      data = args_class.new
      args.each { |k, v| data.send("#{k}=", v) }
      send_message_instance(data)
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
      Thrift.next_message(@iprot)
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

  # BaseClient serializes the calls of its protocol, and hands it over to a
  # stream for as long as it lasts: the calls wait for the stream to be over.
  #
  # A subclass can wrap what is done for every call by overriding around_call,
  # and refuse the streams by overriding ensure_streaming. The block given to
  # the methods opening a stream is called once the stream is over, whether it
  # ended or failed, after the client was released.
  class BaseClient
    include Client

    def call_unary(name, req)
      perform(:unary) do
        @seqid += 1
        @oprot.write_message_begin(name, Thrift::MessageTypes::ONEWAY, @seqid)
        send_message_instance(req)

        nil
      end
    end

    def call_binary(name, req, resp_klass)
      perform(:binary) { rpc_call(name, req, resp_klass) }
    end

    def stream_client(name, req, resp_klass, sink_klass, &on_close)
      stream_open(name, req, resp_klass, on_close) do |resp, unlock|
        stream = TOutboundStream.new(
          @iprot, @oprot, name, @seqid, sink_klass,
          MessageTypes::CLIENT_STREAM_MESSAGE,
          unlock,
          **stream_timeouts
        )

        [stream, [resp, stream]]
      end
    end

    def stream_server(name, req, resp_klass, sink_klass, &on_close)
      stream_open(name, req, resp_klass, on_close) do |resp, unlock|
        stream = TInboundStream.new(
          @iprot, @oprot, name, @seqid, sink_klass,
          MessageTypes::SERVER_STREAM_MESSAGE,
          unlock,
          **stream_timeouts
        )

        [stream, [resp, stream]]
      end
    end

    def stream_bidi(name, req, resp_klass, stream_klass, sink_klass, &on_close)
      stream_open(name, req, resp_klass, on_close) do |resp, unlock|
        stream = TBidiStream.new(
          @iprot, @oprot, name, @seqid,
          stream_klass, sink_klass,
          MessageTypes::SERVER_STREAM_MESSAGE,
          MessageTypes::CLIENT_STREAM_MESSAGE,
          unlock,
          **stream_timeouts
        )

        [
          stream,
          [resp, TBidiInboundStream.new(stream), TBidiOutboundStream.new(stream)]
        ]
      end
    end

    protected

    # around_call wraps every call, and the call opening a stream, whose kind
    # is :unary, :binary or :stream. It is called with the client claimed,
    # before the message is written.
    def around_call(_kind)
      yield
    end

    # ensure_streaming is called before a stream is opened, and raises if the
    # client can not carry it.
    def ensure_streaming; end

    private

    def stream_timeouts
      {
        handshake_timeout: @stream_handshake_timeout,
        idle_timeout: @stream_idle_timeout
      }
    end

    # perform waits for the client, and makes the call while it is held.
    def perform(kind, &block)
      @mutex.synchronize do
        @cond.wait(@mutex) until @ready

        around_call(kind, &block)
      end
    end

    # claim takes the client for a stream, which gives it back with finish_call.
    def claim
      @mutex.synchronize do
        @cond.wait(@mutex) until @ready

        @ready = false
      end
    end

    # stream_unlock is what the stream calls once it is over.
    def stream_unlock(on_close)
      return method(:finish_call) unless on_close

      lambda do
        finish_call
        on_close.call
      end
    end

    # stream_open claims the client and opens the stream with the call: the
    # block gets the reply and what to call once the stream is over, builds the
    # stream and returns it along with what the caller gets.
    #
    # The client is released if the stream can not be opened, and its
    # connection closed unless the server answered, as it could not be told
    # from one holding the replies of a stream that was half opened.
    def stream_open(name, req, resp_klass, on_close)
      ensure_streaming
      claim

      begin
        resp = around_call(:stream) { rpc_call(name, req, resp_klass) }
        stream, res = yield(resp, stream_unlock(on_close))

        stream.ready

        res
      rescue StandardError => e
        @oprot.trans.close unless e.is_a?(ApplicationException)
        finish_call

        raise e
      end
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
