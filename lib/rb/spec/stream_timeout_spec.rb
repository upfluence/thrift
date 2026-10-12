require 'spec_helper'

describe 'Stream timeouts' do
  class TimeoutMessage
    def initialize(arg: nil)
      @arg = arg
    end

    def write(oprot)
      oprot.write_string(@arg.to_s)
    end
  end

  # A stream whose peer reads nothing and writes nothing: what it sends is
  # kept in a pipe, and nothing ever arrives.
  def silent_peer_stream(**timeouts)
    in_r, in_w = IO.pipe
    out_r, out_w = IO.pipe
    prot = Thrift::BinaryProtocol.new(Thrift::IOStreamTransport.new(in_r, out_w))

    @pipes = [in_r, in_w, out_r, out_w]

    Thrift::TOutboundStream.new(
      prot, prot, 'stream', 1, TimeoutMessage,
      Thrift::MessageTypes::SERVER_STREAM_MESSAGE, nil, **timeouts
    )
  end

  def wait_until(timeout = 2)
    deadline = Process.clock_gettime(Process::CLOCK_MONOTONIC) + timeout

    until yield
      return false if Process.clock_gettime(Process::CLOCK_MONOTONIC) > deadline

      sleep 0.01
    end

    true
  end

  around do |example|
    report = Thread.report_on_exception
    Thread.report_on_exception = false

    example.run
  ensure
    Thread.report_on_exception = report
    @pipes&.each { |io| io.close unless io.closed? }
  end

  describe 'handshake timeout' do
    it 'aborts a stream whose go away is never acknowledged' do
      stream = silent_peer_stream(handshake_timeout: 0.1, idle_timeout: 0)

      stream.ready

      closer = Thread.new { stream.close }

      expect(closer.join(2)).not_to be_nil
      expect(stream.aborted?).to be true
    end

    it 'does not abort a stream whose go away is acknowledged' do
      stream = silent_peer_stream(handshake_timeout: 0.2, idle_timeout: 0)

      stream.ready

      closer = Thread.new { stream.close }

      sleep 0.05
      stream.__send__(:mark_as_closed)

      expect(closer.join(2)).not_to be_nil
      sleep 0.3
      expect(stream.aborted?).to be false
    end

    it 'is disabled by a value of zero' do
      stream = silent_peer_stream(handshake_timeout: 0, idle_timeout: 0)

      stream.ready

      closer = Thread.new { stream.close }

      expect(closer.join(0.3)).to be_nil
      expect(stream.aborted?).to be false

      stream.__send__(:abort_stream)

      expect(closer.join(2)).not_to be_nil
    end
  end

  describe 'idle timeout' do
    it 'aborts a quiet stream' do
      stream = silent_peer_stream(handshake_timeout: 5, idle_timeout: 0.1)

      stream.ready

      expect(wait_until { stream.aborted? }).to be true
    end

    it 'is not triggered by a stream that sends messages' do
      stream = silent_peer_stream(handshake_timeout: 5, idle_timeout: 0.3)

      stream.ready

      5.times do
        sleep 0.1
        stream.send('foo')
      end

      expect(stream.aborted?).to be false
      expect(wait_until { stream.aborted? }).to be true
    end

    it 'is disabled by default' do
      stream = silent_peer_stream

      stream.ready
      sleep 0.2

      expect(stream.aborted?).to be false
    end
  end

  describe Thrift::BaseClient do
    class TimeoutClientSpec < Thrift::BaseClient; end

    let(:iprot) { double('MockIProt') }
    let(:oprot) do
      double('MockOProt').tap do |p|
        allow(p).to receive(:trans).and_return(double('trans', open?: true, flush: nil))
      end
    end
    let(:client) { TimeoutClientSpec.new(iprot, oprot) }

    before do
      resp_klass = double('RespKlass')
      resp_obj = double('resp_obj', read: nil, write: nil)

      allow(resp_klass).to receive(:new).and_return(resp_obj)
      allow(oprot).to receive(:write_message_begin)
      allow(oprot).to receive(:write_message_end)
      allow(iprot).to receive(:trans).and_return(double('itrans', next_message: nil))
      allow(iprot).to receive(:read_message_begin).and_return(
        ['test', Thrift::MessageTypes::REPLY, 1]
      )
      allow(iprot).to receive(:read_message_end)

      @resp_klass = resp_klass
    end

    it 'defaults to the default timeouts' do
      expect(client.stream_handshake_timeout).to eq(Thrift::DEFAULT_STREAM_HANDSHAKE_TIMEOUT)
      expect(client.stream_idle_timeout).to eq(Thrift::DEFAULT_STREAM_IDLE_TIMEOUT)
    end

    it 'hands its timeouts to the streams' do
      client.stream_handshake_timeout = 1
      client.stream_idle_timeout = 2

      expect(Thrift::TOutboundStream).to receive(:new).with(
        iprot, oprot, 'test', 1, anything, anything, anything,
        handshake_timeout: 1, idle_timeout: 2
      ).and_return(instance_double(Thrift::TOutboundStream, ready: nil))

      client.stream_client('test', double('req', write: nil), @resp_klass, double('Sink'))
    end
  end
end
