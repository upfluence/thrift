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

require File.join(File.dirname(__FILE__), '../test_helper')

require 'thrift'
require 'streaming_test'

# StreamingHandler implements the StreamingTest service for testing.
class StreamingTestServiceHandler
  # testClientStream: drain the sink in a background thread, return 0 immediately.
  def testClientStream(label, sink)
    Thread.new do
      loop do
        sink.receive
      rescue EOFError, StandardError
        break
      end
    end

    0
  end

  # testServerStream: push count strings of the form "{prefix}_{i}" in a background thread.
  def testServerStream(count, prefix, stream)
    Thread.new do
      count.times { |i| stream.send("#{prefix}_#{i}") }
      stream.close
    rescue StandardError
      stream.close rescue nil
    end

    nil
  end

  # testBidi: echo each received string back uppercased in a background thread.
  def testBidi(label, stream, sink)
    Thread.new do
      loop do
        msg = sink.receive
        stream.send(msg.upcase)
      rescue EOFError, StandardError
        break
      end
      stream.close rescue nil
    end

    nil
  end
end

# TestStreamingCrossTest exercises all three streaming RPC types using a real
# TSimpleServer over loopback TCP.  One server is started for the whole test
# class to keep setup overhead low.
class TestStreamingCrossTest < Test::Unit::TestCase
  PORT = begin
    require 'socket'
    # Pick a free port once at class load time.
    s = TCPServer.new('127.0.0.1', 0)
    p = s.local_address.ip_port
    s.close
    p
  end

  @@server_thread = nil
  @@server_transport = nil

  def self.startup
    handler   = StreamingTestServiceHandler.new
    processor = Thrift::StreamingTest::StreamingTest::Processor.new(handler)

    @@server_transport = Thrift::ServerSocket.new(PORT)
    server = Thrift::SimpleServer.new(
      processor,
      @@server_transport,
      Thrift::BufferedTransportFactory.new,
      Thrift::BinaryProtocolFactory.new
    )

    @@server_thread = Thread.new { server.serve }
    sleep 0.05
  end

  def self.shutdown
    @@server_transport.close rescue nil
    @@server_thread.kill rescue nil
  end

  def build_client
    transport = Thrift::BufferedTransport.new(Thrift::Socket.new('127.0.0.1', PORT))
    protocol  = Thrift::BinaryProtocol.new(transport)
    transport.open

    client = Thrift::StreamingTest::StreamingTest::Client.new(
      Thrift::BaseClient.new(protocol)
    )

    [client, transport]
  end

  def test_server_stream
    c, trans = build_client

    stream = c.testServerStream(3, 'hello')

    received = []
    3.times { received << stream.receive }

    assert_raises(EOFError) { stream.receive }
    stream.close

    assert_equal ['hello_0', 'hello_1', 'hello_2'], received
  ensure
    trans&.close rescue nil
  end

  def test_client_stream
    c, trans = build_client

    sink = c.testClientStream('label')

    5.times { |i| sink.send("msg#{i}") }
    sink.close
    # The handler returns nil immediately; we just verify the sink send+close
    # completes without error.
  ensure
    trans&.close rescue nil
  end

  def test_bidi_stream
    c, trans = build_client

    stream, sink = c.testBidi('bidi-label')

    pings = %w[hello world foo]
    pings.each do |ping|
      sink.send(ping)
      assert_equal ping.upcase, stream.receive
    end

    sink.close
    stream.close
  ensure
    trans&.close rescue nil
  end
end
