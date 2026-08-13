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

describe Thrift::Processor do
  class EmptyArgs
    include Thrift::Struct_Union
    include Thrift::Struct

    FIELDS = {}.freeze

    def struct_fields
      FIELDS
    end

    def validate; end
  end

  class EmptyResult
    include Thrift::Struct_Union
    include Thrift::Struct

    FIELDS = {}.freeze

    def struct_fields
      FIELDS
    end

    def validate; end
  end

  class LegacyProcessor
    include Thrift::Processor

    def process_ping(seqid, iprot, oprot)
      args = read_args(iprot, EmptyArgs)
      result = @middleware.handle_binary('ping', args) do
        @handler.ping
        EmptyResult.new
      end

      write_result(result, oprot, 'ping', seqid)
    end

    def process_notify(_seqid, iprot, _oprot)
      args = read_args(iprot, EmptyArgs)

      @middleware.handle_unary('notify', args) do
        @handler.notify
      end
    end
  end

  class CurrentProcessor
    include Thrift::Processor

    METHODS = {
      'ping' => {
        args_klass:   EmptyArgs,
        result_klass: EmptyResult,
        args:         [],
        exceptions:   {},
        oneway:       false
      }
    }.freeze
  end

  def request_protocol(name, type = Thrift::MessageTypes::CALL, seqid = 17)
    trans = Thrift::MemoryBufferTransport.new
    protocol = Thrift::BinaryProtocol.new(trans)
    protocol.write_message_begin(name, type, seqid)
    EmptyArgs.new.write(protocol)
    protocol.write_message_end

    Thrift::BinaryProtocol.new(trans)
  end

  def output_protocol
    trans = Thrift::MemoryBufferTransport.new

    [Thrift::BinaryProtocol.new(trans), trans]
  end

  def read_exception(protocol)
    name, type, seqid = protocol.read_message_begin
    exception = Thrift::ApplicationException.new
    exception.read(protocol)
    protocol.read_message_end

    {
      name:           name,
      type:           type,
      seqid:          seqid,
      exception_type: exception.type,
      message:        exception.message
    }
  end

  it 'processes requests with a legacy generated stub' do
    handler = double('handler', ping: nil)
    processor = LegacyProcessor.new(handler)
    oprot, trans = output_protocol

    expect(processor.process(request_protocol('ping'), oprot)).to eq(true)
    expect(oprot.read_message_begin).to eq(
      ['ping', Thrift::MessageTypes::REPLY, 17]
    )

    EmptyResult.new.read(oprot)
    oprot.read_message_end

    expect(trans.available).to eq(0)
  end

  it 'writes internal errors raised by a legacy generated stub' do
    handler = double('handler')
    allow(handler).to receive(:ping).and_raise(StandardError, 'boom')
    processor = LegacyProcessor.new(handler)
    oprot, = output_protocol

    expect(processor.process(request_protocol('ping'), oprot)).to eq(true)
    expect(read_exception(oprot)).to eq(
      name:           'ping',
      type:           Thrift::MessageTypes::EXCEPTION,
      seqid:          17,
      exception_type: Thrift::ApplicationException::INTERNAL_ERROR,
      message:        'Internal error processing ping: StandardError: boom'
    )
  end

  it 'raises errors from a legacy oneway method without writing a response' do
    handler = double('handler')
    allow(handler).to receive(:notify).and_raise(StandardError, 'boom')
    processor = LegacyProcessor.new(handler)
    oprot, trans = output_protocol

    expect do
      processor.process(
        request_protocol('notify', Thrift::MessageTypes::ONEWAY),
        oprot
      )
    end.to raise_error(StandardError, 'boom')
    expect(trans.available).to eq(0)
  end

  it 'writes an unknown method exception' do
    processor = CurrentProcessor.new(double('handler'))
    oprot, = output_protocol

    expect(
      processor.process(
        request_protocol('missing', Thrift::MessageTypes::CALL, 4),
        oprot
      )
    ).to eq(false)
    expect(read_exception(oprot)).to eq(
      name:           'missing',
      type:           Thrift::MessageTypes::EXCEPTION,
      seqid:          4,
      exception_type: Thrift::ApplicationException::UNKNOWN_METHOD,
      message:        'Unknown function missing'
    )
  end

  it 'writes internal errors raised by a current processor' do
    handler = double('handler')
    allow(handler).to receive(:ping).and_raise(StandardError, 'boom')
    processor = CurrentProcessor.new(handler)
    oprot, = output_protocol

    expect(processor.process(request_protocol('ping'), oprot)).to eq(true)
    expect(read_exception(oprot)).to eq(
      name:           'ping',
      type:           Thrift::MessageTypes::EXCEPTION,
      seqid:          17,
      exception_type: Thrift::ApplicationException::INTERNAL_ERROR,
      message:        'Internal error processing ping: StandardError: boom'
    )
  end

  it 'preserves application exceptions raised by a current processor' do
    handler = double('handler')
    allow(handler).to receive(:ping).and_raise(
      Thrift::ApplicationException.new(
        Thrift::ApplicationException::PROTOCOL_ERROR,
        'invalid request'
      )
    )
    processor = CurrentProcessor.new(handler)
    oprot, = output_protocol

    expect(processor.process(request_protocol('ping'), oprot)).to eq(true)
    expect(read_exception(oprot)).to eq(
      name:           'ping',
      type:           Thrift::MessageTypes::EXCEPTION,
      seqid:          17,
      exception_type: Thrift::ApplicationException::PROTOCOL_ERROR,
      message:        'invalid request'
    )
  end
end
