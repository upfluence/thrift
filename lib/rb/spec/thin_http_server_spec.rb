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
require 'rack/test'
require 'thrift/server/thin_http_server'
require 'thrift/server/rack_application'

describe Thrift::ThinHTTPServer do

  let(:processor) { double('processor') }

  describe "#initialize" do

    context "when using the defaults" do

      it "binds to port 80, with host 0.0.0.0, a path of '/'" do
        expect(Thin::Server).to receive(:new).with('0.0.0.0', 80, an_instance_of(Rack::Builder))
        Thrift::ThinHTTPServer.new(processor)
      end

      it "uses the BinaryProtocolFactory" do
        expect(Thrift::BinaryProtocolFactory).to receive(:new)
        Thrift::ThinHTTPServer.new(processor)
      end

    end

    context "when using the options" do

      it 'accepts :ip, :port, :path' do
        ip = "192.168.0.1"
        port = 3000
        path = "/thin"
        expect(Thin::Server).to receive(:new).with(ip, port, an_instance_of(Rack::Builder))
        Thrift::ThinHTTPServer.new(processor,
                           :ip => ip,
                           :port => port,
                           :path => path)
      end
    end

  end

  describe "#serve" do

    it 'starts the Thin server' do
      underlying_thin_server = double('thin server', :start => true)
      allow(Thin::Server).to receive(:new).and_return(underlying_thin_server)

      thin_thrift_server = Thrift::ThinHTTPServer.new(processor)

      expect(underlying_thin_server).to receive(:start)
      thin_thrift_server.serve
    end
  end

end
