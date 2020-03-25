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

require 'rack'

module Thrift
  class RackApplication
    THRIFT_HEADER = 'application/x-thrift'.freeze

    def self.for(path, processor, protocol_factory)
      Rack::Builder.new do
        map path do
          run Thrift::RackApplication.new(processor, protocol_factory)
        end
      end
    end

    def initialize(processor, protocol_factory)
      @processor = processor
      @protocol_factory = protocol_factory
      @headers = { 'Content-Type' => THRIFT_HEADER }
    end

    def call(env)
      req = Rack::Request.new(env)

      if valid_thrift_request?(req)
        successful_request(req)
      else
        failed_request
      end
    end

    def successful_request(req)
      resp = Rack::Response.new([], 200, @headers)
      transport = IOStreamTransport.new req.body, resp
      protocol = @protocol_factory.get_protocol transport

      @processor.process protocol, protocol

      [resp.status, resp.headers, resp.body]
    end

    def failed_request
      [404, @headers, 'Not found']
    end

    def valid_thrift_request?(req)
      req.post? && req.env['CONTENT_TYPE'] == THRIFT_HEADER
    end
  end
end
