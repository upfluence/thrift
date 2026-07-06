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

require 'spec_helper'

describe 'Streaming' do
  # ---------------------------------------------------------------------------
  # Helpers shared across streaming processor and client specs
  # ---------------------------------------------------------------------------

  # A minimal args class whose instances know their field.
  class StreamArgs
    attr_accessor :value

    def initialize(value: nil)
      @value = value
    end

    def read(iprot)
      @value = iprot.read_string
      iprot.read_field_end
      iprot.read_field_stop
    end

    def write(oprot)
      oprot.write_string(@value.to_s)
    end
  end

  # A minimal result class that `execute` builds.
  class StreamResult
    attr_accessor :success

    def write(oprot)
      oprot.write_string(@success.to_s)
    end
  end

  # Stub middleware that just calls the next handler.
  class NopMiddleware
    def handle_binary(name, args)
      yield args
    end

    def handle_unary(name, args)
      yield args
    end

    def handle_outbound_stream(name, args, stream)
      yield args, stream
    end

    def handle_inbound_stream(name, args, sink)
      yield args, sink
    end

    def handle_bidi_stream(name, args, sink, stream)
      yield args, sink, stream
    end
  end

  # A stub handler for outbound (server-push) streaming.
  class OutboundHandler
    attr_reader :received_args, :received_stream

    def test_stream(args, stream)
      @received_args = args
      @received_stream = stream
      StreamResult.new.tap { |r| r.success = 'ok' }
    end
  end

  # A stub handler for inbound (client-push) streaming.
  class InboundHandler
    attr_reader :received_args, :received_sink

    def test_sink(args, sink)
      @received_args = args
      @received_sink = sink
      StreamResult.new.tap { |r| r.success = 'ok' }
    end
  end

  # A stub handler for bidirectional streaming.
  class BidiHandler
    attr_reader :received_args, :received_sink, :received_stream

    def test_bidi(args, stream, sink)
      @received_args = args
      @received_stream = stream
      @received_sink = sink
      StreamResult.new.tap { |r| r.success = 'ok' }
    end
  end

  # Build a minimal info hash for a streaming processor.
  def outbound_info(handler_sym: :test_stream)
    {
      args:         [:value],
      args_klass:   StreamArgs,
      result_klass: StreamResult,
      void_result:  false,
      exceptions:   {},
      stream_klass: double('StreamKlass'),
    }
  end

  def inbound_info
    {
      args:         [:value],
      args_klass:   StreamArgs,
      result_klass: StreamResult,
      void_result:  false,
      exceptions:   {},
      sink_klass:   double('SinkKlass'),
    }
  end

  def bidi_info
    {
      args:         [:value],
      args_klass:   StreamArgs,
      result_klass: StreamResult,
      void_result:  false,
      exceptions:   {},
      sink_klass:   double('SinkKlass'),
      stream_klass: double('StreamKlass'),
    }
  end

  # ---------------------------------------------------------------------------
  # Thrift::Processor::OutboundStreamProcessor
  # ---------------------------------------------------------------------------

  describe Thrift::Processor::OutboundStreamProcessor do
    let(:handler)    { OutboundHandler.new }
    let(:middleware) { NopMiddleware.new }
    let(:iprot)      { double('MockIProt') }
    let(:oprot)      { double('MockOProt') }

    subject do
      described_class.new(
        'test_stream',
        outbound_info,
        middleware,
        handler
      )
    end

    it 'initialises with the stream_klass from info' do
      expect(subject.instance_variable_get(:@stream_klass)).not_to be_nil
    end

    it 'reads args, calls the handler, writes the result, then calls stream.ready' do
      # Set up iprot to return a minimal args read.
      allow(iprot).to receive(:read_string).and_return('hello')
      allow(iprot).to receive(:read_field_end)
      allow(iprot).to receive(:read_field_stop)
      allow(iprot).to receive(:read_message_end)

      # Set up oprot to accept the REPLY frame.
      allow(oprot).to receive(:write_message_begin)
      allow(oprot).to receive(:write_string)
      allow(oprot).to receive(:write_message_end)
      allow(oprot).to receive(:trans).and_return(double('trans', flush: nil))

      # Capture the TOutboundStream that the processor creates.
      stream_double = instance_double(
        Thrift::TOutboundStream,
        ready: nil,
        close: nil
      )

      allow(Thrift::TOutboundStream).to receive(:new).and_return(stream_double)

      # Simulate immediate closure so the processor doesn't block in #wait.
      allow(subject).to receive(:wait)

      result = subject.process(1, iprot, oprot)

      expect(result).to be true
      expect(handler.received_stream).to eq(stream_double)
      expect(stream_double).to have_received(:ready)
    end
  end

  # ---------------------------------------------------------------------------
  # Thrift::Processor::InboundStreamProcessor
  # ---------------------------------------------------------------------------

  describe Thrift::Processor::InboundStreamProcessor do
    let(:handler)    { InboundHandler.new }
    let(:middleware) { NopMiddleware.new }
    let(:iprot)      { double('MockIProt') }
    let(:oprot)      { double('MockOProt') }

    subject do
      described_class.new(
        'test_sink',
        inbound_info,
        middleware,
        handler
      )
    end

    it 'initialises with sink_klass from info' do
      expect(subject.instance_variable_get(:@sink_klass)).not_to be_nil
    end

    it 'reads args, calls the handler, writes the result, then calls sink.ready' do
      allow(iprot).to receive(:read_string).and_return('world')
      allow(iprot).to receive(:read_field_end)
      allow(iprot).to receive(:read_field_stop)
      allow(iprot).to receive(:read_message_end)

      allow(oprot).to receive(:write_message_begin)
      allow(oprot).to receive(:write_string)
      allow(oprot).to receive(:write_message_end)
      allow(oprot).to receive(:trans).and_return(double('trans', flush: nil))

      sink_double = instance_double(
        Thrift::TInboundStream,
        ready: nil,
        close: nil
      )

      allow(Thrift::TInboundStream).to receive(:new).and_return(sink_double)
      allow(subject).to receive(:wait)

      result = subject.process(1, iprot, oprot)

      expect(result).to be true
      expect(handler.received_sink).to eq(sink_double)
      expect(sink_double).to have_received(:ready)
    end
  end

  # ---------------------------------------------------------------------------
  # Thrift::Processor::BidiStreamProcessor
  # ---------------------------------------------------------------------------

  describe Thrift::Processor::BidiStreamProcessor do
    let(:handler)    { BidiHandler.new }
    let(:middleware) { NopMiddleware.new }
    let(:iprot)      { double('MockIProt') }
    let(:oprot)      { double('MockOProt') }

    subject do
      described_class.new(
        'test_bidi',
        bidi_info,
        middleware,
        handler
      )
    end

    it 'initialises with both sink_klass and stream_klass from info' do
      expect(subject.instance_variable_get(:@sink_klass)).not_to be_nil
      expect(subject.instance_variable_get(:@stream_klass)).not_to be_nil
    end

    it 'reads args, calls the handler with sink and stream, writes result, calls bidi_stream.ready' do
      allow(iprot).to receive(:read_string).and_return('bidi')
      allow(iprot).to receive(:read_field_end)
      allow(iprot).to receive(:read_field_stop)
      allow(iprot).to receive(:read_message_end)

      allow(oprot).to receive(:write_message_begin)
      allow(oprot).to receive(:write_string)
      allow(oprot).to receive(:write_message_end)
      allow(oprot).to receive(:trans).and_return(double('trans', flush: nil))

      bidi_double = instance_double(
        Thrift::TBidiStream,
        ready: nil,
        close: nil
      )

      allow(Thrift::TBidiStream).to receive(:new).and_return(bidi_double)
      allow(Thrift::TBidiInboundStream).to receive(:new).with(bidi_double).and_call_original
      allow(Thrift::TBidiOutboundStream).to receive(:new).with(bidi_double).and_call_original
      allow(subject).to receive(:wait)

      result = subject.process(1, iprot, oprot)

      expect(result).to be true
      expect(handler.received_stream).to be_a(Thrift::TBidiOutboundStream)
      expect(handler.received_sink).to be_a(Thrift::TBidiInboundStream)
      expect(bidi_double).to have_received(:ready)
    end
  end

  # ---------------------------------------------------------------------------
  # Thrift::BaseClient streaming methods
  # ---------------------------------------------------------------------------

  describe Thrift::BaseClient do
    class StreamingClientSpec < Thrift::BaseClient; end

    let(:mock_trans) { double('MockTransport', open?: true, flush: nil) }
    let(:oprot) do
      double('MockOProt').tap do |p|
        allow(p).to receive(:trans).and_return(mock_trans)
      end
    end
    let(:iprot) { double('MockIProt') }

    let(:client) { StreamingClientSpec.new(iprot, oprot) }

    # Stub the rpc_call internals so tests only concern themselves with the
    # stream objects, not protocol-level details.
    def stub_rpc_call(client, method_name, resp_klass)
      allow(oprot).to receive(:write_message_begin)
      allow(oprot).to receive(:write_message_end)

      result_obj = resp_klass.new

      allow(iprot).to receive(:read_message_begin).and_return(
        [method_name, Thrift::MessageTypes::REPLY, client.instance_variable_get(:@seqid) + 1]
      )
      allow(iprot).to receive(:read_message_end)
      allow(result_obj).to receive(:read)

      result_obj
    end

    describe '#stream_client (outbound from client perspective)' do
      it 'returns [resp, TOutboundStream] and marks the client not ready' do
        resp_klass = double('RespKlass')
        resp_obj   = double('resp_obj')
        allow(resp_klass).to receive(:new).and_return(resp_obj)
        allow(resp_obj).to receive(:read)
        allow(resp_obj).to receive(:write)
        allow(oprot).to receive(:write_message_begin)
        allow(oprot).to receive(:write_message_end)
        allow(iprot).to receive(:read_message_begin).and_return(
          ['test', Thrift::MessageTypes::REPLY, 1]
        )
        allow(iprot).to receive(:read_message_end)

        sink_klass   = double('SinkKlass')
        stream_double = instance_double(Thrift::TOutboundStream, ready: nil)

        allow(Thrift::TOutboundStream).to receive(:new).and_return(stream_double)

        resp, stream = client.stream_client('test', double('req', write: nil), resp_klass, sink_klass)

        expect(stream).to eq(stream_double)
        expect(stream_double).to have_received(:ready)
        expect(client.instance_variable_get(:@ready)).to be false
      end
    end

    describe '#stream_server (inbound to client perspective)' do
      it 'returns [resp, TInboundStream] and marks the client not ready' do
        resp_klass = double('RespKlass')
        resp_obj   = double('resp_obj')
        allow(resp_klass).to receive(:new).and_return(resp_obj)
        allow(resp_obj).to receive(:read)
        allow(resp_obj).to receive(:write)
        allow(oprot).to receive(:write_message_begin)
        allow(oprot).to receive(:write_message_end)
        allow(iprot).to receive(:read_message_begin).and_return(
          ['test', Thrift::MessageTypes::REPLY, 1]
        )
        allow(iprot).to receive(:read_message_end)

        sink_klass    = double('SinkKlass')
        stream_double = instance_double(Thrift::TInboundStream, ready: nil)

        allow(Thrift::TInboundStream).to receive(:new).and_return(stream_double)

        resp, stream = client.stream_server('test', double('req', write: nil), resp_klass, sink_klass)

        expect(stream).to eq(stream_double)
        expect(stream_double).to have_received(:ready)
        expect(client.instance_variable_get(:@ready)).to be false
      end
    end

    describe '#stream_bidi' do
      it 'returns [resp, TBidiInboundStream, TBidiOutboundStream] and marks the client not ready' do
        resp_klass = double('RespKlass')
        resp_obj   = double('resp_obj')
        allow(resp_klass).to receive(:new).and_return(resp_obj)
        allow(resp_obj).to receive(:read)
        allow(resp_obj).to receive(:write)
        allow(oprot).to receive(:write_message_begin)
        allow(oprot).to receive(:write_message_end)
        allow(iprot).to receive(:read_message_begin).and_return(
          ['test', Thrift::MessageTypes::REPLY, 1]
        )
        allow(iprot).to receive(:read_message_end)

        stream_klass  = double('StreamKlass')
        sink_klass    = double('SinkKlass')
        bidi_double   = instance_double(Thrift::TBidiStream, ready: nil)

        allow(Thrift::TBidiStream).to receive(:new).and_return(bidi_double)

        resp, inbound, outbound = client.stream_bidi(
          'test',
          double('req', write: nil),
          resp_klass,
          stream_klass,
          sink_klass
        )

        expect(inbound).to be_a(Thrift::TBidiInboundStream)
        expect(outbound).to be_a(Thrift::TBidiOutboundStream)
        expect(bidi_double).to have_received(:ready)
        expect(client.instance_variable_get(:@ready)).to be false
      end
    end

    describe 'finish_call' do
      it 'sets @ready back to true and signals the condition variable' do
        # Directly mark the client as not ready (as stream_call would do).
        client.instance_variable_set(:@ready, false)
        client.send(:finish_call)

        expect(client.instance_variable_get(:@ready)).to be true
      end
    end
  end
end
