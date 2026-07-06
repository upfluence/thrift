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

$host          = 'localhost'
$port          = 9090
$domain_socket = nil
$protocol_type = 'binary'
$ssl           = false
$transport     = 'buffered'

ARGV.each do |a|
  if a == '--help'
    puts 'Allowed options:'
    puts "\t--host arg (=localhost)     Host to connect"
    puts "\t--port arg (=9090)          Port number to connect"
    puts "\t--domain-socket arg (=)     Unix domain socket path"
    puts "\t--protocol arg (=binary)    binary, compact, json"
    puts "\t--transport arg (=buffered) buffered, framed"
    puts "\t--ssl                       use SSL"
    exit
  elsif a.start_with?('--host')
    $host = a.split('=')[1]
  elsif a.start_with?('--port')
    $port = a.split('=')[1].to_i
  elsif a.start_with?('--domain-socket')
    $domain_socket = a.split('=')[1]
  elsif a.start_with?('--protocol')
    $protocol_type = a.split('=')[1]
  elsif a == '--ssl'
    $ssl = true
  elsif a.start_with?('--transport')
    $transport = a.split('=')[1]
  end
end

# Build the socket
socket =
  if !$domain_socket.to_s.strip.empty?
    Thrift::UNIXSocket.new($domain_socket)
  elsif $ssl
    keys_dir = File.join(File.dirname(File.dirname(Dir.pwd)), 'keys')
    ctx = OpenSSL::SSL::SSLContext.new
    ctx.ca_file = File.join(keys_dir, 'CA.pem')
    ctx.cert    = OpenSSL::X509::Certificate.new(File.open(File.join(keys_dir, 'client.crt')))
    ctx.key     = OpenSSL::PKey::RSA.new(File.open(File.join(keys_dir, 'client.key')))
    Thrift::SSLSocket.new($host, $port, nil, ctx)
  else
    Thrift::Socket.new($host, $port)
  end

transport =
  case $transport
  when 'framed' then Thrift::FramedTransport.new(socket)
  else               Thrift::BufferedTransport.new(socket)
  end

protocol =
  case $protocol_type
  when 'compact' then Thrift::CompactProtocol.new(transport)
  when 'json'    then Thrift::JsonProtocol.new(transport)
  else                Thrift::BinaryProtocol.new(transport)
  end

transport.open

client = Thrift::StreamingTest::StreamingTest::Client.new(
  Thrift::BaseClient.new(protocol)
)

# ── testClientStream ───────────────────────────────────────────────────────────
# Send 5 strings to the server's sink and close it.
sink = client.testClientStream('rb-client')

%w[alpha beta gamma delta epsilon].each { |m| sink.send(m) }
sink.close

$stdout.puts 'testClientStream: OK'

# ── testServerStream ───────────────────────────────────────────────────────────
# Request 3 messages with prefix "x"; verify "x_0", "x_1", "x_2".
COUNT  = 3
PREFIX = 'x'

stream   = client.testServerStream(COUNT, PREFIX)
received = []

COUNT.times do
  received << stream.receive
rescue EOFError
  break
end

begin
  stream.receive
  raise "testServerStream: expected EOFError after #{COUNT} messages"
rescue EOFError
  # expected
end

stream.close

if received.length != COUNT
  raise "testServerStream: expected #{COUNT} messages, got #{received.length}: #{received.inspect}"
end

received.each_with_index do |msg, i|
  want = "#{PREFIX}_#{i}"

  raise "testServerStream: received[#{i}] = #{msg.inspect}, want #{want.inspect}" unless msg == want
end

$stdout.puts "testServerStream: OK (received #{COUNT} messages)"

# ── testBidi ───────────────────────────────────────────────────────────────────
# Send 3 pings; expect each back uppercased.
stream, sink = client.testBidi('rb-client')

%w[hello world foo].each do |ping|
  sink.send(ping)
  reply = stream.receive
  want  = ping.upcase

  raise "testBidi: reply for #{ping.inspect} = #{reply.inspect}, want #{want.inspect}" unless reply == want
end

sink.close
stream.close

$stdout.puts 'testBidi: OK'

transport.close

$stdout.puts 'All streaming tests passed'
