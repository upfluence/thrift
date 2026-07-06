#!/usr/bin/env ruby
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

$:.unshift File.join(File.dirname(__FILE__), '..', '..', '..', 'lib', 'rb', 'lib')
$:.push File.dirname(__FILE__) + '/..'
$:.push File.join(File.dirname(__FILE__), '..', 'gen-rb')

require 'thrift'
require 'streaming_test'

# StreamingHandler implements all three StreamingTest service methods.
class StreamingHandler
  # testClientStream: drain the sink in a background thread; return immediately.
  def testClientStream(label, sink)
    Thread.new do
      loop do
        sink.receive
      rescue EOFError, StandardError
        break
      end
    end

    nil
  end

  # testServerStream: push count messages of the form "{prefix}_{i}" in a
  # background thread so the initial REPLY can be sent before streaming starts.
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

domain_socket = nil
port          = 9090
protocol      = 'binary'
ssl           = false
transport     = 'buffered'

ARGV.each do |a|
  if a == '--help'
    puts 'Allowed options:'
    puts "\t--domain-socket arg (=)    Unix domain socket path"
    puts "\t--port arg (=9090)         Port number to listen"
    puts "\t--protocol arg (=binary)   binary, compact, json"
    puts "\t--transport arg (=buffered) buffered, framed"
    puts "\t--ssl                      use SSL"
    exit
  elsif a.start_with?('--domain-socket')
    domain_socket = a.split('=')[1]
  elsif a.start_with?('--protocol')
    protocol = a.split('=')[1]
  elsif a == '--ssl'
    ssl = true
  elsif a.start_with?('--transport')
    transport = a.split('=')[1]
  elsif a.start_with?('--port')
    port = a.split('=')[1].to_i
  end
end

protocol_factory =
  case protocol
  when 'compact' then Thrift::CompactProtocolFactory.new
  when 'json'    then Thrift::JsonProtocolFactory.new
  else                Thrift::BinaryProtocolFactory.new
  end

transport_factory =
  case transport
  when 'framed' then Thrift::FramedTransportFactory.new
  else               Thrift::BufferedTransportFactory.new
  end

handler   = StreamingHandler.new
processor = Thrift::StreamingTest::StreamingTest::Processor.new(handler)

server_transport =
  if !domain_socket.to_s.strip.empty?
    Thrift::UNIXServerSocket.new(domain_socket)
  elsif ssl
    keys_dir = File.join(File.dirname(File.dirname(Dir.pwd)), 'keys')
    ctx = OpenSSL::SSL::SSLContext.new
    ctx.ca_file = File.join(keys_dir, 'CA.pem')
    ctx.cert    = OpenSSL::X509::Certificate.new(File.open(File.join(keys_dir, 'server.crt')))
    ctx.key     = OpenSSL::PKey::RSA.new(File.open(File.join(keys_dir, 'server.key')))
    Thrift::SSLServerSocket.new(nil, port, ctx)
  else
    Thrift::ServerSocket.new(port)
  end

server = Thrift::SimpleServer.new(processor, server_transport, transport_factory, protocol_factory)

$stdout.puts "Starting StreamingServer on port #{port} (transport=#{transport} protocol=#{protocol})"
$stdout.flush

server.serve
